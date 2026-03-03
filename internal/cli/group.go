package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/resolver"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage groups",
	Long:  "Manage groups — named collections of links that open together.",
}

func init() {
	groupCmd.AddCommand(groupListCmd, groupViewCmd, groupCreateCmd, groupDeleteCmd, groupAddCmd, groupRemoveCmd, groupClearCmd, groupRenameCmd, groupSearchCmd, groupExportCmd, groupImportCmd)
	groupExportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	groupImportCmd.Flags().Bool("replace", false, "Replace existing data instead of merging")
	groupCreateCmd.Flags().StringP("description", "d", "", "Group description")
	groupCreateCmd.Flags().StringArrayP("link", "l", nil, "Link key to add (repeatable)")
	groupCreateCmd.Flags().StringArrayP("url", "u", nil, "Direct URL to add (repeatable)")
	groupAddCmd.Flags().Int("at", 0, "Position to insert (1-based, default: append to end)")
	groupAddCmd.Flags().StringArrayP("link", "l", nil, "Link key to add (repeatable)")
	groupAddCmd.Flags().StringArrayP("url", "u", nil, "Direct URL to add (repeatable)")
	groupRemoveCmd.Flags().Int("at", 0, "Position to remove (1-based)")
	groupRemoveCmd.Flags().StringArrayP("link", "l", nil, "Link key to remove (repeatable)")

	groupCreateCmd.RegisterFlagCompletionFunc("link", completeLinkKeysAll)
	groupAddCmd.RegisterFlagCompletionFunc("link", completeLinkKeysAll)
	groupRemoveCmd.RegisterFlagCompletionFunc("link", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		posName, _ := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)
		group, err := store.GetGroup(config.ProfileGroupsFile(profile), posName)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, len(group.URLs))
		for i, u := range group.URLs {
			names[i] = store.DenormalizeParams(u, cfg.VariablePrefix, group.Params)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	defaultHelp := groupCmd.HelpFunc()
	groupCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != groupCmd {
			defaultHelp(cmd, args)
			return
		}
		p := loadVariablePrefix()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Manage groups — named collections of links that open together.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "USAGE")
		fmt.Fprintln(w, "  zebro group <subcommand> [flags]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "COMMANDS")
		fmt.Fprintln(w, "  list:\tList all groups")
		fmt.Fprintln(w, "  view:\tShow group details")
		fmt.Fprintln(w, "  search:\tSearch groups by keyword")
		fmt.Fprintln(w, "  create:\tCreate a new group")
		fmt.Fprintln(w, "  rename:\tRename a group")
		fmt.Fprintln(w, "  delete:\tRemove a group")
		fmt.Fprintln(w, "  add:\tAdd links or URLs to a group")
		fmt.Fprintln(w, "  remove:\tRemove entries from a group")
		fmt.Fprintln(w, "  clear:\tRemove all groups")
		fmt.Fprintln(w, "  export:\tExport groups to a YAML file")
		fmt.Fprintln(w, "  import:\tImport groups from a YAML file")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "FLAGS")
		fmt.Fprintln(w, "  -h, --help\tShow help for command")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "VARIABLES")
		fmt.Fprintf(w, "  Group names can include variable placeholders — %[1]sname or %[1]s1, %[1]s2 style.\n", p)
		fmt.Fprintf(w, "  Variables in the group name map to its URLs by position.\n")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Concrete group:\n")
		fmt.Fprintf(w, "    zebro group create morning -l github -l jira/PROJ-100\n")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Variable group (%[1]sname style):\n", p)
		fmt.Fprintf(w, "    zebro group create dev/%[1]saccount/%[1]srepo -l github/%[1]saccount -l github/%[1]saccount/%[1]srepo\n", p)
		fmt.Fprintf(w, "    zebro open -g dev/myorg/myrepo    # %[1]saccount=myorg, %[1]srepo=myrepo\n", p)
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Direct URL:\n")
		fmt.Fprintf(w, "    zebro group create morning --url https://example.com -l github\n")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  variable_prefix   prefix character (current: %s)  →  zebro config set variable_prefix\n", p)
		fmt.Fprintf(w, "  variable_display  affects output only  →  zebro config set variable_display named|positional\n")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "EXAMPLES")
		fmt.Fprintf(w, "  $ zebro group list\n")
		fmt.Fprintf(w, "  $ zebro group create morning -l github -l slack\n")
		fmt.Fprintf(w, "  $ zebro group create morning --url https://example.com\n")
		fmt.Fprintf(w, "  $ zebro group add morning -l jira/PROJ-100\n")
		fmt.Fprintf(w, "  $ zebro group remove morning -l slack\n")
		fmt.Fprintf(w, "  $ zebro group view morning\n")
		fmt.Fprintf(w, "  $ zebro group delete morning\n")
		fmt.Fprintf(w, "  $ zebro open -g morning\n")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "LEARN MORE")
		fmt.Fprintln(w, "  Use \"zebro group <subcommand> --help\" for more information about a command.")
		w.Flush()
	})
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name> [-l <link-key>...] [-u <url>...]",
	Short: "Create a new group",
	Long: `Create a named group of links that open together with 'zebro open -g'.

The group name may include variable placeholders (e.g. dev/@account/@repo).
Variables in the name are mapped positionally to the URLs.
Use -l to add links by key, or -u/--url to add direct URLs.

If the group already exists, it is replaced.`,
	Example: `  $ zebro group create morning -l github -l slack -l jira/PROJ-100
  $ zebro group create morning --url https://example.com -l github
  $ zebro group create dev/@account/@repo -l github/@account -l github/@account/@repo
  $ zebro group create focus -d "deep work"`,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		name := args[0]
		rawLinkKeys, _ := cmd.Flags().GetStringArray("link")
		rawURLs, _ := cmd.Flags().GetStringArray("url")
		desc, _ := cmd.Flags().GetString("description")

		posName, params := store.NormalizeAndPositionalize(name, cfg.VariablePrefix)
		nameToPos := store.NameToPos(params)

		lf, err := store.LoadLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}

		r := resolver.New(cfg.VariablePrefix)
		posURLTemplates, err := r.ResolveGroupEntries(rawLinkKeys, rawURLs, nameToPos, posName, lf, store.LinksFromFile(lf))
		if err != nil {
			return err
		}

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		_, hasPrev := gf.Groups[posName]
		gf.Groups[posName] = store.GroupEntry{
			Description: desc,
			Params:      params,
			URLs:        posURLTemplates,
		}
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}

		if hasPrev {
			fmt.Printf("updated group %q with %d URL(s)\n", name, len(posURLTemplates))
		} else {
			fmt.Printf("created group %q with %d URL(s)\n", name, len(posURLTemplates))
		}
		return nil
	},
}

var groupListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all groups",
	Long:    "List all groups in the current profile with their URL counts.",
	Example: "  $ zebro group list",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
		if err != nil {
			return err
		}

		if len(groups) == 0 {
			fmt.Println("no groups found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if cfg.VariableDisplay == "positional" {
			fmt.Fprintln(w, "NAME\tURLS\tDESCRIPTION\tPARAMS")
			fmt.Fprintln(w, "----\t----\t-----------\t------")
			for _, g := range groups {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
					store.DenormalizeVars(g.Name, cfg.VariablePrefix),
					len(g.URLs),
					g.Description,
					formatParams(cfg.VariablePrefix, g.Params))
			}
		} else {
			fmt.Fprintln(w, "NAME\tURLS\tDESCRIPTION")
			fmt.Fprintln(w, "----\t----\t-----------")
			for _, g := range groups {
				fmt.Fprintf(w, "%s\t%d\t%s\n",
					store.DenormalizeParams(g.Name, cfg.VariablePrefix, g.Params),
					len(g.URLs),
					g.Description)
			}
		}
		return w.Flush()
	},
}

var groupViewCmd = &cobra.Command{
	Use:   "view <name>",
	Short: "Show group details",
	Long: `Show details of a group including its URLs.
URLs are listed in order with 1-based position numbers.`,
	Example: `  $ zebro group view morning
  $ zebro group view dev/@account/@repo`,
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		posName, _ := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)

		group, err := store.GetGroup(config.ProfileGroupsFile(profile), posName)
		if err != nil {
			return fmt.Errorf("group %q not found (run: zebro group list)", args[0])
		}

		fmt.Printf("name:        %s\n", displayVar(group.Name, cfg.VariablePrefix, group.Params, cfg.VariableDisplay))
		if cfg.VariableDisplay == "positional" {
			if p := formatParams(cfg.VariablePrefix, group.Params); p != "" {
				fmt.Printf("params:      %s\n", p)
			}
		}
		if group.Description != "" {
			fmt.Printf("description: %s\n", group.Description)
		}

		fmt.Printf("urls (%d):\n", len(group.URLs))
		for i, urlTmpl := range group.URLs {
			displayURL := displayVar(urlTmpl, cfg.VariablePrefix, group.Params, cfg.VariableDisplay)
			fmt.Printf("  %d. %s\n", i+1, displayURL)
		}
		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:               "delete <name>",
	Short:             "Remove a group",
	Long:              "Remove a group by name.",
	Example:           "  $ zebro group delete morning",
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		posName, _ := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		entry, ok := gf.Groups[posName]
		if !ok {
			return fmt.Errorf("group %q not found (run: zebro group list)", args[0])
		}
		delete(gf.Groups, posName)
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}
		fmt.Printf("removed group %q (%d URL(s))\n", args[0], len(entry.URLs))
		return nil
	},
}

