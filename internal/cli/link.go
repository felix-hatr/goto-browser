package cli

import (
	"bytes"
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
	linkCmd.AddCommand(linkListCmd, linkViewCmd, linkCreateCmd, linkDeleteCmd, linkClearCmd, linkRenameCmd, linkSearchCmd, linkExportCmd, linkImportCmd)
	linkExportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	linkImportCmd.Flags().Bool("replace", false, "Replace existing data instead of merging")
	linkCreateCmd.Flags().StringP("description", "d", "", "Link description")

	defaultHelp := linkCmd.HelpFunc()
	linkCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != linkCmd {
			defaultHelp(cmd, args)
			return
		}
		p := loadVariablePrefix()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Manage links — URL patterns with optional variable placeholders.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "USAGE")
		fmt.Fprintln(w, "  zebro link <subcommand> [flags]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "COMMANDS")
		fmt.Fprintln(w, "  list:\tList all links")
		fmt.Fprintln(w, "  view:\tShow link details")
		fmt.Fprintln(w, "  search:\tSearch links by keyword")
		fmt.Fprintln(w, "  create:\tAdd or update a link")
		fmt.Fprintln(w, "  rename:\tRename a link key")
		fmt.Fprintln(w, "  delete:\tRemove a link")
		fmt.Fprintln(w, "  clear:\tRemove all links")
		fmt.Fprintln(w, "  export:\tExport links to a YAML file")
		fmt.Fprintln(w, "  import:\tImport links from a YAML file")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "FLAGS")
		fmt.Fprintln(w, "  -h, --help\tShow help for command")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "VARIABLES")
		fmt.Fprintf(w, "  Variables are matched by position in the key path — %[1]sname is a display label only.\n", p)
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Named style (%[1]sname):\n", p)
		fmt.Fprintf(w, "    zebro link create github/%[1]saccount/%[1]srepo https://github.com/%[1]saccount/%[1]srepo\n", p)
		fmt.Fprintf(w, "    zebro open github/octocat/hello-world    # pos 1 → %[1]saccount=octocat, pos 2 → %[1]srepo=hello-world\n", p)
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Positional style (%[1]s1, %[1]s2)  — same matching, no label:\n", p)
		fmt.Fprintf(w, "    zebro link create repo/%[1]s1/%[1]s2 https://github.com/%[1]s1/%[1]s2\n", p)
		fmt.Fprintf(w, "    zebro open repo/myorg/myrepo            # pos 1 → %[1]s1=myorg, pos 2 → %[1]s2=myrepo\n", p)
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  variable_prefix   prefix character (current: %s)  →  zebro config set variable_prefix\n", p)
		fmt.Fprintf(w, "  variable_display  affects output only: named shows %[1]saccount, positional shows %[1]s1\n", p)
		fmt.Fprintf(w, "                    →  zebro config set variable_display named|positional\n")
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
	Args:              cobra.MaximumNArgs(2),
	ValidArgsFunction: completeLinkKeys,
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
		posKey, params := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)
		nameToPos := store.NameToPos(params)
		posURL, _ := store.ApplyPositional(normURL, nameToPos)

		// Step 3b: Validate positional token sets match (catches @1 vs @1/@2 mismatch)
		keyPos := store.ExtractPositionalNums(posKey)
		urlPos := store.ExtractPositionalNums(posURL)
		if fmt.Sprint(keyPos) != fmt.Sprint(urlPos) {
			return fmt.Errorf("positional variable mismatch between key and URL\n  key positions: %v\n  url positions: %v", keyPos, urlPos)
		}

		desc, _ := cmd.Flags().GetString("description")

		linksPath := config.ProfileLinksFile(profile)
		lf, err := store.LoadLinks(linksPath)
		if err != nil {
			return err
		}

		// Check if same key already exists (update)
		var prev *store.Link
		if entry, ok := lf.Links[posKey]; ok {
			l := store.Link{Key: posKey, URL: entry.URL, Description: entry.Description, Params: entry.Params}
			prev = &l
		}

		lf.Links[posKey] = store.LinkEntry{URL: posURL, Description: desc, Params: params}
		if err := store.SaveLinks(linksPath, lf); err != nil {
			return err
		}
		if prev != nil {
			prevKey := displayVar(prev.Key, cfg.VariablePrefix, prev.Params, cfg.VariableDisplay)
			prevURL := displayVar(prev.URL, cfg.VariablePrefix, prev.Params, cfg.VariableDisplay)
			newURL := displayVar(posURL, cfg.VariablePrefix, params, cfg.VariableDisplay)
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
			newKey := displayVar(posKey, cfg.VariablePrefix, params, cfg.VariableDisplay)
			newURL := displayVar(posURL, cfg.VariablePrefix, params, cfg.VariableDisplay)
			fmt.Printf("added link %q → %s\n", newKey, newURL)
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
		if cfg.VariableDisplay == "positional" {
			fmt.Fprintln(w, "KEY\tURL\tDESCRIPTION\tPARAMS")
			fmt.Fprintln(w, "---\t---\t-----------\t------")
			for _, l := range links {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					l.Description,
					formatParams(cfg.VariablePrefix, l.Params))
			}
		} else {
			fmt.Fprintln(w, "KEY\tURL\tDESCRIPTION")
			fmt.Fprintln(w, "---\t---\t-----------")
			for _, l := range links {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					l.Description)
			}
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

		posKey, _ := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)

		link, err := store.GetLink(config.ProfileLinksFile(profile), posKey)
		if err != nil {
			return fmt.Errorf("link %q not found (run: zebro link list)", args[0])
		}

		fmt.Printf("key:         %s\n", displayVar(link.Key, cfg.VariablePrefix, link.Params, cfg.VariableDisplay))
		fmt.Printf("url:         %s\n", displayVar(link.URL, cfg.VariablePrefix, link.Params, cfg.VariableDisplay))
		if cfg.VariableDisplay == "positional" {
			if p := formatParams(cfg.VariablePrefix, link.Params); p != "" {
				fmt.Printf("params:      %s\n", p)
			}
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

		posKey, _ := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)

		linksPath := config.ProfileLinksFile(profile)
		lf, err := store.LoadLinks(linksPath)
		if err != nil {
			return err
		}
		entry, ok := lf.Links[posKey]
		if !ok {
			return fmt.Errorf("link %q not found (run: zebro link list)", args[0])
		}
		link := store.Link{Key: posKey, URL: entry.URL, Description: entry.Description, Params: entry.Params}
		delete(lf.Links, posKey)
		if err := store.SaveLinks(linksPath, lf); err != nil {
			return err
		}
		fmt.Printf("removed link %q: %s\n", displayVar(link.Key, cfg.VariablePrefix, link.Params, cfg.VariableDisplay), displayVar(link.URL, cfg.VariablePrefix, link.Params, cfg.VariableDisplay))
		return nil
	},
}

var linkClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all links",
	Long:  "Remove all links from the current profile.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		path := config.ProfileLinksFile(profile)
		if err := store.SaveLinks(path, &store.LinkFile{
			Version: "1",
			Links:   map[string]store.LinkEntry{},
		}); err != nil {
			return err
		}
		fmt.Println("cleared all links")
		return nil
	},
}

var linkRenameCmd = &cobra.Command{
	Use:               "rename <old-key> <new-key>",
	Short:             "Rename a link key",
	Args:              cobra.MaximumNArgs(2),
	ValidArgsFunction: completeLinkKeys,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return cmd.Help()
		}
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		oldPosKey, oldParams := store.NormalizeAndPositionalize(args[0], cfg.VariablePrefix)
		newPosKey, newParams := store.NormalizeAndPositionalize(args[1], cfg.VariablePrefix)

		if len(oldParams) != len(newParams) {
			return fmt.Errorf("variable count mismatch: old key has %d variable(s), new key has %d", len(oldParams), len(newParams))
		}

		linksPath := config.ProfileLinksFile(profile)
		lf, err := store.LoadLinks(linksPath)
		if err != nil {
			return err
		}

		if _, ok := lf.Links[oldPosKey]; !ok {
			return fmt.Errorf("link %q not found (run: zebro link list)", args[0])
		}
		if _, ok := lf.Links[newPosKey]; ok {
			return fmt.Errorf("link %q already exists", args[1])
		}

		entry := lf.Links[oldPosKey]
		// Replace params with new key's params (same count, new names)
		entry.Params = newParams
		lf.Links[newPosKey] = entry
		delete(lf.Links, oldPosKey)

		if err := store.SaveLinks(linksPath, lf); err != nil {
			return err
		}

		displayOld := displayVar(oldPosKey, cfg.VariablePrefix, oldParams, cfg.VariableDisplay)
		displayNew := displayVar(newPosKey, cfg.VariablePrefix, newParams, cfg.VariableDisplay)
		fmt.Printf("renamed link %q → %q\n", displayOld, displayNew)
		return nil
	},
}

