package cli

import (
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
	groupCmd.AddCommand(groupListCmd, groupViewCmd, groupCreateCmd, groupDeleteCmd, groupAddCmd, groupRemoveCmd, groupClearCmd)
	groupCreateCmd.Flags().StringP("description", "d", "", "Group description")
	groupAddCmd.Flags().Int("at", 0, "Position to insert (1-based, default: append to end)")
	groupRemoveCmd.Flags().Int("at", 0, "Position to remove (1-based)")
	groupClearCmd.Flags().Bool("force", false, "Skip backup and delete immediately")

	defaultHelp := groupCmd.HelpFunc()
	groupCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != groupCmd {
			defaultHelp(cmd, args)
			return
		}
		p := "@"
		if cfg, err := config.Load(); err == nil {
			p = cfg.VariablePrefix
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Manage groups — named collections of links that open together.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "USAGE")
		fmt.Fprintln(w, "  zebro group <subcommand> [flags]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "COMMANDS")
		fmt.Fprintln(w, "  list:\tList all groups")
		fmt.Fprintln(w, "  view:\tShow group details")
		fmt.Fprintln(w, "  create:\tCreate a new group")
		fmt.Fprintln(w, "  delete:\tRemove a group")
		fmt.Fprintln(w, "  add:\tAdd links to a group")
		fmt.Fprintln(w, "  remove:\tRemove links from a group")
		fmt.Fprintln(w, "  clear:\tRemove all groups")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "FLAGS")
		fmt.Fprintln(w, "  -h, --help\tShow help for command")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "VARIABLES")
		fmt.Fprintf(w, "  Group names can include variable placeholders — %[1]sname or %[1]s1, %[1]s2 style.\n", p)
		fmt.Fprintf(w, "  Variables in the group name map to its link keys by position.\n")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Concrete group:\n")
		fmt.Fprintf(w, "    zebro group create morning github jira/PROJ-100\n")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Variable group (%[1]sname style):\n", p)
		fmt.Fprintf(w, "    zebro group create dev/%[1]saccount/%[1]srepo github/%[1]saccount github/%[1]saccount/%[1]srepo\n", p)
		fmt.Fprintf(w, "    zebro open -g dev/myorg/myrepo    # %[1]saccount=myorg, %[1]srepo=myrepo\n", p)
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  variable_prefix   prefix character (current: %s)  →  zebro config set variable_prefix\n", p)
		fmt.Fprintf(w, "  variable_display  affects output only  →  zebro config set variable_display named|positional\n")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "EXAMPLES")
		fmt.Fprintf(w, "  $ zebro group list\n")
		fmt.Fprintf(w, "  $ zebro group create morning github slack\n")
		fmt.Fprintf(w, "  $ zebro group add morning jira/PROJ-100\n")
		fmt.Fprintf(w, "  $ zebro group remove morning slack\n")
		fmt.Fprintf(w, "  $ zebro group view morning\n")
		fmt.Fprintf(w, "  $ zebro open -g morning\n")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "LEARN MORE")
		fmt.Fprintln(w, "  Use \"zebro group <subcommand> --help\" for more information about a command.")
		w.Flush()
	})
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name> [link-key...]",
	Short: "Create a new group",
	Long: `Create a named group of links that can be opened together.
The group name may include variables (e.g. dev/@account/@repo).
Link keys may reference the group's variables or be concrete.`,
	Example: `  $ zebro group create morning github jira/PROJ-100
  $ zebro group create dev/@account/@repo github/@account github/@account/@repo`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeLinkKeysAll(cmd, args, toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		name := args[0]
		rawLinkKeys := args[1:]
		desc, _ := cmd.Flags().GetString("description")

		normName := store.NormalizeVars(name, cfg.VariablePrefix)
		posName, params := store.NormalizeToPositional(normName)
		nameToPos := store.NameToPos(params)

		posLinkKeys, err := normalizeGroupLinks(rawLinkKeys, cfg.VariablePrefix, nameToPos, posName)
		if err != nil {
			return err
		}
		if err := validateGroupLinks(profile, cfg.VariablePrefix, posLinkKeys); err != nil {
			return err
		}

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		prevEntry, hasPrev := gf.Groups[posName]
		gf.Groups[posName] = store.GroupEntry{
			Description: desc,
			Params:      params,
			Links:       posLinkKeys,
		}
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}

		if hasPrev {
			fmt.Printf("updated group %q\n", name)
			fmt.Printf("  was: [%s]\n", denormalizeGroupLinks(prevEntry.Links, cfg.VariablePrefix, prevEntry.Params))
			fmt.Printf("  now: [%s]\n", denormalizeGroupLinks(posLinkKeys, cfg.VariablePrefix, params))
		} else {
			fmt.Printf("created group %q with %d link(s)\n", name, len(rawLinkKeys))
		}
		return nil
	},
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups",
	Args:  cobra.NoArgs,
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
			fmt.Fprintln(w, "NAME\tLINKS\tDESCRIPTION\tPARAMS")
			fmt.Fprintln(w, "----\t-----\t-----------\t------")
			for _, g := range groups {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
					store.DenormalizeVars(g.Name, cfg.VariablePrefix),
					len(g.Links),
					g.Description,
					formatParams(cfg.VariablePrefix, g.Params))
			}
		} else {
			fmt.Fprintln(w, "NAME\tLINKS\tDESCRIPTION")
			fmt.Fprintln(w, "----\t-----\t-----------")
			for _, g := range groups {
				fmt.Fprintf(w, "%s\t%d\t%s\n",
					store.DenormalizeParams(g.Name, cfg.VariablePrefix, g.Params),
					len(g.Links),
					g.Description)
			}
		}
		return w.Flush()
	},
}

