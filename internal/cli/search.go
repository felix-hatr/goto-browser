package cli

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search links, groups, and history",
	Long: `Search across links, groups, and history for the given keyword.

Displays results in LINKS / GROUPS / HISTORY sections (sections with no
results are omitted). Keyword matching is case-insensitive substring match.`,
	Example: `  $ zebro search github
  $ zebro search jira`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		kLower := strings.ToLower(keyword)

		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		// --- Links ---
		links, err := store.ListLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}
		var matchedLinks []store.Link
		for _, l := range links {
			key := displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay)
			url := displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay)
			if strings.Contains(strings.ToLower(key), kLower) ||
				strings.Contains(strings.ToLower(url), kLower) ||
				strings.Contains(strings.ToLower(l.Description), kLower) {
				matchedLinks = append(matchedLinks, l)
			}
		}

		// --- Groups ---
		groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
		if err != nil {
			return err
		}
		var matchedGroups []store.Group
		for _, g := range groups {
			name := displayVar(g.Name, cfg.VariablePrefix, g.Params, cfg.VariableDisplay)
			if strings.Contains(strings.ToLower(name), kLower) ||
				strings.Contains(strings.ToLower(g.Description), kLower) {
				matchedGroups = append(matchedGroups, g)
			}
		}

		// --- History ---
		histEntries, err := allHistoryEntries(profile)
		if err != nil {
			return err
		}
		var matchedHistory []store.HistoryEntry
		for _, e := range histEntries {
			urlSummary := e.DisplayURLs()
			if strings.Contains(strings.ToLower(e.Target), kLower) ||
				strings.Contains(strings.ToLower(urlSummary), kLower) {
				matchedHistory = append(matchedHistory, e)
			}
		}
		// Reverse history to most-recent-first
		if len(matchedHistory) > 0 {
			rev := make([]store.HistoryEntry, len(matchedHistory))
			for i, e := range matchedHistory {
				rev[len(matchedHistory)-1-i] = e
			}
			matchedHistory = rev
		}

		if len(matchedLinks) == 0 && len(matchedGroups) == 0 && len(matchedHistory) == 0 {
			fmt.Printf("no results matching %q\n", keyword)
			return nil
		}

		var buf bytes.Buffer
		first := true

		if len(matchedLinks) > 0 {
			first = false
			fmt.Fprintln(&buf, "LINKS")
			w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
			if cfg.VariableDisplay == "positional" {
				fmt.Fprintln(w, "  KEY\tURL\tDESCRIPTION\tPARAMS")
				fmt.Fprintln(w, "  ---\t---\t-----------\t------")
				for _, l := range matchedLinks {
					fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
						displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
						displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
						l.Description,
						formatParams(cfg.VariablePrefix, l.Params))
				}
			} else {
				fmt.Fprintln(w, "  KEY\tURL\tDESCRIPTION")
				fmt.Fprintln(w, "  ---\t---\t-----------")
				for _, l := range matchedLinks {
					fmt.Fprintf(w, "  %s\t%s\t%s\n",
						displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
						displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
						l.Description)
				}
			}
			w.Flush()
		}

		if len(matchedGroups) > 0 {
			if !first {
				fmt.Fprintln(&buf)
			}
			first = false
			fmt.Fprintln(&buf, "GROUPS")
			w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
			if cfg.VariableDisplay == "positional" {
				fmt.Fprintln(w, "  NAME\tURLS\tDESCRIPTION\tPARAMS")
				fmt.Fprintln(w, "  ----\t----\t-----------\t------")
				for _, g := range matchedGroups {
					fmt.Fprintf(w, "  %s\t%d\t%s\t%s\n",
						store.DenormalizeVars(g.Name, cfg.VariablePrefix),
						len(g.URLs),
						g.Description,
						formatParams(cfg.VariablePrefix, g.Params))
				}
			} else {
				fmt.Fprintln(w, "  NAME\tURLS\tDESCRIPTION")
				fmt.Fprintln(w, "  ----\t----\t-----------")
				for _, g := range matchedGroups {
					fmt.Fprintf(w, "  %s\t%d\t%s\n",
						store.DenormalizeParams(g.Name, cfg.VariablePrefix, g.Params),
						len(g.URLs),
						g.Description)
				}
			}
			w.Flush()
		}

		if len(matchedHistory) > 0 {
			if !first {
				fmt.Fprintln(&buf)
			}
			fmt.Fprintln(&buf, "HISTORY")
			w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  TIME\tTYPE\tTARGET\tURLs")
			fmt.Fprintln(w, "  ----\t----\t------\t----")
			for _, e := range matchedHistory {
				timeStr := e.Time.Local().Format("2006-01-02 15:04")
				urlSummary := e.DisplayURLs()
				if len(urlSummary) > 60 {
					urlSummary = urlSummary[:57] + "..."
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", timeStr, e.Type, e.Target, urlSummary)
			}
			w.Flush()
		}

		fmt.Print(highlightKeyword(buf.String(), keyword))
		return nil
	},
}
