package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

// allHistoryEntries loads and merges history from all type-specific files, sorted oldest-first.
func allHistoryEntries(profile string) ([]store.HistoryEntry, error) {
	var all []store.HistoryEntry
	for _, typ := range store.HistoryTypes {
		entries, err := store.LoadHistory(config.ProfileHistoryFile(profile, typ))
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Time.Before(all[j].Time)
	})
	return all, nil
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View and manage open history",
	Long:  "View, search, and manage your zebro open history.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	historyCmd.AddCommand(historyListCmd, historyStatsCmd, historyClearCmd, historyCompactCmd)
	historyListCmd.Flags().SortFlags = false
	historyListCmd.Flags().BoolP("link", "l", false, "Show only link history")
	historyListCmd.Flags().BoolP("group", "g", false, "Show only group history")
	historyListCmd.Flags().BoolP("url", "u", false, "Show only URL history")
	historyListCmd.Flags().IntP("count", "n", 0, "Limit number of entries to show (default: all)")
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List history entries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}

		linkOnly, _ := cmd.Flags().GetBool("link")
		groupOnly, _ := cmd.Flags().GetBool("group")
		urlOnly, _ := cmd.Flags().GetBool("url")

		flagCount := 0
		if linkOnly {
			flagCount++
		}
		if groupOnly {
			flagCount++
		}
		if urlOnly {
			flagCount++
		}
		if flagCount > 1 {
			return fmt.Errorf("-l/--link, -g/--group, and -u/--url are mutually exclusive")
		}

		var entries []store.HistoryEntry
		switch {
		case linkOnly:
			entries, err = store.LoadHistory(config.ProfileHistoryFile(profile, "link"))
		case groupOnly:
			entries, err = store.LoadHistory(config.ProfileHistoryFile(profile, "group"))
		case urlOnly:
			entries, err = store.LoadHistory(config.ProfileHistoryFile(profile, "url"))
		default:
			entries, err = allHistoryEntries(profile)
		}
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("no history found")
			return nil
		}

		// Show most recent first
		reversed := make([]store.HistoryEntry, len(entries))
		for i, e := range entries {
			reversed[len(entries)-1-i] = e
		}
		entries = reversed

		// Limit count
		n, _ := cmd.Flags().GetInt("count")
		if n > 0 && n < len(entries) {
			entries = entries[:n]
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIME\tTYPE\tTARGET\tURLs")
		fmt.Fprintln(w, "----\t----\t------\t----")
		for _, e := range entries {
			timeStr := e.Time.Local().Format("2006-01-02 15:04")
			urlSummary := e.DisplayURLs()
			if len(urlSummary) > 60 {
				urlSummary = urlSummary[:57] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", timeStr, e.Type, e.Target, urlSummary)
		}
		return w.Flush()
	},
}

var historyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show history statistics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}

		entries, err := allHistoryEntries(profile)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("no history found")
			return nil
		}

		// Count by type
		typeCounts := map[string]int{}
		for _, e := range entries {
			typeCounts[e.Type]++
		}

		// Count by target (frequency)
		type targetCount struct {
			target string
			kind   string
			count  int
		}
		targetMap := map[string]*targetCount{}
		for _, e := range entries {
			key := e.Type + ":" + e.Target
			if tc, ok := targetMap[key]; ok {
				tc.count++
			} else {
				targetMap[key] = &targetCount{target: e.Target, kind: e.Type, count: 1}
			}
		}

		// Sort by frequency descending
		topTargets := make([]*targetCount, 0, len(targetMap))
		for _, tc := range targetMap {
			topTargets = append(topTargets, tc)
		}
		sort.Slice(topTargets, func(i, j int) bool {
			if topTargets[i].count != topTargets[j].count {
				return topTargets[i].count > topTargets[j].count
			}
			return topTargets[i].target < topTargets[j].target
		})

		// Find oldest and newest
		oldest := entries[0].Time
		newest := entries[len(entries)-1].Time
		for _, e := range entries {
			if e.Time.Before(oldest) {
				oldest = e.Time
			}
			if e.Time.After(newest) {
				newest = e.Time
			}
		}

		fmt.Printf("total entries: %d\n", len(entries))
		fmt.Printf("by type:\n")
		for _, t := range store.HistoryTypes {
			if c, ok := typeCounts[t]; ok {
				fmt.Printf("  %-8s %d\n", t+":", c)
			}
		}

		fmt.Printf("most opened:\n")
		limit := 5
		if limit > len(topTargets) {
			limit = len(topTargets)
		}
		for _, tc := range topTargets[:limit] {
			fmt.Printf("  %-40s (%s, %d\u00d7)\n", tc.target, tc.kind, tc.count)
		}

		fmt.Printf("oldest: %s\n", oldest.Local().Format("2006-01-02 15:04"))
		fmt.Printf("newest: %s\n", newest.Local().Format("2006-01-02 15:04"))
		return nil
	},
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all history",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		for _, typ := range store.HistoryTypes {
			if err := store.SaveHistory(config.ProfileHistoryFile(profile, typ), nil); err != nil {
				return err
			}
		}
		fmt.Println("cleared history")
		return nil
	},
}

var historyCompactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Deduplicate and trim history",
	Long: `Apply erasedups-style deduplication and history_size limit.

Keeps only the latest occurrence of each target per type.
Always applies history_size limit if configured.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		totalBefore, totalAfter := 0, 0
		for _, typ := range store.HistoryTypes {
			path := config.ProfileHistoryFile(profile, typ)
			entries, err := store.LoadHistory(path)
			if err != nil {
				return err
			}
			totalBefore += len(entries)

			// Erasedups: keep only the latest occurrence of each target.
			seen := make(map[string]bool, len(entries))
			deduped := make([]store.HistoryEntry, 0, len(entries))
			for i := len(entries) - 1; i >= 0; i-- {
				key := entries[i].Target
				if !seen[key] {
					seen[key] = true
					deduped = append(deduped, entries[i])
				}
			}
			// Reverse to restore chronological order
			for i, j := 0, len(deduped)-1; i < j; i, j = i+1, j-1 {
				deduped[i], deduped[j] = deduped[j], deduped[i]
			}

			// Apply size limit
			if cfg.HistorySize > 0 && len(deduped) > cfg.HistorySize {
				deduped = deduped[len(deduped)-cfg.HistorySize:]
			}

			totalAfter += len(deduped)
			if err := store.SaveHistory(path, deduped); err != nil {
				return err
			}
		}

		removed := totalBefore - totalAfter
		if removed > 0 {
			fmt.Printf("compacted history: removed %d entries, kept %d\n", removed, totalAfter)
		} else {
			fmt.Printf("history already compact (%d entries)\n", totalAfter)
		}
		return nil
	},
}
