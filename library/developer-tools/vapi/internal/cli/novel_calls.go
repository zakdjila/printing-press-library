// Hand-authored novel features under the "calls" plural alias group.
// Distinct from the generated singular "call" CRUD parent.
package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/store"

	"github.com/spf13/cobra"
)

func newCallsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calls",
		Short: "Local roll-ups across synced calls (plural; singular 'call' is upstream CRUD)",
	}
	cmd.AddCommand(newCallsWhyCmd(flags))
	cmd.AddCommand(newCallWatchSubCmd(flags))
	cmd.AddCommand(newCallBulkSubCmd(flags))
	return cmd
}

type endedReasonRow struct {
	Reason string `json:"endedReason"`
	Count  int    `json:"count"`
}

func newCallsWhyCmd(flags *rootFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:         "why",
		Short:       "Histogram of ended reasons across recent calls",
		Example:     `  vapi-pp-cli calls why --since 24h --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
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

			rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM call WHERE created_at >= ?`, cutoff.Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("query: %w", err)
			}
			defer rows.Close()
			counts := map[string]int{}
			for rows.Next() {
				var raw []byte
				if err := rows.Scan(&raw); err != nil {
					return err
				}
				var c map[string]any
				if err := json.Unmarshal(raw, &c); err != nil {
					continue
				}
				reason, _ := c["endedReason"].(string)
				if reason == "" {
					reason = "(in-progress)"
				}
				counts[reason]++
			}
			out := make([]endedReasonRow, 0, len(counts))
			for k, v := range counts {
				out = append(out, endedReasonRow{Reason: k, Count: v})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&since, "since", "24h", "Window (24h, 7d, 30d)")
	return cmd
}
