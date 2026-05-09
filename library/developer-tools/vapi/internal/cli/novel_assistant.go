// Hand-authored novel features that surface under a parallel "assistants"
// (plural) parent — the singular "assistant" parent is upstream CRUD.
package cli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/store"

	"github.com/spf13/cobra"
)

func newAssistantsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assistants",
		Short: "Cross-assistant analytics (plural; singular 'assistant' is upstream CRUD)",
	}
	cmd.AddCommand(newAssistantCompareCmd(flags))
	cmd.AddCommand(newStaleAssistantsSubCmd(flags, "stale"))
	cmd.AddCommand(newDriftAssistantsSubCmd(flags, "drift"))
	return cmd
}

type assistantStats struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name,omitempty"`
	Calls              int            `json:"calls"`
	TotalUSD           float64        `json:"totalUsd"`
	AvgDurationSeconds float64        `json:"avgDurationSeconds"`
	EndedReasons       map[string]int `json:"endedReasons"`
}

func newAssistantCompareCmd(flags *rootFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:         "compare <id-a> <id-b>",
		Short:       "Side-by-side cost, duration, and ended-reason for two assistants",
		Example:     `  vapi-pp-cli assistants compare a1b2 c3d4 --since 7d --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return cmd.Help()
			}
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

			results := []assistantStats{}
			for _, id := range []string{args[0], args[1]} {
				s, err := computeAssistantStats(cmd, db, id, cutoff)
				if err != nil {
					return err
				}
				results = append(results, s)
			}
			return printJSONFiltered(cmd.OutOrStdout(), results, flags)
		},
	}
	cmd.Flags().StringVar(&since, "since", "7d", "Window")
	return cmd
}

func computeAssistantStats(cmd *cobra.Command, db *store.Store, id string, cutoff time.Time) (assistantStats, error) {
	s := assistantStats{ID: id, EndedReasons: map[string]int{}}
	// Pull name
	if raw, err := db.Get("assistant", id); err == nil && len(raw) > 0 {
		var a map[string]any
		_ = json.Unmarshal(raw, &a)
		if n, ok := a["name"].(string); ok {
			s.Name = n
		}
	}
	rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM call WHERE created_at >= ?`, cutoff.Format(time.RFC3339))
	if err != nil {
		return s, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	totalDur := 0.0
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return s, err
		}
		var c map[string]any
		if err := json.Unmarshal(raw, &c); err != nil {
			continue
		}
		if aid, _ := c["assistantId"].(string); aid != id {
			continue
		}
		s.Calls++
		if cost, ok := c["cost"].(float64); ok {
			s.TotalUSD += cost
		}
		startedAt, _ := c["startedAt"].(string)
		endedAt, _ := c["endedAt"].(string)
		if startedAt != "" && endedAt != "" {
			st, e1 := time.Parse(time.RFC3339, startedAt)
			et, e2 := time.Parse(time.RFC3339, endedAt)
			if e1 == nil && e2 == nil {
				totalDur += et.Sub(st).Seconds()
			}
		}
		reason, _ := c["endedReason"].(string)
		if reason == "" {
			reason = "(in-progress)"
		}
		s.EndedReasons[reason]++
	}
	if s.Calls > 0 {
		s.AvgDurationSeconds = totalDur / float64(s.Calls)
	}
	return s, nil
}

// ----- stale assistants -----

type staleRow struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	LastCall string `json:"lastCall,omitempty"`
}

