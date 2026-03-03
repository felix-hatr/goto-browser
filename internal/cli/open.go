package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/felix-hatr/goto-browser/internal/browser"
	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/resolver"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

// openItem holds a single open target with its kind (link/group/url).
type openItem struct {
	kind  string
	value string
}

// openItemAccumulator implements pflag.Value so that -l/-g/-u flags append to a shared slice
// in command-line order.
type openItemAccumulator struct {
	items *[]openItem
	kind  string
}

func (a *openItemAccumulator) Set(v string) error {
	*a.items = append(*a.items, openItem{a.kind, v})
	return nil
}

func (a *openItemAccumulator) Type() string { return "string" }

func (a *openItemAccumulator) String() string {
	var vals []string
	for _, it := range *a.items {
		if it.kind == a.kind {
			vals = append(vals, it.value)
		}
	}
	return strings.Join(vals, ",")
}

var openItems []openItem
var openNewWindow bool
var openNewTab bool
var openDryRun bool
var openBrowserOverride string
var openNoHistory bool

func init() {
	openCmd.Flags().SortFlags = false
	openCmd.Flags().VarP(&openItemAccumulator{&openItems, "link"}, "link", "l", "Open a link by key (repeatable)")
	openCmd.Flags().VarP(&openItemAccumulator{&openItems, "group"}, "group", "g", "Open a group by name (repeatable)")
	openCmd.Flags().VarP(&openItemAccumulator{&openItems, "url"}, "url", "u", "Open a direct URL (repeatable)")
	openCmd.Flags().BoolVarP(&openNewWindow, "new-window", "n", false, "Open in a new window (overrides config open_mode)")
	openCmd.Flags().BoolVarP(&openNewTab, "new-tab", "t", false, "Open in a new tab (overrides config open_mode)")
	openCmd.Flags().StringVarP(&openBrowserOverride, "browser", "b", "", "Browser to use for this command")
	openCmd.Flags().BoolVar(&openDryRun, "dry-run", false, "Print URL(s) without opening the browser")
	openCmd.Flags().BoolVar(&openNoHistory, "no-history", false, "Do not record to history")

	openCmd.RegisterFlagCompletionFunc("link", completeLinkKeysAll)
	openCmd.RegisterFlagCompletionFunc("group", completeGroupNamesAll)

	openCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Determine last kind from openItems for context-aware completion
		if len(openItems) > 0 {
			last := openItems[len(openItems)-1]
			switch last.kind {
			case "group":
				return completeGroupNames(cmd, args, toComplete)
			case "link":
				return completeLinkKeys(cmd, args, toComplete)
			}
		}
		_, cfg, err := currentProfile()
		if err == nil && cfg.OpenDefault == "group" {
			return completeGroupNames(cmd, args, toComplete)
		}
		return completeLinkKeys(cmd, args, toComplete)
	}
}

var openCmd = &cobra.Command{
	Use:   "open [key...]",
	Short: "Open a link or group in the browser",
	Long: `Open one or more links, groups, or URLs in the browser.

Use -l/--link to open a link, -g/--group to open a group, or -u/--url to open a direct URL.
Flags are repeatable and processed in the order they appear on the command line.
Without flags, positional arguments are treated as links, groups, or URLs
based on the open_default config setting (default: link).`,
	Example: `  $ zebro open github/octocat/hello-world
  $ zebro open jira/PROJ-123
  $ zebro open -g morning
  $ zebro open -l github -l jira/PROJ-100
  $ zebro open -l github -g morning
  $ zebro open -u https://example.com
  $ zebro open jira/PROJ-123 --dry-run
  $ zebro open github --no-history`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if openNewWindow && openNewTab {
			return fmt.Errorf("--new-window and --new-tab are mutually exclusive")
		}

		// Load config once for all paths
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		// Build the ordered list of items to open
		// First: -l/-g/-u flags (already in openItems in command-line order)
		// Then: positional args (treated by open_default)
		items := make([]openItem, len(openItems))
		copy(items, openItems)

		for _, arg := range args {
			switch cfg.OpenDefault {
			case "group":
				items = append(items, openItem{"group", arg})
			case "url":
				items = append(items, openItem{"url", arg})
			default:
				items = append(items, openItem{"link", arg})
			}
		}

		if len(items) == 0 {
			return cmd.Help()
		}

		// Process items in order, fail-fast
		for _, item := range items {
			switch item.kind {
			case "url":
				if err := openURLWithConfig(cfg, item.value); err != nil {
					return err
				}
				if !openDryRun && !openNoHistory {
					recordHistory(profile, cfg, store.HistoryEntry{
						Time:   time.Now().UTC(),
						Type:   "url",
						Target: item.value,
					})
				}
			case "group":
				if err := runOpenGroup(item.value, profile, cfg); err != nil {
					return err
				}
			default: // "link"
				if err := runOpenLinkKey(item.value, profile, cfg); err != nil {
					return err
				}
			}
		}
		return nil
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
	if !openDryRun && !openNoHistory {
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
	if !openNoHistory {
		recordHistory(profile, cfg, store.HistoryEntry{
			Time:   time.Now().UTC(),
			Type:   "group",
			Target: input,
			URLs:   urls,
		})
	}
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
