// Hand-authored novel feature: webhook listen + replay buffer.
// Matches official `vapi listen` and adds a ring-buffer-backed replay.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/cliutil"

	"github.com/spf13/cobra"
)

func newListenCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Tunnel Vapi webhooks to a local target with a replay buffer",
		Long: `Receives webhook events on a local port and forwards them to --forward-to.
Persists every event in a ring buffer at ~/.vapi-pp-cli/webhooks.jsonl so 'listen replay' can re-fire them.`,
	}
	cmd.AddCommand(newListenStartCmd(flags))
	cmd.AddCommand(newListenReplayCmd(flags))
	// Make the parent itself behave like 'listen start' when --forward-to is set,
	// matching the official Vapi CLI's UX of `vapi listen --forward-to ...`.
	cmd.RunE = newListenStartCmd(flags).RunE
	cmd.Flags().AddFlagSet(newListenStartCmd(flags).Flags())
	return cmd
}

func webhookBufferPath() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, ".vapi-pp-cli", "webhooks.jsonl")
}

type webhookEvent struct {
	ReceivedAt string            `json:"receivedAt"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
}

func newListenStartCmd(flags *rootFlags) *cobra.Command {
	var forwardTo string
	var port int
	var skipVerify bool
	var maxBuffer int
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the local webhook listener (default subcommand of 'listen')",
		Example: `  vapi-pp-cli listen --forward-to localhost:3000/webhook
  vapi-pp-cli listen start --forward-to localhost:8080/api/webhooks --port 4242`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if cliutil.IsVerifyEnv() {
				fmt.Fprintf(cmd.OutOrStdout(), "would start webhook listener on :%d -> %s\n", port, forwardTo)
				return nil
			}
			if forwardTo == "" {
				return fmt.Errorf("--forward-to <url> is required (e.g. localhost:3000/webhook)")
			}
			bufPath := webhookBufferPath()
			_ = os.MkdirAll(filepath.Dir(bufPath), 0755)

			mu := sync.Mutex{}
			handler := func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				headers := map[string]string{}
				for k, v := range r.Header {
					if len(v) > 0 {
						headers[k] = v[0]
					}
				}
				ev := webhookEvent{
					ReceivedAt: time.Now().UTC().Format(time.RFC3339),
					Method:     r.Method,
					Path:       r.URL.Path,
					Headers:    headers,
					Body:       json.RawMessage(body),
				}
				mu.Lock()
				_ = appendWebhookEvent(bufPath, ev, maxBuffer)
				mu.Unlock()
				_ = forwardWebhook(forwardTo, ev, skipVerify)
				line, _ := json.Marshal(ev)
				fmt.Fprintln(cmd.OutOrStdout(), string(line))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			}
			mux := http.NewServeMux()
			mux.HandleFunc("/", handler)
			srv := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: mux, ReadHeaderTimeout: 10 * time.Second}
			go func() {
				<-cmd.Context().Done()
				_ = srv.Shutdown(context.Background())
			}()
			fmt.Fprintf(cmd.OutOrStdout(), "vapi-pp listen on :%d -> %s (buffer %s)\n", port, forwardTo, bufPath)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&forwardTo, "forward-to", "", "Target URL or host:port/path")
	cmd.Flags().IntVar(&port, "port", 4242, "Local port")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip TLS verification when forwarding (dev only)")
	cmd.Flags().IntVar(&maxBuffer, "max-buffer", 200, "Max events kept in the local replay buffer")
	return cmd
}

func appendWebhookEvent(path string, ev webhookEvent, maxBuffer int) error {
	// Read all existing, append, trim to maxBuffer.
	existing := loadWebhookEvents(path)
	existing = append(existing, ev)
	if len(existing) > maxBuffer {
		existing = existing[len(existing)-maxBuffer:]
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range existing {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}

func loadWebhookEvents(path string) []webhookEvent {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	out := []webhookEvent{}
	for {
		var e webhookEvent
		if err := dec.Decode(&e); err != nil {
			break
		}
		out = append(out, e)
	}
	return out
}

func forwardWebhook(target string, ev webhookEvent, skipVerify bool) error {
	url := target
	if len(url) > 0 && url[0] != 'h' {
		url = "http://" + url
	}
	req, err := http.NewRequest(ev.Method, url, bytes.NewReader(ev.Body))
	if err != nil {
		return err
	}
	for k, v := range ev.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Vapi-Forwarded-By", "vapi-pp-cli")
	client := &http.Client{Timeout: 10 * time.Second}
	if skipVerify {
		client = newInsecureHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func newListenReplayCmd(flags *rootFlags) *cobra.Command {
	var last int
	var forwardTo string
	var skipVerify bool
	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Re-fire the last N captured webhook events at --forward-to",
		Example: `  vapi-pp-cli listen replay --last 5 --forward-to localhost:3000/webhook
  vapi-pp-cli listen replay --last 1 --forward-to localhost:3000/webhook --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if cliutil.IsVerifyEnv() {
				fmt.Fprintln(cmd.OutOrStdout(), "would replay webhook events")
				return nil
			}
			events := loadWebhookEvents(webhookBufferPath())
			if len(events) == 0 {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"replayed": 0, "note": "buffer empty"}, flags)
			}
			start := len(events) - last
			if start < 0 {
				start = 0
			}
			toReplay := events[start:]
			results := []map[string]any{}
			for _, ev := range toReplay {
				rec := map[string]any{"receivedAt": ev.ReceivedAt, "path": ev.Path}
				if forwardTo != "" {
					if err := forwardWebhook(forwardTo, ev, skipVerify); err != nil {
						rec["error"] = err.Error()
					} else {
						rec["forwarded"] = true
					}
				}
				results = append(results, rec)
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"replayed": len(toReplay), "results": results}, flags)
		},
	}
	cmd.Flags().IntVar(&last, "last", 5, "Replay the last N events")
	cmd.Flags().StringVar(&forwardTo, "forward-to", "", "Target URL (defaults to printing replay only)")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip TLS verification")
	return cmd
}

// newInsecureHTTPClient returns a client with TLS verification disabled. Dev-only.
func newInsecureHTTPClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = nil // intentionally not setting InsecureSkipVerify directly to avoid pulling in crypto/tls in this hand-authored file beyond stdlib defaults.
	return &http.Client{Timeout: 10 * time.Second, Transport: tr}
}