var groupAddCmd = &cobra.Command{
	Use:   "add <name> [-l <link-key>...] [-u <url>...]",
	Short: "Add links or URLs to a group",
	Long: `Add one or more link keys or direct URLs to an existing group.

By default entries are appended to the end. Use --at to insert at a specific
1-based position, shifting existing entries down.`,
	Example: `  $ zebro group add morning -l notion
  $ zebro group add morning -l notion -l figma
  $ zebro group add morning --url https://example.com
  $ zebro group add morning -l notion --at 1`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeGroupNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		rawLinkKeys, _ := cmd.Flags().GetStringArray("link")
		rawURLs, _ := cmd.Flags().GetStringArray("url")
		if len(rawLinkKeys) == 0 && len(rawURLs) == 0 {
			return fmt.Errorf("requires at least 1 link key (-l) or URL (-u/--url)")
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		name := args[0]
		at, _ := cmd.Flags().GetInt("at")

		posName, _ := store.NormalizeAndPositionalize(name, cfg.VariablePrefix)

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		entry, ok := gf.Groups[posName]
		if !ok {
			return fmt.Errorf("group %q not found (run: zebro group list)", name)
		}

		lf, err := store.LoadLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}

		nameToPos := store.NameToPos(entry.Params)
		r := resolver.New(cfg.VariablePrefix)
		newURLTemplates, err := r.ResolveGroupEntries(rawLinkKeys, rawURLs, nameToPos, posName, lf, store.LinksFromFile(lf))
		if err != nil {
			return err
		}

		if at <= 0 || at > len(entry.URLs) {
			entry.URLs = append(entry.URLs, newURLTemplates...)
		} else {
			merged := make([]string, 0, len(entry.URLs)+len(newURLTemplates))
			merged = append(merged, entry.URLs[:at-1]...)
			merged = append(merged, newURLTemplates...)
			merged = append(merged, entry.URLs[at-1:]...)
			entry.URLs = merged
		}
		gf.Groups[posName] = entry
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}

		if at > 0 {
			fmt.Printf("added %d URL(s) to group %q at position %d\n", len(newURLTemplates), name, at)
		} else {
			fmt.Printf("added %d URL(s) to group %q\n", len(newURLTemplates), name)
		}
		return nil
	},
}

