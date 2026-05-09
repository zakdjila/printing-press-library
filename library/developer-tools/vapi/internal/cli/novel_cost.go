// Hand-authored novel feature: cost summary.
package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/store"

	"github.com/spf13/cobra"
)

func newCostCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Aggregate call cost across the synced store",
		Long:  "Local roll-ups of call cost. Run 'sync calls' first.",
	}
	cmd.AddCommand(newCostSummaryCmd(flags))
	return cmd
}

type costRow struct {
	Key      string  `json:"key"`
	Calls    int     `json:"calls"`
	TotalUSD float64 `json:"totalUsd"`
	AvgUSD   float64 `json:"avgUsd"`
}

func newCostSummaryCmd(flags *rootFlags) *cobra.Command {
	var since string
	var by string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Aggregate call cost by assistant, day, or phone number",
		Example: `  vapi-pp-cli cost summary --since 7d --by assistant --json
  vapi-pp-cli cost summary --since 30d --by day --json
  vapi-pp-cli cost summary --by phone-number --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			cutoff, err := parseSince(since)
			if err != nil {
				return fmt.Errorf("--since: %w", err)
			}
			groupExpr, ok := costGroupExpr(by)
			if !ok {
				return fmt.Errorf("--by: must be one of assistant, day, phone-number; got %q", by)
			}
			db, err := store.OpenWithContext(cmd.Context(), defaultDBPath("vapi-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer db.Close()

			rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM call WHERE created_at >= ? ORDER BY created_at DESC`, cutoff.Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("query calls: %w", err)
			}
			defer rows.Close()

			agg := map[string]*costRow{}
			for rows.Next() {
				var raw []byte
				if err := rows.Scan(&raw); err != nil {
					return err
				}
				var c map[string]any
				if err := json.Unmarshal(raw, &c); err != nil {
					continue
				}
				key := costGroupKey(c, groupExpr)
				if key == "" {
					key = "(unknown)"
				}
				cost, _ := c["cost"].(float64)
				if r, ok := agg[key]; ok {
					r.Calls++
					r.TotalUSD += cost
				} else {
					agg[key] = &costRow{Key: key, Calls: 1, TotalUSD: cost}
				}
			}
			out := make([]costRow, 0, len(agg))
			for _, r := range agg {
				if r.Calls > 0 {
					r.AvgUSD = r.TotalUSD / float64(r.Calls)
				}
				out = append(out, *r)
			}
			sort.Slice(out, func(i, j int) bool { return out[i].TotalUSD > out[j].TotalUSD })
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&since, "since", "30d", "Window (e.g. 24h, 7d, 30d)")
	cmd.Flags().StringVar(&by, "by", "assistant", "Group by: assistant, day, phone-number")
	return cmd
}

func costGroupExpr(by string) (string, bool) {
	switch strings.ToLower(by) {
	case "assistant", "assistantid":
		return "assistantId", true
	case "day", "date":
		return "day", true
	case "phone", "phonenumber", "phone-number", "phonenumberid":
		return "phoneNumberId", true
	}
	return "", false
}

func costGroupKey(c map[string]any, expr string) string {
	if expr == "day" {
		s, _ := c["startedAt"].(string)
		if s == "" {
			s, _ = c["createdAt"].(string)
		}
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	}
	if v, ok := c[expr].(string); ok {
		return v
	}
	return ""
}

// parseSince accepts strings like "24h", "7d", "30d", or RFC3339; returns the cutoff time.
func parseSince(s string) (time.Time, error) {
	now := time.Now().UTC()
	if s == "" {
		return now.Add(-30 * 24 * time.Hour), nil
	}
	// suffix d
	if strings.HasSuffix(s, "d") {
		var n int
		_, err := fmt.Sscanf(s, "%dd", &n)
		if err != nil {
			return now, err
		}
		return now.Add(-time.Duration(n) * 24 * time.Hour), nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return now.Add(-d), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return now, fmt.Errorf("could not parse %q (use 24h, 7d, 30d, or RFC3339)", s)
}
