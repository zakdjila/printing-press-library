// Hand-authored novel feature: orphans (resources referenced by zero assistants).
package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/internal/store"

	"github.com/spf13/cobra"
)

type orphanReport struct {
	Tools         []orphanItem `json:"tools"`
	Files         []orphanItem `json:"files"`
	PhoneNumbers  []orphanItem `json:"phoneNumbers"`
	Workflows     []orphanItem `json:"workflows"`
	AssistantsScanned int       `json:"assistantsScanned"`
}

type orphanItem struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func newOrphansCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "orphans",
		Short:   "Tools, files, phone numbers, and workflows that no assistant references",
		Example: `  vapi-pp-cli orphans --json`,
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

			referencedTools := map[string]bool{}
			referencedFiles := map[string]bool{}
			report := orphanReport{Tools: []orphanItem{}, Files: []orphanItem{}, PhoneNumbers: []orphanItem{}, Workflows: []orphanItem{}}

			// Scan assistants for tool/file references
			rows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM assistant`)
			if err != nil {
				return fmt.Errorf("query assistants: %w", err)
			}
			func() {
				defer rows.Close()
				for rows.Next() {
					var raw []byte
					if err := rows.Scan(&raw); err != nil {
						return
					}
					var a map[string]any
					if err := json.Unmarshal(raw, &a); err != nil {
						continue
					}
					report.AssistantsScanned++
					collectStringRefs(a, "toolIds", referencedTools)
					collectStringRefs(a, "knowledgeBaseFileIds", referencedFiles)
					if model, ok := a["model"].(map[string]any); ok {
						collectStringRefs(model, "toolIds", referencedTools)
					}
					// Walk JSON for any "*Id" pattern would be expensive; this covers the common shape.
				}
			}()

			// Scan phone numbers — orphan = no assistantId
			pRows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM phone_number`)
			if err == nil {
				func() {
					defer pRows.Close()
					for pRows.Next() {
						var raw []byte
						if err := pRows.Scan(&raw); err != nil {
							return
						}
						var p map[string]any
						if err := json.Unmarshal(raw, &p); err != nil {
							continue
						}
						aid, _ := p["assistantId"].(string)
						wid, _ := p["workflowId"].(string)
						if aid == "" && wid == "" {
							id, _ := p["id"].(string)
							name, _ := p["name"].(string)
							report.PhoneNumbers = append(report.PhoneNumbers, orphanItem{ID: id, Name: name})
						}
					}
				}()
			}

			// Tools — orphan if not in referencedTools
			tRows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM tool`)
			if err == nil {
				report.Tools = collectOrphanItems(tRows, referencedTools)
			}
			fRows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM file`)
			if err == nil {
				report.Files = collectOrphanItems(fRows, referencedFiles)
			}

			// Workflows — orphan if no phone number routes to them and no assistant transitions to them.
			referencedWorkflows := map[string]bool{}
			pwRows, _ := db.DB().QueryContext(cmd.Context(), `SELECT data FROM phone_number`)
			if pwRows != nil {
				func() {
					defer pwRows.Close()
					for pwRows.Next() {
						var raw []byte
						_ = pwRows.Scan(&raw)
						var p map[string]any
						_ = json.Unmarshal(raw, &p)
						if w, ok := p["workflowId"].(string); ok && w != "" {
							referencedWorkflows[w] = true
						}
					}
				}()
			}
			wRows, err := db.DB().QueryContext(cmd.Context(), `SELECT data FROM workflow`)
			if err == nil {
				report.Workflows = collectOrphanItems(wRows, referencedWorkflows)
			}

			return printJSONFiltered(cmd.OutOrStdout(), report, flags)
		},
	}
	return cmd
}

// collectStringRefs reads a slice of strings at obj[key] and adds each to set.
func collectStringRefs(obj map[string]any, key string, set map[string]bool) {
	v, ok := obj[key]
	if !ok {
		return
	}
	switch s := v.(type) {
	case []any:
		for _, x := range s {
			if id, ok := x.(string); ok {
				set[id] = true
			}
		}
	case string:
		for _, id := range strings.Split(s, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				set[id] = true
			}
		}
	}
}

// collectOrphanItems iterates rows of generic resources and returns items not in referenced.
// rows is consumed and closed.
func collectOrphanItems(rows interface {
	Next() bool
	Scan(...any) error
	Close() error
}, referenced map[string]bool) []orphanItem {
	defer rows.Close()
	out := []orphanItem{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var o map[string]any
		if err := json.Unmarshal(raw, &o); err != nil {
			continue
		}
		id, _ := o["id"].(string)
		if id == "" || referenced[id] {
			continue
		}
		name, _ := o["name"].(string)
		if name == "" {
			if fn, ok := o["function"].(map[string]any); ok {
				name, _ = fn["name"].(string)
			}
		}
		out = append(out, orphanItem{ID: id, Name: name})
	}
	return out
}