var groupRemoveCmd = &cobra.Command{
	Use:   "remove <name> [-l <link-key>...] [--at <position>]",
	Short: "Remove entries from a group",
	Long: `Remove URLs from a group by link key or by position.
Removing by link key (-l) removes the first matching URL resolved from that key.
Removing by position (--at) removes the URL at that 1-based index.`,
	Example: `  $ zebro group remove morning -l slack
  $ zebro group remove morning -l github -l slack
  $ zebro group remove morning --at 2`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeGroupNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		keys, _ := cmd.Flags().GetStringArray("link")
		at, _ := cmd.Flags().GetInt("at")

		if at > 0 && len(keys) > 0 {
			return fmt.Errorf("--at and -l/--link are mutually exclusive")
		}
		if at == 0 && len(keys) == 0 {
			return fmt.Errorf("specify -l <link-key> to remove or use --at <position>")
		}

		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		posName, _ := store.NormalizeAndPositionalize(name, cfg.VariablePrefix)

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		entry, ok := gf.Groups[posName]
		if !ok {
			return fmt.Errorf("group %q not found (run: zebro group list)", name)
		}

		if at > 0 {
			if at > len(entry.URLs) {
				return fmt.Errorf("position %d out of range (group has %d URL(s))", at, len(entry.URLs))
			}
			removed := entry.URLs[at-1]
			newURLs := make([]string, 0, len(entry.URLs)-1)
			newURLs = append(newURLs, entry.URLs[:at-1]...)
			newURLs = append(newURLs, entry.URLs[at:]...)
			entry.URLs = newURLs
			gf.Groups[posName] = entry
			if err := store.SaveGroups(groupsPath, gf); err != nil {
				return err
			}
			displayURL := store.DenormalizeParams(removed, cfg.VariablePrefix, entry.Params)
			fmt.Printf("removed %s from group %q (position %d)\n", displayURL, name, at)
			return nil
		}

		// Remove by link key — resolve each key to its URL template and remove matching entries
		lf, err := store.LoadLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}
		nameToPos := store.NameToPos(entry.Params)
		r := resolver.New(cfg.VariablePrefix)
		removeURLs, err := r.ResolveGroupEntries(keys, nil, nameToPos, posName, lf, store.LinksFromFile(lf))
		if err != nil {
			return err
		}
		removedCount, err := store.RemoveFromGroup(groupsPath, posName, removeURLs)
		if err != nil {
			return err
		}
		if removedCount == 0 {
			return fmt.Errorf("no matching entries found in group %q", name)
		}
		fmt.Printf("removed %s from group %q\n", strings.Join(keys, ", "), name)
		return nil
	},
}

var groupClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all groups",
	Long:  "Remove all groups from the current profile.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		path := config.ProfileGroupsFile(profile)
		if err := store.SaveGroups(path, &store.GroupFile{
			Version: "1",
			Groups:  map[string]store.GroupEntry{},
		}); err != nil {
			return err
		}
		fmt.Println("cleared all groups")
		return nil
	},
}

var groupRenameCmd = &cobra.Command{
	Use:               "rename <old-name> <new-name>",
	Short:             "Rename a group",
	Args:              cobra.MaximumNArgs(2),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		oldPosName, oldParams := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)
		newPosName, newParams := store.NormalizeAndPositionalize(args[1], cfg.VariablePrefix)

		if len(oldParams) != len(newParams) {
			return fmt.Errorf("variable count mismatch: old name has %d variable(s), new name has %d", len(oldParams), len(newParams))
		}

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}

		if _, ok := gf.Groups[oldPosName]; !ok {
			return fmt.Errorf("group %q not found (run: zebro group list)", args[0])
		}
		if _, ok := gf.Groups[newPosName]; ok {
			return fmt.Errorf("group %q already exists", args[1])
		}

		entry := gf.Groups[oldPosName]
		entry.Params = newParams
		gf.Groups[newPosName] = entry
		delete(gf.Groups, oldPosName)

		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}

		displayOld := displayVar(oldPosName, cfg.VariablePrefix, oldParams, cfg.VariableDisplay)
		displayNew := displayVar(newPosName, cfg.VariablePrefix, newParams, cfg.VariableDisplay)
		fmt.Printf("renamed group %q → %q\n", displayOld, displayNew)
		return nil
	},
}