var groupViewCmd = &cobra.Command{
	Use:               "view <name>",
	Short:             "Show group details",
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		normName := store.NormalizeVars(args[0], cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)

		group, err := store.GetGroup(config.ProfileGroupsFile(profile), posName)
		if err != nil {
			return fmt.Errorf("group %q not found", args[0])
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

		links, err := store.ListLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}
		linkMap := make(map[string]store.Link, len(links))
		for _, lnk := range links {
			linkMap[lnk.Key] = lnk
		}

		fmt.Printf("links (%d):\n", len(group.Links))
		for i, l := range group.Links {
			displayKey := displayVar(l, cfg.VariablePrefix, group.Params, cfg.VariableDisplay)
			var displayURL string
			if lnk, ok := linkMap[l]; ok {
				displayURL = displayVar(lnk.URL, cfg.VariablePrefix, group.Params, cfg.VariableDisplay)
			} else {
				displayURL = resolveLinkURL(l, links, cfg.VariablePrefix)
			}
			if displayURL != "" {
				fmt.Printf("  %d. %s → %s\n", i+1, displayKey, displayURL)
			} else {
				fmt.Printf("  %d. %s\n", i+1, displayKey)
			}
		}
		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:               "delete <name>",
	Short:             "Remove a group",
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		normName := store.NormalizeVars(args[0], cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		entry, ok := gf.Groups[posName]
		if !ok {
			return fmt.Errorf("group %q not found", args[0])
		}
		delete(gf.Groups, posName)
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}
		fmt.Printf("removed group %q: [%s]\n", args[0], denormalizeGroupLinks(entry.Links, cfg.VariablePrefix, entry.Params))
		return nil
	},
}

