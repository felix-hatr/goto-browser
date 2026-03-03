package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "Manage profiles — isolated sets of links and groups.",
	Example: `  $ zebro profile view          # show active profile
  $ zebro profile list          # list all profiles
  $ zebro profile create work   # create a new profile
  $ zebro profile use work      # switch active profile`,
}

func init() {
	profileCmd.AddCommand(
		profileListCmd,
		profileViewCmd,
		profileCreateCmd,
		profileDeleteCmd,
		profileUseCmd,
		profileRenameCmd,
		profileBackupCmd,
		profileExportCmd,
		profileImportCmd,
	)
	profileCreateCmd.Flags().StringP("description", "d", "", "Profile description")
	profileCreateCmd.Flags().StringP("source", "s", "", "Copy links and groups from an existing profile")
	profileExportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	profileImportCmd.Flags().String("as", "", "Import as a new profile with this name (default: use name from file)")
	profileImportCmd.Flags().Bool("force", false, "Overwrite existing profile if it exists")

	defaultHelp := profileCmd.HelpFunc()
	profileCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != profileCmd {
			defaultHelp(cmd, args)
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Manage profiles — isolated sets of links, aliases, and groups.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "USAGE")
		fmt.Fprintln(w, "  zebro profile <subcommand> [flags]")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "COMMANDS")
		fmt.Fprintln(w, "  use:\tSwitch the active profile")
		fmt.Fprintln(w, "  list:\tList all profiles")
		fmt.Fprintln(w, "  view:\tShow profile details")
		fmt.Fprintln(w, "  create:\tCreate a new profile")
		fmt.Fprintln(w, "  rename:\tRename a profile")
		fmt.Fprintln(w, "  delete:\tRemove a profile")
		fmt.Fprintln(w, "  export:\tExport a profile to a YAML file")
		fmt.Fprintln(w, "  import:\tImport a profile from a YAML file")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "BACKUP COMMANDS")
		fmt.Fprintln(w, "  backup list:\tList all backups (or backups for a specific profile)")
		fmt.Fprintln(w, "  backup view:\tShow contents of a specific backup")
		fmt.Fprintln(w, "  backup create:\tCreate a manual snapshot of a profile")
		fmt.Fprintln(w, "  backup restore:\tRestore a profile from backup")
		fmt.Fprintln(w, "  backup delete:\tDelete a specific backup")
		fmt.Fprintln(w, "  backup clear:\tDelete all backups for a profile")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "FLAGS")
		fmt.Fprintln(w, "  -h, --help\tShow help for command")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "EXAMPLES")
		fmt.Fprintln(w, "  $ zebro profile use work")
		fmt.Fprintln(w, "  $ zebro profile list")
		fmt.Fprintln(w, "  $ zebro profile view                        # show active profile")
		fmt.Fprintln(w, "  $ zebro profile view work -d                # show with full link/alias/group lists")
		fmt.Fprintln(w, "  $ zebro profile create work -d \"Work links\"")
		fmt.Fprintln(w, "  $ zebro profile rename work work-archive")
		fmt.Fprintln(w, "  $ zebro profile delete work")
		fmt.Fprintln(w, "  $ zebro profile backup create work          # snapshot before changes")
		fmt.Fprintln(w, "  $ zebro profile backup list work            # show all snapshots")
		fmt.Fprintln(w, "  $ zebro profile backup restore work         # restore from latest backup")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "LEARN MORE")
		fmt.Fprintln(w, "  Use \"zebro profile <subcommand> --help\" for more information about a command.")
		w.Flush()
	})
}

// validateProfileName checks that a profile name is safe to use as a directory name.
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("profile name cannot start with '.'")
	}
	if strings.ContainsAny(name, "/ \t\n\\") {
		return fmt.Errorf("profile name cannot contain slashes or whitespace")
	}
	return nil
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Example: `  $ zebro profile create work
  $ zebro profile create work -d "Work profile"
  $ zebro profile create work --source default`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		if err := validateProfileName(name); err != nil {
			return err
		}

		desc, _ := cmd.Flags().GetString("description")
		from, _ := cmd.Flags().GetString("source")

		if config.ProfileExists(name) {
			return fmt.Errorf("profile %q already exists", name)
		}
		if from != "" && !config.ProfileExists(from) {
			return fmt.Errorf("source profile %q does not exist", from)
		}

		if err := config.EnsureProfile(name, desc); err != nil {
			return err
		}

		if from != "" {
			if err := copyProfileData(from, name); err != nil {
				return fmt.Errorf("copying profile data: %w", err)
			}
			fmt.Printf("created profile %q (copied from %q)\n", name, from)
		} else {
			fmt.Printf("created profile %q\n", name)
		}
		return nil
	},
}