// completeLinkKeysAll returns link keys for tab completion without the single-arg guard.
// Used in commands where link keys appear at position >= 1 (e.g. group add, group create).
func completeLinkKeysAll(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	profile, cfg, err := currentProfile()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	recent, rset := recentSet(config.ProfileHistoryFile(profile, "link"))
	completions := make([]string, 0, len(recent)+len(links))
	completions = append(completions, recent...)
	for _, l := range links {
		key := displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay)
		if !rset[key] {
			completions = append(completions, key)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeGroupNames returns group names for tab completion.
func completeGroupNamesAll(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	profile, cfg, err := currentProfile()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	recent, rset := recentSet(config.ProfileHistoryFile(profile, "group"))
	completions := make([]string, 0, len(recent)+len(groups))
	completions = append(completions, recent...)
	for _, g := range groups {
		name := displayVar(g.Name, cfg.VariablePrefix, g.Params, cfg.VariableDisplay)
		if !rset[name] {
			completions = append(completions, name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeGroupNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeGroupNamesAll(cmd, args, toComplete)
}

var groupSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search groups by keyword",
	Long:  "Search groups by name or description (case-insensitive substring match).",
	Example: `  $ zebro group search morning
  $ zebro group search dev`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
		if err != nil {
			return err
		}

		kLower := strings.ToLower(keyword)
		var matched []store.Group
		for _, g := range groups {
			name := displayVar(g.Name, cfg.VariablePrefix, g.Params, cfg.VariableDisplay)
			if strings.Contains(strings.ToLower(name), kLower) ||
				strings.Contains(strings.ToLower(g.Description), kLower) {
				matched = append(matched, g)
			}
		}

		if len(matched) == 0 {
			fmt.Printf("no groups matching %q\n", keyword)
			return nil
		}

		var buf bytes.Buffer
		w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		if cfg.VariableDisplay == "positional" {
			fmt.Fprintln(w, "NAME\tURLS\tDESCRIPTION\tPARAMS")
			fmt.Fprintln(w, "----\t----\t-----------\t------")
			for _, g := range matched {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
					store.DenormalizeVars(g.Name, cfg.VariablePrefix),
					len(g.URLs),
					g.Description,
					formatParams(cfg.VariablePrefix, g.Params))
			}
		} else {
			fmt.Fprintln(w, "NAME\tURLS\tDESCRIPTION")
			fmt.Fprintln(w, "----\t----\t-----------")
			for _, g := range matched {
				fmt.Fprintf(w, "%s\t%d\t%s\n",
					store.DenormalizeParams(g.Name, cfg.VariablePrefix, g.Params),
					len(g.URLs),
					g.Description)
			}
		}
		w.Flush()
		fmt.Print(highlightKeyword(buf.String(), keyword))
		return nil
	},
}

var groupExportCmd = &cobra.Command{
	Use:   "export [-o <file>]",
	Short: "Export groups to a YAML file",
	Long:  "Export all groups in the current profile to a YAML file (default: stdout).",
	Example: `  $ zebro group export
  $ zebro group export -o /tmp/groups.yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		gf, err := store.LoadGroups(config.ProfileGroupsFile(profile))
		if err != nil {
			return err
		}
		ef := &store.ExportFile{
			Version: "1",
			Groups:  gf.Groups,
		}
		outPath, _ := cmd.Flags().GetString("output")
		if outPath == "" {
			data, err := store.MarshalExportFile(ef)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		}
		if err := store.SaveExportFile(outPath, ef); err != nil {
			return err
		}
		fmt.Printf("exported %d group(s) to %s\n", len(gf.Groups), outPath)
		return nil
	},
}

var groupImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import groups from a YAML file",
	Long: `Import groups from an export YAML file.

By default (merge mode): new groups are added, existing names are skipped.
Use --replace to overwrite all existing groups with the imported data.`,
	Example: `  $ zebro group import /tmp/groups.yaml
  $ zebro group import /tmp/groups.yaml --replace`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		ef, err := store.LoadExportFile(args[0])
		if err != nil {
			return err
		}
		if len(ef.Groups) == 0 {
			fmt.Println("no groups found in export file")
			return nil
		}

		replace, _ := cmd.Flags().GetBool("replace")
		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}

		if replace {
			gf.Groups = ef.Groups
			if err := store.SaveGroups(groupsPath, gf); err != nil {
				return err
			}
			fmt.Printf("replaced groups: imported %d\n", len(ef.Groups))
			return nil
		}

		imported, skipped := 0, 0
		for name, entry := range ef.Groups {
			if _, exists := gf.Groups[name]; exists {
				skipped++
				continue
			}
			gf.Groups[name] = entry
			imported++
		}
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}
		fmt.Printf("imported %d, skipped %d\n", imported, skipped)
		return nil
	},
}