// completeLinkKeys returns link keys for tab completion.
func completeLinkKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeLinkKeysAll(cmd, args, toComplete)
}

var linkSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search links by keyword",
	Long:  "Search links by key, URL, or description (case-insensitive substring match).",
	Example: `  $ zebro link search github
  $ zebro link search jira`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		profile, cfg, err := currentProfile()
		if err != nil {
			return err
		}

		links, err := store.ListLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}

		kLower := strings.ToLower(keyword)
		var matched []store.Link
		for _, l := range links {
			key := displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay)
			url := displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay)
			if strings.Contains(strings.ToLower(key), kLower) ||
				strings.Contains(strings.ToLower(url), kLower) ||
				strings.Contains(strings.ToLower(l.Description), kLower) {
				matched = append(matched, l)
			}
		}

		if len(matched) == 0 {
			fmt.Printf("no links matching %q\n", keyword)
			return nil
		}

		var buf bytes.Buffer
		w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		if cfg.VariableDisplay == "positional" {
			fmt.Fprintln(w, "KEY\tURL\tDESCRIPTION\tPARAMS")
			fmt.Fprintln(w, "---\t---\t-----------\t------")
			for _, l := range matched {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					l.Description,
					formatParams(cfg.VariablePrefix, l.Params))
			}
		} else {
			fmt.Fprintln(w, "KEY\tURL\tDESCRIPTION")
			fmt.Fprintln(w, "---\t---\t-----------")
			for _, l := range matched {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					displayVar(l.Key, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					displayVar(l.URL, cfg.VariablePrefix, l.Params, cfg.VariableDisplay),
					l.Description)
			}
		}
		w.Flush()
		fmt.Print(highlightKeyword(buf.String(), keyword))
		return nil
	},
}

var linkExportCmd = &cobra.Command{
	Use:   "export [-o <file>]",
	Short: "Export links to a YAML file",
	Long:  "Export all links in the current profile to a YAML file (default: stdout).",
	Example: `  $ zebro link export
  $ zebro link export -o /tmp/links.yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _, err := currentProfile()
		if err != nil {
			return err
		}
		lf, err := store.LoadLinks(config.ProfileLinksFile(profile))
		if err != nil {
			return err
		}
		ef := &store.ExportFile{
			Version: "1",
			Links:   lf.Links,
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
		fmt.Printf("exported %d link(s) to %s\n", len(lf.Links), outPath)
		return nil
	},
}

var linkImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import links from a YAML file",
	Long: `Import links from an export YAML file.

By default (merge mode): new links are added, existing keys are skipped.
Use --replace to overwrite all existing links with the imported data.`,
	Example: `  $ zebro link import /tmp/links.yaml
  $ zebro link import /tmp/links.yaml --replace`,
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
		if len(ef.Links) == 0 {
			fmt.Println("no links found in export file")
			return nil
		}

		replace, _ := cmd.Flags().GetBool("replace")
		linksPath := config.ProfileLinksFile(profile)
		lf, err := store.LoadLinks(linksPath)
		if err != nil {
			return err
		}

		if replace {
			lf.Links = ef.Links
			if err := store.SaveLinks(linksPath, lf); err != nil {
				return err
			}
			fmt.Printf("replaced links: imported %d\n", len(ef.Links))
			return nil
		}

		imported, skipped := 0, 0
		for key, entry := range ef.Links {
			if _, exists := lf.Links[key]; exists {
				skipped++
				continue
			}
			lf.Links[key] = entry
			imported++
		}
		if err := store.SaveLinks(linksPath, lf); err != nil {
			return err
		}
		fmt.Printf("imported %d, skipped %d\n", imported, skipped)
		return nil
	},
}
