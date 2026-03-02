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

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage links",
	Long:  "Manage links — URL patterns with optional variable placeholders.",
}

func init() {
	linkCmd.AddCommand(linkListCmd, linkViewCmd, linkCreateCmd, linkDeleteCmd, linkClearCmd)
	linkCreateCmd.Flags().StringP("description", "d", "", "Link description")

	defaultHelp := linkCmd.HelpFunc()
	linkCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != linkCmd {
			defaultHelp(cmd, args)
			return
		}
		p := "@"
		if cfg, err := config.Load(); err == nil {
			p = cfg.VariablePrefix
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Manage links — URL patterns with optional variable placeholders.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "USAGE")
		fmt.Fprintln(w, "  zebro link <subcommand> [flags]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "COMMANDS")
		fmt.Fprintln(w, "  list:\tList all links")
		fmt.Fprintln(w, "  view:\tShow link details")
		fmt.Fprintln(w, "  create:\tAdd or update a link")
		fmt.Fprintln(w, "  delete:\tRemove a link")
		fmt.Fprintln(w, "  clear:\tRemove all links")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "FLAGS")
		fmt.Fprintln(w, "  -h, --help\tShow help for command")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "EXAMPLES")
		fmt.Fprintf(w, "  $ zebro link list\n")
		fmt.Fprintf(w, "  $ zebro link view github\n")
		fmt.Fprintf(w, "  $ zebro link create github https://github.com\n")
		fmt.Fprintf(w, "  $ zebro link create github/%[1]saccount/%[1]srepo https://github.com/%[1]saccount/%[1]srepo\n", p)
		fmt.Fprintf(w, "  $ zebro link create jira/%[1]sticket https://jira.company.com/browse/%[1]sticket -d \"Jira issue\"\n", p)
		fmt.Fprintf(w, "  $ zebro link delete github\n")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "LEARN MORE")
		fmt.Fprintln(w, "  Use \"zebro link <subcommand> --help\" for more information about a command.")
		w.Flush()
	})
}

var linkCreateCmd = &cobra.Command{
	Use:   "create <key> <url>",
	Short: "Add or update a link",
	Long:  "Add a link with an optional description. Variable patterns use the configured prefix (default: @).",
	Example: `  $ zebro link create github https://github.com
  $ zebro link create github/@account/@repo https://github.com/@account/@repo
  $ zebro link create jira/@ticket https://jira.company.com/browse/@ticket -d "Jira issue"`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		rawURL := args[1]
		if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "http://") {
			return fmt.Errorf("URL must start with https:// or http:// (got %q)", rawURL)
		}

		// Step 1: Normalize user-facing prefix (@, ^) to internal VarToken
		normKey := store.NormalizeVars(args[0], cfg.VariablePrefix)
		normURL := store.NormalizeVars(rawURL, cfg.VariablePrefix)

		// Step 2: Validate key and URL have the same variable set
		keyVars := store.ExtractVarNames(normKey)
		urlVars := store.ExtractVarNames(normURL)
		if strings.Join(keyVars, ",") != strings.Join(urlVars, ",") {
			return fmt.Errorf("variable mismatch between key and URL\n  key vars: [%s]\n  url vars: [%s]",
				strings.Join(keyVars, ", "), strings.Join(urlVars, ", "))
		}

		// Step 3: Convert to positional storage
		posKey, params := store.NormalizeToPositional(normKey)
		nameToPos := store.NameToPos(params)
		posURL, _ := store.ApplyPositional(normURL, nameToPos)

		desc, _ := cmd.Flags().GetString("description")

		existing, err := store.ListLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}

		// Check if same key already exists (update)
		var prev *store.Link
		for i, l := range existing {
			if l.Key == posKey {
				prev = &existing[i]
				break
			}
		}

		link := store.Link{
			Key:         posKey,
			URL:         posURL,
			Description: desc,
			Params:      params,
		}

		if err := store.AddLink(config.ProfileLinksFile(profile), link); err != nil {
			return err
		}
		if prev != nil {
			prevKey := store.DenormalizeParams(prev.Key, cfg.VariablePrefix, prev.Params)
			prevURL := store.DenormalizeParams(prev.URL, cfg.VariablePrefix, prev.Params)
			newURL := store.DenormalizeParams(posURL, cfg.VariablePrefix, params)
			fmt.Printf("updated link %q\n", prevKey)
			fmt.Printf("  was: %s", prevURL)
			if prev.Description != "" {
				fmt.Printf(" (%s)", prev.Description)
			}
			fmt.Println()
			fmt.Printf("  now: %s", newURL)
			if desc != "" {
				fmt.Printf(" (%s)", desc)
			}
			fmt.Println()
		} else {
			fmt.Printf("added link %q → %s\n", args[0], args[1])
		}
		return nil
	},
}

var linkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all links",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		links, err := store.ListLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}

		if len(links) == 0 {
			fmt.Println("no links found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tURL\tDESCRIPTION")
		fmt.Fprintln(w, "---\t---\t-----------")
		for _, l := range links {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				store.DenormalizeParams(l.Key, cfg.VariablePrefix, l.Params),
				store.DenormalizeParams(l.URL, cfg.VariablePrefix, l.Params),
				l.Description)
		}
		return w.Flush()
	},
}

var linkViewCmd = &cobra.Command{
	Use:               "view <key>",
	Short:             "Show link details",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeLinkKeys,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		normKey := store.NormalizeVars(args[0], cfg.VariablePrefix)
		posKey, _ := store.NormalizeToPositional(normKey)

		link, err := store.GetLink(config.ProfileLinksFile(profile), posKey)
		if err != nil {
			return fmt.Errorf("link %q not found", args[0])
		}

		fmt.Printf("key:         %s\n", store.DenormalizeParams(link.Key, cfg.VariablePrefix, link.Params))
		fmt.Printf("url:         %s\n", store.DenormalizeParams(link.URL, cfg.VariablePrefix, link.Params))
		if p := formatParams(cfg.VariablePrefix, link.Params); p != "" {
			fmt.Printf("params:      %s\n", p)
		}
		if link.Description != "" {
			fmt.Printf("description: %s\n", link.Description)
		}
		return nil
	},
}

var linkDeleteCmd = &cobra.Command{
	Use:               "delete <key>",
	Short:             "Remove a link",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeLinkKeys,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		normKey := store.NormalizeVars(args[0], cfg.VariablePrefix)
		posKey, _ := store.NormalizeToPositional(normKey)

		link, err := store.GetLink(config.ProfileLinksFile(profile), posKey)
		if err != nil {
			return fmt.Errorf("link %q not found", args[0])
		}
		if err := store.RemoveLink(config.ProfileLinksFile(profile), posKey); err != nil {
			return err
		}
		fmt.Printf("removed link %q: %s\n", args[0], store.DenormalizeParams(link.URL, cfg.VariablePrefix, link.Params))
		return nil
	},
}

var linkClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all links",
	Long: `Remove all links. Creates a backup at links.yaml.bak by default.
Use --force to skip the backup.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		path := config.ProfileLinksFile(profile)
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			if err := backupFile(path); err != nil {
				return fmt.Errorf("creating backup: %w", err)
			}
		}
		if err := store.SaveLinks(path, &store.LinkFile{
			Version: "1",
			Links:   map[string]store.LinkEntry{},
		}); err != nil {
			return err
		}
		if force {
			fmt.Println("cleared all links")
		} else {
			fmt.Printf("cleared all links (backup: %s.bak)\n", path)
		}
		return nil
	},
}

func init() {
	linkClearCmd.Flags().Bool("force", false, "Skip backup and delete immediately")
}

// completeLinkKeys returns link keys and alias names for tab completion.
func completeLinkKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	profile, cfg, err := currentProfile()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err == nil {
		for _, l := range links {
			completions = append(completions, store.DenormalizeParams(l.Key, cfg.VariablePrefix, l.Params))
		}
	}

	aliases, err := store.ListAliases(config.ProfileAliasesFile(profile))
	if err == nil {
		for _, a := range aliases {
			completions = append(completions, a.Name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
