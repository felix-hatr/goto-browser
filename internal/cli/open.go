package cli

import (
	"fmt"
	"time"

	"github.com/felix-hatr/goto-browser/internal/browser"
	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/resolver"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var openNewWindow bool
var openNewTab bool
var openDryRun bool
var openBrowserOverride string
var openGroupFlag string
var openLinkFlag string
var openURLFlag string

func init() {
	openCmd.Flags().SortFlags = false
	openCmd.Flags().StringVarP(&openLinkFlag, "link", "l", "", "Open a link by key")
	openCmd.Flags().StringVarP(&openGroupFlag, "group", "g", "", "Open a group by name")
	openCmd.Flags().StringVarP(&openURLFlag, "url", "u", "", "Open a direct URL")
	openCmd.Flags().BoolVarP(&openNewWindow, "new-window", "n", false, "Open in a new window (overrides config open_mode)")
	openCmd.Flags().BoolVarP(&openNewTab, "new-tab", "t", false, "Open in a new tab (overrides config open_mode)")
	openCmd.Flags().StringVarP(&openBrowserOverride, "browser", "b", "", "Browser to use for this command")
	openCmd.Flags().BoolVar(&openDryRun, "dry-run", false, "Print URL(s) without opening the browser")

	openCmd.RegisterFlagCompletionFunc("link", completeLinkKeysAll)
	openCmd.RegisterFlagCompletionFunc("group", completeGroupNamesAll)

	openCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if openGroupFlag != "" {
			return completeGroupNames(cmd, args, toComplete)
		}
		if openLinkFlag != "" {
			return completeLinkKeys(cmd, args, toComplete)
		}
		_, cfg, err := currentProfile()
		if err == nil && cfg.OpenDefault == "group" {
			return completeGroupNames(cmd, args, toComplete)
		}
		return completeLinkKeys(cmd, args, toComplete)
	}
}

var openCmd = &cobra.Command{
	Use:   "open [key]",
	Short: "Open a link or group in the browser",
	Long: `Open a link key or group in the browser.

Use -l/--link to open a link, -g/--group to open a group, or -u/--url to open a direct URL.
Without a flag, the positional argument is treated as a link, group, or URL
based on the open_default config setting (default: link).`,
	Example: `  $ zebro open github/octocat/hello-world
  $ zebro open jira/PROJ-123
  $ zebro open -g morning
  $ zebro open -l github/octocat/hello-world
  $ zebro open -u https://example.com
  $ zebro open jira/PROJ-123 --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if openNewWindow && openNewTab {
			return fmt.Errorf("--new-window and --new-tab are mutually exclusive")
		}
		// Mutual exclusion among -g, -l, -u
		flagCount := 0
		if openGroupFlag != "" {
			flagCount++
		}
		if openLinkFlag != "" {
			flagCount++
		}
		if openURLFlag != "" {
			flagCount++
		}
		if flagCount > 1 {
			return fmt.Errorf("-g/--group, -l/--link, and -u/--url are mutually exclusive")
		}

		// Load config once for all paths
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		switch {
		case openURLFlag != "":
			if err := openURLWithConfig(cfg, openURLFlag); err != nil {
				return err
			}
			if !openDryRun {
				recordHistory(profile, cfg, store.HistoryEntry{
					Time:   time.Now().UTC(),
					Type:   "url",
					Target: openURLFlag,
				})
			}
			return nil
		case openGroupFlag != "":
			return runOpenGroup(openGroupFlag, profile, cfg)
		case openLinkFlag != "":
			return runOpenLinkKey(openLinkFlag, profile, cfg)
		case len(args) == 1:
			target := args[0]
			switch cfg.OpenDefault {
			case "group":
				return runOpenGroup(target, profile, cfg)
			case "url":
				if err := openURLWithConfig(cfg, target); err != nil {
					return err
				}
				if !openDryRun {
					recordHistory(profile, cfg, store.HistoryEntry{
						Time:   time.Now().UTC(),
						Type:   "url",
						Target: target,
					})
				}
				return nil
			default:
				return runOpenLinkKey(target, profile, cfg)
			}
		default:
			return cmd.Help()
		}
	},
}

func runOpenLinkKey(key, profile string, cfg *config.GlobalConfig) error {
	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return err
	}

	r := resolver.New(cfg.VariablePrefix)
	result, err := r.Resolve(key, links)
	if err != nil {
		return err
	}

	if err := openURLWithConfig(cfg, result.URL); err != nil {
		return err
	}
	if !openDryRun {
		recordHistory(profile, cfg, store.HistoryEntry{
			Time:   time.Now().UTC(),
			Type:   "link",
			Target: key,
			URL:    result.URL,
		})
	}
	return nil
}

func runOpenGroup(input, profile string, cfg *config.GlobalConfig) error {
	groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
	if err != nil {
		return err
	}

	r := resolver.New(cfg.VariablePrefix)

	group, groupVars, err := r.MatchGroup(input, groups)
	if err != nil {
		return err
	}

	if len(group.URLs) == 0 {
		return fmt.Errorf("group %q has no URLs", group.Name)
	}

	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return err
	}

	urls, errs := r.ResolveGroupLinks(group.URLs, groupVars, links)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("warning: %v\n", e)
		}
	}
	if len(urls) == 0 {
		return fmt.Errorf("no URLs could be resolved for group %q", group.Name)
	}

	if openDryRun {
		for _, u := range urls {
			fmt.Println(u)
		}
		return nil
	}

	browserName := openBrowserOverride
	if browserName == "" {
		browserName = cfg.Browser
	}

	b, err := browser.New(browserName)
	if err != nil {
		return err
	}

	if err := b.OpenURLs(urls, browser.OpenOptions{NewWindow: true}); err != nil {
		return err
	}
	recordHistory(profile, cfg, store.HistoryEntry{
		Time:   time.Now().UTC(),
		Type:   "group",
		Target: input,
		URLs:   urls,
	})
	return nil
}

func openURLWithConfig(cfg *config.GlobalConfig, url string) error {
	if openDryRun {
		fmt.Println(url)
		return nil
	}

	browserName := openBrowserOverride
	if browserName == "" {
		browserName = cfg.Browser
	}

	b, err := browser.New(browserName)
	if err != nil {
		return err
	}

	opts := browser.OpenOptions{
		NewWindow: openNewWindow,
	}
	if openNewTab {
		opts.NewWindow = false
	} else if !openNewWindow && cfg.OpenMode == "new_window" {
		opts.NewWindow = true
	}

	return b.OpenURL(url, opts)
}

// recordHistory appends an entry to the type-specific history file.
// Errors are silently ignored to not disrupt the open operation.
func recordHistory(profile string, cfg *config.GlobalConfig, entry store.HistoryEntry) {
	_ = store.AppendHistory(
		config.ProfileHistoryFile(profile, entry.Type),
		entry,
		cfg.HistorySize,
		cfg.HistoryDedup,
	)
}
