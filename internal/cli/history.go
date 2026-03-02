package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View and manage open history",
	Long:  "View, search, and manage your zebro open history.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return historyListCmd.RunE(cmd, args)
	},
}

func init() {
	historyCmd.AddCommand(historyListCmd, historyStatsCmd, historyClearCmd, historyCompactCmd)
	historyListCmd.Flags().IntP("count", "n", 0, "Limit number of entries to show (default: all)")
	historyListCmd.Flags().String("type", "", "Filter by type: link, group, or url")
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

		hf, err := store.LoadHistory(config.ProfileHistoryFile(profile))
		if err != nil {
			return err
		}

		entries := hf.History
		if len(entries) == 0 {
			fmt.Println("no history found")
			return nil
		}

		// Filter by type
		typeFilter, _ := cmd.Flags().GetString("type")
		if typeFilter != "" {
			filtered := entries[:0]
			for _, e := range entries {
				if e.Type == typeFilter {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
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
			urlSummary := strings.Join(e.URLs, ", ")
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

		hf, err := store.LoadHistory(config.ProfileHistoryFile(profile))
		if err != nil {
			return err
		}

		entries := hf.History
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
		for _, t := range []string{"link", "group", "url"} {
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
		path := config.ProfileHistoryFile(profile)
		if err := store.SaveHistory(path, &store.HistoryFile{
			Version: "1",
			History: []store.HistoryEntry{},
		}); err != nil {
			return err
		}
		fmt.Println("cleared history")
		return nil
	},
}

var historyCompactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Apply TTL and limit without opening anything",
	Long:  "Remove expired and excess entries based on history_limit and history_ttl config.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		path := config.ProfileHistoryFile(profile)
		hf, err := store.LoadHistory(path)
		if err != nil {
			return err
		}
		before := len(hf.History)

		// Apply TTL
		if cfg.HistoryTTL > 0 {
			cutoff := time.Now().AddDate(0, 0, -cfg.HistoryTTL)
			filtered := hf.History[:0]
			for _, e := range hf.History {
				if e.Time.After(cutoff) {
					filtered = append(filtered, e)
				}
			}
			hf.History = filtered
		}

		// Apply limit
		if cfg.HistoryLimit > 0 && len(hf.History) > cfg.HistoryLimit {
			hf.History = hf.History[len(hf.History)-cfg.HistoryLimit:]
		}

		after := len(hf.History)
		if err := store.SaveHistory(path, hf); err != nil {
			return err
		}
		removed := before - after
		if removed > 0 {
			fmt.Printf("compacted history: removed %d entries, kept %d\n", removed, after)
		} else {
			fmt.Printf("history already compact (%d entries)\n", after)
		}
		return nil
	},
}