var groupAddCmd = &cobra.Command{
	Use:   "add <name> <link-key...>",
	Short: "Add links to a group",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeGroupNames(cmd, args, toComplete)
		}
		return completeLinkKeysAll(cmd, args, toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		if len(args) < 2 {
			return fmt.Errorf("requires at least 1 link key: group add <name> <link-key...>")
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		name := args[0]
		rawLinkKeys := args[1:]
		at, _ := cmd.Flags().GetInt("at")

		normName := store.NormalizeVars(name, cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		entry, ok := gf.Groups[posName]
		if !ok {
			return fmt.Errorf("group %q not found", name)
		}

		posLinkKeys, err := normalizeGroupLinks(rawLinkKeys, cfg.VariablePrefix, store.NameToPos(entry.Params), posName)
		if err != nil {
			return err
		}
		if err := validateGroupLinks(profile, cfg.VariablePrefix, posLinkKeys); err != nil {
			return err
		}

		if at <= 0 || at > len(entry.Links) {
			entry.Links = append(entry.Links, posLinkKeys...)
		} else {
			merged := make([]string, 0, len(entry.Links)+len(posLinkKeys))
			merged = append(merged, entry.Links[:at-1]...)
			merged = append(merged, posLinkKeys...)
			merged = append(merged, entry.Links[at-1:]...)
			entry.Links = merged
		}
		gf.Groups[posName] = entry
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}

		displayKeys := denormalizeGroupLinks(posLinkKeys, cfg.VariablePrefix, entry.Params)
		if at > 0 {
			fmt.Printf("added %s to group %q at position %d\n", displayKeys, name, at)
		} else {
			fmt.Printf("added %s to group %q\n", displayKeys, name)
		}
		return nil
	},
}

var groupRemoveCmd = &cobra.Command{
	Use:   "remove <name> [link-key...]",
	Short: "Remove links from a group",
	Long: `Remove links from a group by key or by position.
Removing by key removes all occurrences of that link.
Removing by position (--at) removes the link at that 1-based index.`,
	Example: `  $ zebro group remove morning slack
  $ zebro group remove morning github slack
  $ zebro group remove morning --at 2`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeGroupNames(cmd, args, toComplete)
		}
		at, _ := cmd.Flags().GetInt("at")
		if at > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// Suggest link keys currently in this group
		profile, cfg, err := currentProfile()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		normName := store.NormalizeVars(args[0], cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)
		group, err := store.GetGroup(config.ProfileGroupsFile(profile), posName)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, len(group.Links))
		for i, l := range group.Links {
			names[i] = displayVar(l, cfg.VariablePrefix, group.Params, cfg.VariableDisplay)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		keys := args[1:]
		at, _ := cmd.Flags().GetInt("at")

		if at > 0 && len(keys) > 0 {
			return fmt.Errorf("--at and link-key arguments are mutually exclusive")
		}
		if at == 0 && len(keys) == 0 {
			return fmt.Errorf("specify link-key(s) to remove or use --at <position>")
		}

		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		normName := store.NormalizeVars(name, cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)

		groupsPath := config.ProfileGroupsFile(profile)
		gf, err := store.LoadGroups(groupsPath)
		if err != nil {
			return err
		}
		entry, ok := gf.Groups[posName]
		if !ok {
			return fmt.Errorf("group %q not found", name)
		}

		if at > 0 {
			if at > len(entry.Links) {
				return fmt.Errorf("position %d out of range (group has %d link(s))", at, len(entry.Links))
			}
			removed := entry.Links[at-1]
			newLinks := make([]string, 0, len(entry.Links)-1)
			newLinks = append(newLinks, entry.Links[:at-1]...)
			newLinks = append(newLinks, entry.Links[at:]...)
			entry.Links = newLinks
			gf.Groups[posName] = entry
			if err := store.SaveGroups(groupsPath, gf); err != nil {
				return err
			}
			fmt.Printf("removed %s from group %q (position %d)\n",
				displayVar(removed, cfg.VariablePrefix, entry.Params, cfg.VariableDisplay), name, at)
			return nil
		}

		// Remove by key — all occurrences
		normKeys, err := normalizeGroupLinks(keys, cfg.VariablePrefix, store.NameToPos(entry.Params), posName)
		if err != nil {
			return err
		}
		removeSet := make(map[string]bool, len(normKeys))
		for _, k := range normKeys {
			removeSet[k] = true
		}
		filtered := make([]string, 0, len(entry.Links))
		for _, l := range entry.Links {
			if !removeSet[l] {
				filtered = append(filtered, l)
			}
		}
		removedCount := len(entry.Links) - len(filtered)
		if removedCount == 0 {
			return fmt.Errorf("no matching links found in group %q", name)
		}
		entry.Links = filtered
		gf.Groups[posName] = entry
		if err := store.SaveGroups(groupsPath, gf); err != nil {
			return err
		}
		fmt.Printf("removed %s from group %q\n",
			denormalizeGroupLinks(normKeys, cfg.VariablePrefix, entry.Params), name)
		return nil
	},
}

var groupClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all groups",
	Long: `Remove all groups. Creates a backup at groups.yaml.bak by default.
Use --force to skip the backup.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		path := config.ProfileGroupsFile(profile)
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			if err := backupFile(path); err != nil {
				return fmt.Errorf("creating backup: %w", err)
			}
		}
		if err := store.SaveGroups(path, &store.GroupFile{
			Version: "1",
			Groups:  map[string]store.GroupEntry{},
		}); err != nil {
			return err
		}
		if force {
			fmt.Println("cleared all groups")
		} else {
			fmt.Printf("cleared all groups (backup: %s.bak)\n", path)
		}
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
	completions := make([]string, len(links))
	for i, l := range links {
		completions[i] = displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// normalizeGroupLinks normalizes a list of user-facing link keys to positional form
// using the group's variable mapping.
func normalizeGroupLinks(keys []string, variablePrefix string, nameToPos map[string]int, posGroupName string) ([]string, error) {
	result := make([]string, len(keys))
	isPositionalGroup := len(nameToPos) == 0 && store.HasVars(posGroupName)
	for i, key := range keys {
		norm := store.NormalizeVars(key, variablePrefix)
		switch {
		case len(nameToPos) > 0:
			pos, err := store.ApplyPositional(norm, nameToPos)
			if err != nil {
				return nil, fmt.Errorf("%q: %w", key, err)
			}
			result[i] = pos
		case isPositionalGroup:
			if len(store.ExtractVarNames(norm)) > 0 {
				return nil, fmt.Errorf("%q uses named variables — positional group requires %[2]s1, %[2]s2, ... style", key, variablePrefix)
			}
			result[i] = norm
		default:
			if store.HasVars(norm) {
				return nil, fmt.Errorf("%q contains a variable — this group has no variables defined in its name", key)
			}
			result[i] = norm
		}
	}
	return result, nil
}

// validateGroupLinks checks that all group link keys (in positional form) are resolvable.
func validateGroupLinks(profile, variablePrefix string, posLinkKeys []string) error {
	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return err
	}

	r := resolver.New(variablePrefix)
	dummy := []string{"x", "x", "x", "x", "x", "x", "x", "x"}

	var invalid []string
	for _, posKey := range posLinkKeys {
		testKey := store.DenormalizeVars(store.FillPositional(posKey, dummy), variablePrefix)
		if _, err := r.Resolve(testKey, links); err != nil {
			invalid = append(invalid, store.DenormalizeVars(posKey, variablePrefix))
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("unknown link key(s): %s\nrun 'zebro link list' to see available keys", strings.Join(invalid, ", "))
	}
	return nil
}

// denormalizeGroupLinks converts a list of positional group link templates to display form.
func denormalizeGroupLinks(links []string, prefix string, params []string) string {
	names := make([]string, len(links))
	for i, l := range links {
		names[i] = store.DenormalizeParams(l, prefix, params)
	}
	return strings.Join(names, ", ")
}

// completeGroupNames returns group names for tab completion.
func completeGroupNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	profile, cfg, err := currentProfile()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := store.ListGroups(config.ProfileGroupsFile(profile))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = displayVar(g.Name, cfg.VariablePrefix, g.Params, cfg.VariableDisplay)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