// copyProfileData copies links and groups (not config) from src to dst profile.
func copyProfileData(src, dst string) error {
	return copyFilesBetweenDirs(config.ProfileDir(src), config.ProfileDir(dst),
		[]string{"links.yaml", "groups.yaml"})
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profiles, err := config.ListProfiles()
		if err != nil {
			return err
		}

		if len(profiles) == 0 {
			fmt.Println("no profiles found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION")
		fmt.Fprintln(w, "----\t-----------")
		for _, name := range profiles {
			p, err := config.LoadProfile(name)
			if err != nil {
				continue
			}
			active := ""
			if name == cfg.ActiveProfile {
				active = "  (active)"
			}
			fmt.Fprintf(w, "%s%s\t%s\n", name, active, p.Description)
		}
		return w.Flush()
	},
}

var profileViewCmd = &cobra.Command{
	Use:               "view [name]",
	Short:             "Show profile details",
	Long:              "Show details of a profile. Without a name, shows the currently active profile.",
	Example: `  $ zebro profile view           # show active profile
  $ zebro profile view work      # show specific profile
  $ zebro profile view -d        # show with full link/group lists
  $ zebro profile view -s        # show summary (overrides profile_view_mode=detail)`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		name := cfg.ActiveProfile
		if len(args) > 0 {
			name = args[0]
		}

		if !config.ProfileExists(name) {
			return fmt.Errorf("profile %q does not exist", name)
		}

		p, err := config.LoadProfile(name)
		if err != nil {
			return err
		}

		// detail mode: config default, overridden by explicit flags; -s wins over -d
		detail := cfg.ProfileViewMode == "detail"
		if cmd.Flags().Changed("detail") {
			detail = true
		}
		if cmd.Flags().Changed("summary") {
			detail = false
		}

		active := ""
		if name == cfg.ActiveProfile {
			active = " (active)"
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "name:\t%s%s\n", p.Name, active)
		if p.Description != "" {
			fmt.Fprintf(w, "description:\t%s\n", p.Description)
		}
		fmt.Fprintf(w, "dir:\t%s\n", config.ProfileDir(name))

		fmt.Fprintf(w, "config:\t\n")
		for _, k := range profileConfigKeys() {
			if k.profileOnly {
				continue
			}
			profileVal, _ := p.Get(k.key)
			if profileVal != "" {
				fmt.Fprintf(w, "  %s:\t%s\n", k.key, profileVal)
			} else {
				globalVal, _ := cfg.Get(k.key)
				if globalVal != "" {
					fmt.Fprintf(w, "  %s:\t%s  (global)\n", k.key, globalVal)
				}
			}
		}

		if detail {
			links, _ := store.ListLinks(config.ProfileLinksFile(name))
			groups, _ := store.ListGroups(config.ProfileGroupsFile(name))
			fmt.Fprintf(w, "links (%d):\t\n", len(links))
			for _, l := range links {
				fmt.Fprintf(w, "  %s:\t%s\n", store.DenormalizeVars(l.Key, cfg.VariablePrefix), store.DenormalizeVars(l.URL, cfg.VariablePrefix))
			}
			fmt.Fprintf(w, "groups (%d):\t\n", len(groups))
			for _, g := range groups {
				fmt.Fprintf(w, "  %s:\t\n", store.DenormalizeVars(g.Name, cfg.VariablePrefix))
				for _, u := range g.URLs {
					displayURL := store.DenormalizeParams(u, cfg.VariablePrefix, g.Params)
					fmt.Fprintf(w, "    - %s\t\n", displayURL)
				}
			}
		} else {
			links, _ := store.ListLinks(config.ProfileLinksFile(name))
			groups, _ := store.ListGroups(config.ProfileGroupsFile(name))
			fmt.Fprintf(w, "links:\t%d\n", len(links))
			fmt.Fprintf(w, "groups:\t%d\n", len(groups))
		}

		return w.Flush()
	},
}

