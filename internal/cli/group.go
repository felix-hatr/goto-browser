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
	groupCmd.AddCommand(groupListCmd, groupViewCmd, groupCreateCmd, groupDeleteCmd, groupAddCmd, groupClearCmd)
	groupCreateCmd.Flags().StringP("description", "d", "", "Group description")
	groupAddCmd.Flags().Int("at", 0, "Position to insert (1-based, default: append to end)")
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name> [link-key...]",
	Short: "Create a new group",
	Long: `Create a named group of links that can be opened together.
The group name may include variables (e.g. dev/@account/@repo).
Link keys may reference the group's variables or be concrete.`,
	Example: `  $ zebro group create morning github jira/PROJ-100
  $ zebro group create dev/@account/@repo github/@account github/@account/@repo`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		name := args[0]
		rawLinkKeys := args[1:]
		desc, _ := cmd.Flags().GetString("description")

		// Normalize group name and extract variable mapping
		normName := store.NormalizeVars(name, cfg.VariablePrefix)
		posName, params := store.NormalizeToPositional(normName)
		nameToPos := store.NameToPos(params)

		// Process link keys: normalize and apply group's variable mapping
		posLinkKeys, err := normalizeGroupLinks(rawLinkKeys, cfg.VariablePrefix, nameToPos, posName)
		if err != nil {
			return err
		}

		// Validate all link keys are resolvable (concrete or variable template)
		if err := validateGroupLinks(profile, cfg.VariablePrefix, posLinkKeys); err != nil {
			return err
		}

		prev, _ := store.GetGroup(config.ProfileGroupsFile(profile), posName)

		group := store.Group{
			Name:        posName,
			Description: desc,
			Params:      params,
			Links:       posLinkKeys,
		}

		if err := store.AddGroup(config.ProfileGroupsFile(profile), group); err != nil {
			return err
		}

		if prev != nil {
			fmt.Printf("updated group %q\n", name)
			fmt.Printf("  was: [%s]\n", denormalizeGroupLinks(prev.Links, cfg.VariablePrefix, prev.Params))
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
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		// Normalize the input name to find the group
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

		// Build a key→link map for O(1) lookup
		linkMap := make(map[string]store.Link, len(links))
		for _, lnk := range links {
			linkMap[lnk.Key] = lnk
		}

		fmt.Printf("links (%d):\n", len(group.Links))
		for _, l := range group.Links {
			displayKey := displayVar(l, cfg.VariablePrefix, group.Params, cfg.VariableDisplay)
			var displayURL string
			if lnk, ok := linkMap[l]; ok {
				// Use group params for URL display — positions align with group vars
				displayURL = displayVar(lnk.URL, cfg.VariablePrefix, group.Params, cfg.VariableDisplay)
			} else {
				// Direct URL or other: fall back to resolver-based lookup
				displayURL = resolveLinkURL(l, links, cfg.VariablePrefix)
			}
			if displayURL != "" {
				fmt.Printf("  - %s → %s\n", displayKey, displayURL)
			} else {
				fmt.Printf("  - %s\n", displayKey)
			}
		}
		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:               "delete <name>",
	Short:             "Remove a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		normName := store.NormalizeVars(args[0], cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)

		prev, err := store.GetGroup(config.ProfileGroupsFile(profile), posName)
		if err != nil {
			return fmt.Errorf("group %q not found", args[0])
		}
		if err := store.RemoveGroup(config.ProfileGroupsFile(profile), posName); err != nil {
			return err
		}
		fmt.Printf("removed group %q: [%s]\n", args[0], denormalizeGroupLinks(prev.Links, cfg.VariablePrefix, prev.Params))
		return nil
	},
}

var groupAddCmd = &cobra.Command{
	Use:               "add <name> <link-key...>",
	Short:             "Add links to a group",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		name := args[0]
		rawLinkKeys := args[1:]
		at, _ := cmd.Flags().GetInt("at")

		// Load existing group to get its variable mapping
		normName := store.NormalizeVars(name, cfg.VariablePrefix)
		posName, _ := store.NormalizeToPositional(normName)

		group, err := store.GetGroup(config.ProfileGroupsFile(profile), posName)
		if err != nil {
			return fmt.Errorf("group %q not found", name)
		}
		nameToPos := store.NameToPos(group.Params)

		// Process new link keys using group's variable mapping
		posLinkKeys, err := normalizeGroupLinks(rawLinkKeys, cfg.VariablePrefix, nameToPos, group.Name)
		if err != nil {
			return err
		}

		// Validate all link keys are resolvable (concrete or variable template)
		if err := validateGroupLinks(profile, cfg.VariablePrefix, posLinkKeys); err != nil {
			return err
		}

		if err := store.InsertIntoGroup(config.ProfileGroupsFile(profile), posName, posLinkKeys, at); err != nil {
			return err
		}

		displayKeys := denormalizeGroupLinks(posLinkKeys, cfg.VariablePrefix, group.Params)
		if at > 0 {
			fmt.Printf("added %s to group %q at position %d\n", displayKeys, name, at)
		} else {
			fmt.Printf("added %s to group %q\n", displayKeys, name)
		}
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

func init() {
	groupClearCmd.Flags().Bool("force", false, "Skip backup and delete immediately")
}

// normalizeGroupLinks normalizes a list of user-facing link keys to positional form.
// posGroupName is the stored group name (may contain <vp>N tokens for positional groups).
func normalizeGroupLinks(keys []string, variablePrefix string, nameToPos map[string]int, posGroupName string) ([]string, error) {
	result := make([]string, len(keys))
	isPositionalGroup := len(nameToPos) == 0 && store.HasVars(posGroupName)
	for i, key := range keys {
		norm := store.NormalizeVars(key, variablePrefix)
		switch {
		case len(nameToPos) > 0:
			// Named variable group: map link vars to group positions
			pos, err := store.ApplyPositional(norm, nameToPos)
			if err != nil {
				return nil, fmt.Errorf("%q: %w", key, err)
			}
			result[i] = pos
		case isPositionalGroup:
			// Positional variable group (@1, @2): only positional refs allowed
			if len(store.ExtractVarNames(norm)) > 0 {
				return nil, fmt.Errorf("%q uses named variables — positional group requires @1, @2, ... style", key)
			}
			result[i] = norm
		default:
			// Concrete group: no variables allowed
			if store.HasVars(norm) {
				return nil, fmt.Errorf("%q contains a variable — this group has no variables defined in its name", key)
			}
			result[i] = norm
		}
	}
	return result, nil
}

// validateGroupLinks checks that all group link keys (in positional form) are resolvable.
// For concrete links, it resolves directly. For variable templates (<vp>N tokens),
// it substitutes a dummy value and validates the resulting key pattern.
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