func newStaleAssistantsSubCmd(flags *rootFlags, use string) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   use,
		Short: "Assistants with no calls in the last N days",
		Example: `  vapi-pp-cli assistants stale --days 30 --json
  vapi-pp-cli stale assistants --days 30 --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
			db, err := store.OpenWithContext(cmd.Context(), defaultDBPath("vapi-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer db.Close()

			// Most-recent call per assistant
			lastByAssistant := map[string]string{}
			rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM call ORDER BY created_at DESC`)
			if err != nil {
				return fmt.Errorf("query calls: %w", err)
			}
			func() {
				defer rows.Close()
				for rows.Next() {
					var raw []byte
					if err := rows.Scan(&raw); err != nil {
						return
					}
					var c map[string]any
					if err := json.Unmarshal(raw, &c); err != nil {
						continue
					}
					aid, _ := c["assistantId"].(string)
					if aid == "" {
						continue
					}
					ts, _ := c["startedAt"].(string)
					if ts == "" {
						ts, _ = c["createdAt"].(string)
					}
					if existing, ok := lastByAssistant[aid]; !ok || ts > existing {
						lastByAssistant[aid] = ts
					}
				}
			}()

			aRows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM assistant`)
			if err != nil {
				return fmt.Errorf("query assistants: %w", err)
			}
			defer aRows.Close()
			out := []staleRow{}
			for aRows.Next() {
				var raw []byte
				if err := aRows.Scan(&raw); err != nil {
					return err
				}
				var a map[string]any
				if err := json.Unmarshal(raw, &a); err != nil {
					continue
				}
				id, _ := a["id"].(string)
				if id == "" {
					continue
				}
				last, hasCall := lastByAssistant[id]
				stale := !hasCall
				if hasCall {
					if t, err := time.Parse(time.RFC3339, last); err == nil {
						stale = t.Before(cutoff)
					}
				}
				if !stale {
					continue
				}
				name, _ := a["name"].(string)
				out = append(out, staleRow{ID: id, Name: name, LastCall: last})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].LastCall < out[j].LastCall })
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Days of inactivity")
	return cmd
}

// ----- drift assistants -----

type driftDiff struct {
	ID            string         `json:"id"`
	Name          string         `json:"name,omitempty"`
	DriftedFields []string       `json:"driftedFields"`
	Detail        map[string]any `json:"detail,omitempty"`
}

func newDriftAssistantsSubCmd(flags *rootFlags, use string) *cobra.Command {
	var baseline string
	var detail bool
	cmd := &cobra.Command{
		Use:   use,
		Short: "Diff every assistant against a baseline assistant",
		Example: `  vapi-pp-cli assistants drift --baseline a1b2 --json
  vapi-pp-cli drift assistants --baseline a1b2 --detail --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if baseline == "" {
				return fmt.Errorf("--baseline <assistant-id> is required")
			}
			if dryRunOK(flags) {
				return nil
			}
			db, err := store.OpenWithContext(cmd.Context(), defaultDBPath("vapi-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer db.Close()

			baseRaw, err := db.Get("assistant", baseline)
			if err != nil || len(baseRaw) == 0 {
				return fmt.Errorf("baseline %s not found in local store; run 'sync assistants' first", baseline)
			}
			var base map[string]any
			if err := json.Unmarshal(baseRaw, &base); err != nil {
				return err
			}
			fields := []string{"model", "voice", "transcriber", "firstMessage", "endCallMessage", "voicemailMessage"}

			rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM assistant`)
			if err != nil {
				return err
			}
			defer rows.Close()
			out := []driftDiff{}
			for rows.Next() {
				var raw []byte
				if err := rows.Scan(&raw); err != nil {
					return err
				}
				var a map[string]any
				if err := json.Unmarshal(raw, &a); err != nil {
					continue
				}
				id, _ := a["id"].(string)
				if id == "" || id == baseline {
					continue
				}
				drifted := []string{}
				det := map[string]any{}
				for _, f := range fields {
					if !reflect.DeepEqual(base[f], a[f]) {
						drifted = append(drifted, f)
						if detail {
							det[f] = map[string]any{"baseline": base[f], "this": a[f]}
						}
					}
				}
				if len(drifted) == 0 {
					continue
				}
				name, _ := a["name"].(string)
				d := driftDiff{ID: id, Name: name, DriftedFields: drifted}
				if detail {
					d.Detail = det
				}
				out = append(out, d)
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&baseline, "baseline", "", "Baseline assistant ID")
	cmd.Flags().BoolVar(&detail, "detail", false, "Include per-field baseline vs this object diff")
	return cmd
}

// Top-level "stale" / "drift" parents that mirror the research.json command paths.
func newStaleCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stale",
		Short: "Find unused resources",
	}
	cmd.AddCommand(newStaleAssistantsSubCmd(flags, "assistants"))
	return cmd
}

func newDriftCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Detect config drift across resource fleets",
	}
	cmd.AddCommand(newDriftAssistantsSubCmd(flags, "assistants"))
	return cmd
}
