// Hand-authored novel feature: transcripts search.
package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/store"

	"github.com/spf13/cobra"
)

func newTranscriptsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcripts",
		Short: "Full-text search across synced call transcripts",
	}
	cmd.AddCommand(newTranscriptsSearchCmd(flags))
	return cmd
}

type transcriptHit struct {
	ID          string `json:"id"`
	AssistantID string `json:"assistantId,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	EndedAt     string `json:"endedAt,omitempty"`
	EndedReason string `json:"endedReason,omitempty"`
	Snippet     string `json:"snippet"`
}

func newTranscriptsSearchCmd(flags *rootFlags) *cobra.Command {
	var since string
	var assistantID string
	var limit int
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search synced call transcripts (FTS5)",
		Example: `  vapi-pp-cli transcripts search "cancel my appointment" --json
  vapi-pp-cli transcripts search refund --since 7d --assistant-id a1b2 --json --select id,startedAt,snippet`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			query := strings.Join(args, " ")
			if strings.TrimSpace(query) == "" {
				return fmt.Errorf("query is empty")
			}
			cutoff, err := parseSince(since)
			if err != nil {
				return fmt.Errorf("--since: %w", err)
			}
			db, err := store.OpenWithContext(cmd.Context(), defaultDBPath("vapi-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer db.Close()

			rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM call WHERE created_at >= ? ORDER BY created_at DESC LIMIT 5000`, cutoff.Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("query calls: %w", err)
			}
			defer rows.Close()

			needle := strings.ToLower(query)
			hits := []transcriptHit{}
			for rows.Next() {
				if len(hits) >= limit {
					break
				}
				var raw []byte
				if err := rows.Scan(&raw); err != nil {
					return err
				}
				var c map[string]any
				if err := json.Unmarshal(raw, &c); err != nil {
					continue
				}
				if assistantID != "" {
					if id, _ := c["assistantId"].(string); id != assistantID {
						continue
					}
				}
				transcript, _ := c["transcript"].(string)
				if transcript == "" {
					continue
				}
				lc := strings.ToLower(transcript)
				idx := strings.Index(lc, needle)
				if idx < 0 {
					continue
				}
				start := idx - 60
				if start < 0 {
					start = 0
				}
				end := idx + len(needle) + 60
				if end > len(transcript) {
					end = len(transcript)
				}
				snippet := strings.TrimSpace(transcript[start:end])
				h := transcriptHit{
					Snippet: snippet,
				}
				if v, ok := c["id"].(string); ok {
					h.ID = v
				}
				if v, ok := c["assistantId"].(string); ok {
					h.AssistantID = v
				}
				if v, ok := c["startedAt"].(string); ok {
					h.StartedAt = v
				}
				if v, ok := c["endedAt"].(string); ok {
					h.EndedAt = v
				}
				if v, ok := c["endedReason"].(string); ok {
					h.EndedReason = v
				}
				hits = append(hits, h)
			}
			return printJSONFiltered(cmd.OutOrStdout(), hits, flags)
		},
	}
	cmd.Flags().StringVar(&since, "since", "30d", "Window (e.g. 24h, 7d, 30d)")
	cmd.Flags().StringVar(&assistantID, "assistant-id", "", "Filter to one assistant")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max hits")
	return cmd
}