func init() {
	profileViewCmd.Flags().BoolP("detail", "d", false, "Show full lists of links and groups")
	profileViewCmd.Flags().BoolP("summary", "s", false, "Show summary only (overrides profile_view_mode=detail)")
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active profile",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		profiles, err := config.ListProfiles()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		active, _ := config.GetActiveProfile()
		completions := make([]string, len(profiles))
		for i, p := range profiles {
			if p == active {
				completions[i] = p + "\t(*)"
			} else {
				completions[i] = p
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		if !config.ProfileExists(name) {
			return fmt.Errorf("profile %q does not exist (run: zebro profile create %s)", name, name)
		}
		prev, _ := config.GetActiveProfile()
		if err := config.SetActiveProfile(name); err != nil {
			return err
		}
		fmt.Printf("switched to profile %q (from %q)\n", name, prev)
		return nil
	},
}

var profileRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a profile",
	Long:  "Rename a profile. If it is the active profile, the active profile is updated automatically.\nExisting backups are also renamed to match the new profile name.",
	Example: `  $ zebro profile rename work work-old
  $ zebro profile rename personal home`,
	Args: cobra.MaximumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeProfileNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return cmd.Help()
		}
		oldName, newName := args[0], args[1]
		if err := validateProfileName(newName); err != nil {
			return err
		}
		if !config.ProfileExists(oldName) {
			return fmt.Errorf("profile %q does not exist", oldName)
		}
		if config.ProfileExists(newName) {
			return fmt.Errorf("profile %q already exists", newName)
		}

		// Move the profile directory
		if err := os.Rename(config.ProfileDir(oldName), config.ProfileDir(newName)); err != nil {
			return fmt.Errorf("renaming profile: %w", err)
		}

		// Update the name field in the profile config
		if p, err := config.LoadProfile(newName); err == nil {
			p.Name = newName
			_ = config.SaveProfile(newName, p)
		}

		// Rename backup directories to match the new profile name
		baks, _ := findBackupsFor(oldName)
		bakBaseDir := filepath.Join(config.ProfilesDir(), ".bak")
		for _, b := range baks {
			newBakPath := filepath.Join(bakBaseDir, newName+"."+b.Timestamp)
			_ = os.Rename(b.Path, newBakPath)
		}

		// If renamed profile was active, update active profile
		if active, _ := config.GetActiveProfile(); active == oldName {
			if err := config.SetActiveProfile(newName); err != nil {
				return fmt.Errorf("updating active profile: %w", err)
			}
			fmt.Printf("renamed profile %q to %q (active profile updated)\n", oldName, newName)
		} else {
			fmt.Printf("renamed profile %q to %q\n", oldName, newName)
		}
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:               "delete <name>",
	Short:             "Remove a profile",
	Long:              "Remove a profile. By default, follows the profile_delete_mode config (backup or permanent).\nUse --force to delete immediately, --backup to move to backup instead of deleting.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if name == cfg.ActiveProfile {
			return fmt.Errorf("cannot remove active profile %q (switch first with: zebro profile use <other>)", name)
		}
		if !config.ProfileExists(name) {
			return fmt.Errorf("profile %q does not exist", name)
		}

		force, _ := cmd.Flags().GetBool("force")
		backup, _ := cmd.Flags().GetBool("backup")
		purge, _ := cmd.Flags().GetBool("purge")

		if purge && (force || backup) {
			return fmt.Errorf("--purge cannot be used with --force or --backup")
		}
		if force && backup {
			return fmt.Errorf("--force and --backup are mutually exclusive")
		}

		if purge {
			if err := os.RemoveAll(config.ProfileDir(name)); err != nil {
				return fmt.Errorf("removing profile: %w", err)
			}
			baks, _ := findBackupsFor(name)
			for _, b := range baks {
				os.RemoveAll(b.Path)
			}
			fmt.Printf("purged profile %q and %d backup(s)\n", name, len(baks))
			return nil
		}

		doBackup := cfg.ProfileDeleteMode == "backup"
		if force {
			doBackup = false
		} else if backup {
			doBackup = true
		}

		dir := config.ProfileDir(name)
		if doBackup {
			ts := time.Now().Format("20060102-150405")
			bakDir := ensureUniqueBakDir(name, ts)
			if err := os.MkdirAll(filepath.Dir(bakDir), 0700); err != nil {
				return fmt.Errorf("creating backup dir: %w", err)
			}
			if err := os.Rename(dir, bakDir); err != nil {
				return fmt.Errorf("backing up profile: %w", err)
			}
			fmt.Printf("removed profile %q (backup: %s)\n", name, ts)
		} else {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("removing profile: %w", err)
			}
			fmt.Printf("removed profile %q\n", name)
		}
		return nil
	},
}

