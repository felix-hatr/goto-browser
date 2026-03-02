package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage aliases",
	Long:  "Manage aliases — short names that expand to link key prefixes.",
}

func init() {
	aliasCmd.AddCommand(aliasListCmd, aliasViewCmd, aliasCreateCmd, aliasDeleteCmd, aliasClearCmd)
}

var aliasCreateCmd = &cobra.Command{
	Use:   "create <name> <link-key>",
	Short: "Create an alias for a link key",
	Long:  "Create a short alias that expands to a link key prefix.",
	Example: `  $ zebro alias create gh github
  $ zebro alias create g google
  $ zebro open gh/octocat/hello-world   # expands to github/octocat/hello-world`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		target := args[1]
		if err := validateAliasTarget(profile, target, cfg.VariablePrefix); err != nil {
			return err
		}

		prevTarget, _ := store.GetAlias(config.ProfileAliasesFile(profile), args[0])

		if err := store.AddAlias(config.ProfileAliasesFile(profile), args[0], target); err != nil {
			return err
		}
		if prevTarget != "" {
			fmt.Printf("updated alias %q\n", args[0])
			fmt.Printf("  was: %s\n", prevTarget)
			fmt.Printf("  now: %s\n", target)
		} else {
			fmt.Printf("added alias %q → %q\n", args[0], target)
		}
		return nil
	},
}

var aliasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all aliases",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		aliases, err := store.ListAliases(config.ProfileAliasesFile(profile))
		if err != nil {
			return err
		}

		if len(aliases) == 0 {
			fmt.Println("no aliases found")
			return nil
		}

		links, aliasMap, err := loadLinksAndAliases(profile)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tLINK KEY\tURL")
		fmt.Fprintln(w, "----\t--------\t---")
		for _, a := range aliases {
			url := resolveLinkURL(a.LinkKey, links, aliasMap, cfg.VariablePrefix)
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.LinkKey, url)
		}
		return w.Flush()
	},
}

var aliasViewCmd = &cobra.Command{
	Use:               "view <name>",
	Short:             "Show alias target",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAliasNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		target, err := store.GetAlias(config.ProfileAliasesFile(profile), args[0])
		if err != nil {
			return err
		}

		links, aliasMap, err := loadLinksAndAliases(profile)
		if err != nil {
			return err
		}
		url := resolveLinkURL(target, links, aliasMap, cfg.VariablePrefix)

		if url != "" {
			fmt.Printf("%s → %s (%s)\n", args[0], target, url)
		} else {
			fmt.Printf("%s → %s\n", args[0], target)
		}
		return nil
	},
}

var aliasDeleteCmd = &cobra.Command{
	Use:               "delete <name>",
	Short:             "Remove an alias",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAliasNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}

		if err := store.RemoveAlias(config.ProfileAliasesFile(profile), args[0]); err != nil {
			return err
		}
		fmt.Printf("removed alias %q\n", args[0])
		return nil
	},
}

var aliasClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all aliases",
	Long: `Remove all aliases. Creates a backup at aliases.yaml.bak by default.
Use --force to skip the backup.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		path := config.ProfileAliasesFile(profile)
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			if err := backupFile(path); err != nil {
				return fmt.Errorf("creating backup: %w", err)
			}
		}
		if err := store.SaveAliases(path, &store.AliasFile{
			Version: "1",
			Aliases: map[string]string{},
		}); err != nil {
			return err
		}
		if force {
			fmt.Println("cleared all aliases")
		} else {
			fmt.Printf("cleared all aliases (backup: %s.bak)\n", path)
		}
		return nil
	},
}

func init() {
	aliasClearCmd.Flags().Bool("force", false, "Skip backup and delete immediately")
}

// validateAliasTarget checks that the target matches the first segment of at least one link key.
func validateAliasTarget(profile, target, variablePrefix string) error {
	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return err
	}
	for _, l := range links {
		key := store.DenormalizeVars(l.Key, variablePrefix)
		first := strings.SplitN(key, "/", 2)[0]
		if first == target {
			return nil
		}
	}
	return fmt.Errorf("no link key starts with %q\nrun 'zebro link list' to see available keys", target)
}

// completeAliasNames returns alias names for tab completion.
func completeAliasNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	profile, _, err := currentProfile()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	aliases, err := store.ListAliases(config.ProfileAliasesFile(profile))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, len(aliases))
	for i, a := range aliases {
		names[i] = a.Name
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
