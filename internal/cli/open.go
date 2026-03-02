package cli

import (
	"fmt"

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

func init() {
	openCmd.Flags().BoolVarP(&openNewWindow, "new-window", "n", false, "Open in a new window (overrides config open_mode)")
	openCmd.Flags().BoolVarP(&openNewTab, "new-tab", "t", false, "Open in a new tab (overrides config open_mode)")
	openCmd.Flags().BoolVar(&openDryRun, "dry-run", false, "Print URL(s) without opening the browser")
	openCmd.Flags().StringVarP(&openBrowserOverride, "browser", "b", "", "Browser to use for this command")
	openCmd.Flags().StringVarP(&openGroupFlag, "group", "g", "", "Open a group by name")
	openCmd.Flags().StringVarP(&openLinkFlag, "link", "l", "", "Open a link by key")

	openCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if openGroupFlag != "" {
			return completeGroupNames(cmd, args, toComplete)
		}
		if openLinkFlag != "" {
			return completeLinkKeys(cmd, args, toComplete)
		}
		// No explicit flag: use open_default config to decide completion type
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

Use -l/--link to open a link, or -g/--group to open a group.
Without a flag, the positional argument is treated as a link or group
based on the open_default config setting (default: link).`,
	Example: `  $ zebro open github/octocat/hello-world
  $ zebro open jira/PROJ-123
  $ zebro open -g morning
  $ zebro open -l github/octocat/hello-world
  $ zebro open jira/PROJ-123 --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if openNewWindow && openNewTab {
			return fmt.Errorf("--new-window and --new-tab are mutually exclusive")
		}
		if openGroupFlag != "" && openLinkFlag != "" {
			return fmt.Errorf("-g/--group and -l/--link are mutually exclusive")
		}
		if openGroupFlag != "" {
			return runOpenGroup(openGroupFlag)
		}
		if openLinkFlag != "" {
			return runOpenLinkKey(openLinkFlag)
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		// No explicit flag: use open_default config
		_, cfg, err := currentProfile()
		if err != nil {
			return err
		}
		if cfg.OpenDefault == "group" {
			return runOpenGroup(args[0])
		}
		return runOpenLinkKey(args[0])
	},
}

func runOpenLinkKey(key string) error {
	profile, cfg, err := currentProfile()
	if err != nil {
		return err
	}

	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return err
	}

	r := resolver.New(cfg.VariablePrefix)
	result, err := r.Resolve(key, links)
	if err != nil {
		return err
	}

	return openURLWithConfig(cfg, result.URL)
}

func runOpenGroup(input string) error {
	profile, cfg, err := currentProfile()
	if err != nil {
		return err
	}

	groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
	if err != nil {
		return err
	}

	r := resolver.New(cfg.VariablePrefix)

	group, groupVars, err := r.MatchGroup(input, groups)
	if err != nil {
		return err
	}

	if len(group.Links) == 0 {
		return fmt.Errorf("group %q has no links", group.Name)
	}

	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return err
	}

	urls, errs := r.ResolveGroupLinks(group.Links, groupVars, links)
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

	return b.OpenURLs(urls, browser.OpenOptions{
		NewWindow: true,
	})
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