func init() {
	profileDeleteCmd.Flags().BoolP("force", "f", false, "Delete immediately without backup")
	profileDeleteCmd.Flags().BoolP("backup", "b", false, "Move to backup instead of deleting (recoverable)")
	profileDeleteCmd.Flags().Bool("purge", false, "Delete profile and all its backups permanently")
}

// completeProfileNames returns profile names for tab completion.
func completeProfileNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	profiles, err := config.ListProfiles()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return profiles, cobra.ShellCompDirectiveNoFileComp
}

var profileExportCmd = &cobra.Command{
	Use:   "export <name> [-o <file>]",
	Short: "Export a profile to a YAML file",
	Long:  "Export links, groups, and config of a profile to a YAML file (default: stdout).",
	Example: `  $ zebro profile export default
  $ zebro profile export work -o /tmp/work-profile.yaml`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		name := cfg.ActiveProfile
		if len(args) > 0 {
			name = args[0]
		}
		if !config.ProfileExists(name) {
			return fmt.Errorf("profile %q does not exist", name)
		}

		lf, err := store.LoadLinks(config.ProfileLinksFile(name))
		if err != nil {
			return err
		}
		gf, err := store.LoadGroups(config.ProfileGroupsFile(name))
		if err != nil {
			return err
		}
		pc, err := config.LoadProfile(name)
		if err != nil {
			return err
		}

		// Serialize ProfileConfig fields to map
		cfgMap := make(map[string]string)
		for _, k := range profileConfigKeys() {
			if v, err := pc.Get(k.key); err == nil && v != "" {
				cfgMap[k.key] = v
			}
		}

		ef := &store.ExportFile{
			Version: "1",
			Links:   lf.Links,
			Groups:  gf.Groups,
			Config:  cfgMap,
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
		fmt.Printf("exported profile %q (%d links, %d groups) to %s\n", name, len(lf.Links), len(gf.Groups), outPath)
		return nil
	},
}

var profileImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a profile from a YAML file",
	Long: `Import a profile from an export YAML file.

The profile name is taken from the file's config.name field.
Use --as to override the profile name.
Use --force to overwrite an existing profile.`,
	Example: `  $ zebro profile import /tmp/work-profile.yaml
  $ zebro profile import /tmp/work-profile.yaml --as work2
  $ zebro profile import /tmp/work-profile.yaml --force`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		ef, err := store.LoadExportFile(args[0])
		if err != nil {
			return err
		}

		asName, _ := cmd.Flags().GetString("as")
		force, _ := cmd.Flags().GetBool("force")

		// Determine profile name
		name := asName
		if name == "" {
			name = ef.Config["name"]
		}
		if name == "" {
			return fmt.Errorf("profile name not found in export file; use --as <name>")
		}
		if err := validateProfileName(name); err != nil {
			return err
		}

		if config.ProfileExists(name) && !force {
			return fmt.Errorf("profile %q already exists (use --force to overwrite)", name)
		}

		desc := ef.Config["description"]
		if err := config.EnsureProfile(name, desc); err != nil {
			return err
		}

		// Restore links
		if len(ef.Links) > 0 {
			lf, err := store.LoadLinks(config.ProfileLinksFile(name))
			if err != nil {
				return err
			}
			if force {
				lf.Links = ef.Links
			} else {
				for key, entry := range ef.Links {
					if _, exists := lf.Links[key]; !exists {
						lf.Links[key] = entry
					}
				}
			}
			if err := store.SaveLinks(config.ProfileLinksFile(name), lf); err != nil {
				return err
			}
		}

		// Restore groups
		if len(ef.Groups) > 0 {
			gf, err := store.LoadGroups(config.ProfileGroupsFile(name))
			if err != nil {
				return err
			}
			if force {
				gf.Groups = ef.Groups
			} else {
				for gname, entry := range ef.Groups {
					if _, exists := gf.Groups[gname]; !exists {
						gf.Groups[gname] = entry
					}
				}
			}
			if err := store.SaveGroups(config.ProfileGroupsFile(name), gf); err != nil {
				return err
			}
		}

		// Restore config
		if len(ef.Config) > 0 {
			pc, err := config.LoadProfile(name)
			if err != nil {
				return err
			}
			for key, val := range ef.Config {
				if key == "name" || key == "description" {
					continue
				}
				_ = pc.Set(key, val)
			}
			if err := config.SaveProfile(name, pc); err != nil {
				return err
			}
		}

		fmt.Printf("imported profile %q (%d links, %d groups)\n", name, len(ef.Links), len(ef.Groups))
		return nil
	},
}
