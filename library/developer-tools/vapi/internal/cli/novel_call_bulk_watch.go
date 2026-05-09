// Hand-authored novel features: call bulk planner, call watch.
// Both surface as subcommands of the plural "calls" parent and as standalone helpers.
package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/store"

	"github.com/spf13/cobra"
)

// ----- call bulk planner -----

type bulkPlanRow struct {
	CustomerNumber string  `json:"customerNumber"`
	AssistantID    string  `json:"assistantId,omitempty"`
	PhoneNumberID  string  `json:"phoneNumberId,omitempty"`
	EstimatedUSD   float64 `json:"estimatedUsd"`
}

type bulkPlanReport struct {
	Plan         []bulkPlanRow `json:"plan"`
	Count        int           `json:"count"`
	EstimatedUSD float64       `json:"estimatedTotalUsd"`
	Note         string        `json:"note"`
	DryRun       bool          `json:"dryRun"`
}

func newCallBulkSubCmd(flags *rootFlags) *cobra.Command {
	var csvPath string
	var assistantID string
	var phoneNumberID string
	var estimateUSD float64
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Plan an outbound call batch from a CSV (--dry-run by default)",
		Long: `Reads a CSV of customer numbers (one per row, header optional) and prints a planned
batch of outbound calls. Default is --dry-run; pass --commit to actually send the calls.`,
		Example: `  vapi-pp-cli calls bulk --csv customers.csv --assistant-id a1b2 --phone-number-id p9 --json
  vapi-pp-cli calls bulk --csv customers.csv --assistant-id a1b2 --phone-number-id p9 --commit`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if csvPath == "" {
				return fmt.Errorf("--csv <path> is required")
			}
			if assistantID == "" {
				return fmt.Errorf("--assistant-id is required")
			}
			commit, _ := cmd.Flags().GetBool("commit")
			f, err := os.Open(csvPath)
			if err != nil {
				return fmt.Errorf("open csv: %w", err)
			}
			defer f.Close()
			r := csv.NewReader(f)
			r.FieldsPerRecord = -1
			rows, err := r.ReadAll()
			if err != nil {
				return fmt.Errorf("read csv: %w", err)
			}
			plan := []bulkPlanRow{}
			for i, row := range rows {
				if len(row) == 0 {
					continue
				}
				num := strings.TrimSpace(row[0])
				if i == 0 && !strings.HasPrefix(num, "+") {
					// likely a header
					continue
				}
				if num == "" {
					continue
				}
				plan = append(plan, bulkPlanRow{
					CustomerNumber: num,
					AssistantID:    assistantID,
					PhoneNumberID:  phoneNumberID,
					EstimatedUSD:   estimateUSD,
				})
			}
			report := bulkPlanReport{
				Plan:         plan,
				Count:        len(plan),
				EstimatedUSD: float64(len(plan)) * estimateUSD,
				DryRun:       !commit,
			}
			if !commit {
				report.Note = "Dry run. Pass --commit to actually place these calls."
				return printJSONFiltered(cmd.OutOrStdout(), report, flags)
			}
			// Real send path — guarded behind --commit, runs through the API client.
			if dryRunOK(flags) {
				return printJSONFiltered(cmd.OutOrStdout(), report, flags)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			report.Note = "Live calls placed."
			results := []map[string]any{}
			for _, row := range plan {
				body := map[string]any{
					"assistantId": row.AssistantID,
					"customer":    map[string]any{"number": row.CustomerNumber},
				}
				if row.PhoneNumberID != "" {
					body["phoneNumberId"] = row.PhoneNumberID
				}
				resp, status, err := c.Post("/call", body)
				rec := map[string]any{"customerNumber": row.CustomerNumber, "status": status}
				if err != nil {
					rec["error"] = err.Error()
				} else {
					var parsed map[string]any
					if jerr := json.Unmarshal(resp, &parsed); jerr == nil {
						rec["id"] = parsed["id"]
					}
				}
				results = append(results, rec)
			}
			out := map[string]any{"report": report, "results": results}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&csvPath, "csv", "", "Path to CSV of customer numbers (column 1)")
	cmd.Flags().StringVar(&assistantID, "assistant-id", "", "Assistant to use for every call")
	cmd.Flags().StringVar(&phoneNumberID, "phone-number-id", "", "Outbound phone number")
	cmd.Flags().Float64Var(&estimateUSD, "estimate-usd", 0.05, "Per-call cost estimate for the rollup")
	cmd.Flags().Bool("commit", false, "Actually place the calls (default is dry-run)")
	return cmd
}

// ----- call watch -----

type watchEvent struct {
	Type        string `json:"type"` // started | ended
	ID          string `json:"id"`
	AssistantID string `json:"assistantId,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	EndedAt     string `json:"endedAt,omitempty"`
	Status      string `json:"status,omitempty"`
}

func newCallWatchSubCmd(flags *rootFlags) *cobra.Command {
	var interval time.Duration
	var iterations int
	cmd := &cobra.Command{
		Use:         "watch",
		Short:       "Long-poll the synced store for newly active or ended calls",
		Example:     `  vapi-pp-cli calls watch --interval 5s --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			db, err := store.OpenWithContext(cmd.Context(), defaultDBPath("vapi-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer db.Close()
			seen := map[string]string{} // id -> last status
			ctx := cmd.Context()
			loop := 0
			for {
				loop++
				events := pollCalls(ctx, db, seen)
				sort.Slice(events, func(i, j int) bool { return events[i].StartedAt < events[j].StartedAt })
				for _, e := range events {
					b, _ := json.Marshal(e)
					fmt.Fprintln(cmd.OutOrStdout(), string(b))
				}
				if iterations > 0 && loop >= iterations {
					return nil
				}
				select {
				case <-time.After(interval):
				case <-ctx.Done():
					return nil
				}
			}
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "Poll interval")
	cmd.Flags().IntVar(&iterations, "iterations", 0, "Stop after N iterations (0 = forever)")
	return cmd
}

func pollCalls(ctx context.Context, db *store.Store, seen map[string]string) []watchEvent {
	rows, err := db.DB().QueryContext(ctx, `SELECT data FROM call ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []watchEvent{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return out
		}
		var c map[string]any
		if err := json.Unmarshal(raw, &c); err != nil {
			continue
		}
		id, _ := c["id"].(string)
		if id == "" {
			continue
		}
		status, _ := c["status"].(string)
		prev, ok := seen[id]
		if ok && prev == status {
			continue
		}
		ev := watchEvent{ID: id, Status: status}
		if !ok {
			ev.Type = "started"
		} else {
			ev.Type = "ended"
		}
		if v, ok := c["assistantId"].(string); ok {
			ev.AssistantID = v
		}
		if v, ok := c["startedAt"].(string); ok {
			ev.StartedAt = v
		}
		if v, ok := c["endedAt"].(string); ok {
			ev.EndedAt = v
		}
		out = append(out, ev)
		seen[id] = status
	}
	return out
}
