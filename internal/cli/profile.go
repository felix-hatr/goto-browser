package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "Manage profiles — isolated sets of links, aliases, and groups.",
	Example: `  $ zebro profile view          # show active profile
  $ zebro profile list          # list all profiles
  $ zebro profile create work   # create a new profile
  $ zebro profile use work      # switch active profile`,
}

func init() {
	profileCmd.AddCommand(profileListCmd, profileViewCmd, profileCreateCmd, profileDeleteCmd, profileUseCmd, profileRestoreCmd)
	profileCreateCmd.Flags().StringP("description", "d", "", "Profile description")
	profileCreateCmd.Flags().StringP("source", "s", "", "Copy links, aliases, and groups from an existing profile")
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

// copyProfileData copies links, aliases, and groups from src to dst profile.
func copyProfileData(src, dst string) error {
	files := []func(string) string{
		config.ProfileLinksFile,
		config.ProfileAliasesFile,
		config.ProfileGroupsFile,
	}
	for _, fn := range files {
		data, err := os.ReadFile(fn(src))
		if err != nil {
			return err
		}
		if err := os.WriteFile(fn(dst), data, 0600); err != nil {
			return err
		}
	}
	return nil
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		showBackups, _ := cmd.Flags().GetBool("backups")
		if showBackups {
			backups, err := listAllBackups()
			if err != nil {
				return err
			}
			if len(backups) == 0 {
				fmt.Println("no backups found")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROFILE\tTIMESTAMP")
			fmt.Fprintln(w, "-------\t---------")
			for _, b := range backups {
				fmt.Fprintf(w, "%s\t%s\n", b.ProfileName, b.Timestamp)
			}
			return w.Flush()
		}

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

		fmt.Printf("%-20s %s\n", "NAME", "DESCRIPTION")
		fmt.Printf("%-20s %s\n", "----", "-----------")
		for _, name := range profiles {
			p, err := config.LoadProfile(name)
			if err != nil {
				continue
			}
			active := ""
			if name == cfg.ActiveProfile {
				active = " *"
			}
			fmt.Printf("%-20s %s%s\n", name, p.Description, active)
		}
		return nil
	},
}

func init() {
	profileListCmd.Flags().Bool("backups", false, "List backups instead of profiles")
}

var profileViewCmd = &cobra.Command{
	Use:               "view [name]",
	Short:             "Show profile details",
	Long:              "Show details of a profile. Without a name, shows the currently active profile.",
	Example: `  $ zebro profile view           # show active profile
  $ zebro profile view work      # show specific profile
  $ zebro profile view -d        # show with full link/alias/group lists
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

		detailFlag := cmd.Flags().Changed("detail")
		summaryFlag := cmd.Flags().Changed("summary")
		detail, _ := cmd.Flags().GetBool("detail")
		if summaryFlag {
			detail = false
		} else if !detailFlag {
			detail = cfg.ProfileViewMode == "detail"
		}

		active := ""
		if name == cfg.ActiveProfile {
			active = " (active)"
		}

		links, _ := store.ListLinks(config.ProfileLinksFile(name))
		aliases, _ := store.ListAliases(config.ProfileAliasesFile(name))
		groups, _ := store.ListGroups(config.ProfileGroupsFile(name))
		af, _ := store.LoadAliases(config.ProfileAliasesFile(name))
		aliasesMap := map[string]string{}
		if af != nil {
			aliasesMap = af.Aliases
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "name:\t%s%s\n", p.Name, active)

		// description (profile-only key)
		if p.Description != "" {
			fmt.Fprintf(w, "description:\t%s\n", p.Description)
		}
		fmt.Fprintf(w, "dir:\t%s\n", config.ProfileDir(name))

		// config section
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

		// links / aliases / groups
		if detail {
			fmt.Fprintf(w, "links (%d):\t\n", len(links))
			for _, l := range links {
				fmt.Fprintf(w, "  %s:\t%s\n", store.DenormalizeVars(l.Key, cfg.VariablePrefix), store.DenormalizeVars(l.URL, cfg.VariablePrefix))
			}
			fmt.Fprintf(w, "aliases (%d):\t\n", len(aliases))
			for _, a := range aliases {
				fmt.Fprintf(w, "  %s:\t%s\n", a.Name, a.LinkKey)
			}
			fmt.Fprintf(w, "groups (%d):\t\n", len(groups))
			for _, g := range groups {
				fmt.Fprintf(w, "  %s:\t\n", store.DenormalizeVars(g.Name, cfg.VariablePrefix))
				for _, ref := range g.Links {
					displayKey := store.DenormalizeVars(ref, cfg.VariablePrefix)
					url := resolveLinkURL(ref, links, aliasesMap, cfg.VariablePrefix)
					if url != "" {
						fmt.Fprintf(w, "    - %s:\t%s\n", displayKey, url)
					} else {
						fmt.Fprintf(w, "    - %s\t\n", displayKey)
					}
				}
			}
		} else {
			fmt.Fprintf(w, "links:\t%d\n", len(links))
			fmt.Fprintf(w, "aliases:\t%d\n", len(aliases))
			fmt.Fprintf(w, "groups:\t%d\n", len(groups))
		}

		// backups section
		showBackups, _ := cmd.Flags().GetBool("backups")
		if showBackups {
			baks, _ := findBackupsFor(name)
			fmt.Fprintf(w, "backups (%d):\t\n", len(baks))
			for _, b := range baks {
				fmt.Fprintf(w, "  %s:\t%s\n", b.Timestamp, b.Path)
			}
		}

		return w.Flush()
	},
}

func init() {
	profileViewCmd.Flags().BoolP("detail", "d", false, "Show full lists of links, aliases, and groups")
	profileViewCmd.Flags().BoolP("summary", "s", false, "Show summary only (overrides profile_view_mode=detail)")
	profileViewCmd.Flags().Bool("backups", false, "Show backup list for the profile")
}

var profileUseCmd = &cobra.Command{
	Use:               "use <name>",
	Short:             "Switch the active profile",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !config.ProfileExists(name) {
			return fmt.Errorf("profile %q does not exist (run: zebro profile create %s)", name, name)
		}

		if err := config.SetActiveProfile(name); err != nil {
			return err
		}

		fmt.Printf("switched to profile %q\n", name)
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

		if force && backup {
			return fmt.Errorf("--force and --backup are mutually exclusive")
		}

		if purge {
			// Delete profile permanently + all its backups
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
			bakDir := filepath.Join(config.ProfilesDir(), ".bak", name+"."+ts)
			if err := os.MkdirAll(filepath.Dir(bakDir), 0700); err != nil {
				return fmt.Errorf("creating backup dir: %w", err)
			}
			if err := os.Rename(dir, bakDir); err != nil {
				return fmt.Errorf("backing up profile: %w", err)
			}
			fmt.Printf("removed profile %q (backup: %s)\n", name, bakDir)
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

var profileRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore a profile from backup",
	Long:  "Restore a profile from its most recent backup. Use --backup to pick a specific timestamp.",
	Example: `  $ zebro profile restore work                      # restore latest backup
  $ zebro profile restore work --backup 20260302-150405
  $ zebro profile restore work --as work2            # restore as a new name
  $ zebro profile restore work --force               # overwrite existing profile`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		as, _ := cmd.Flags().GetString("as")
		force, _ := cmd.Flags().GetBool("force")
		backupTS, _ := cmd.Flags().GetString("backup")

		targetName := name
		if as != "" {
			targetName = as
		}

		// Find the backup to restore from
		baks, err := findBackupsFor(name)
		if err != nil {
			return err
		}
		if len(baks) == 0 {
			return fmt.Errorf("no backups found for profile %q", name)
		}

		var bak *profileBackup
		if backupTS != "" {
			for i, b := range baks {
				if b.Timestamp == backupTS {
					bak = &baks[i]
					break
				}
			}
			if bak == nil {
				return fmt.Errorf("backup %q not found for profile %q", backupTS, name)
			}
		} else {
			bak = &baks[0] // newest first
		}

		// Handle conflict
		if config.ProfileExists(targetName) {
			if !force {
				return fmt.Errorf("profile %q already exists\n  use --as <name> to restore as a different name\n  use --force to overwrite", targetName)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.ProfileDeleteMode == "backup" {
				ts := time.Now().Format("20060102-150405")
				bakDir := filepath.Join(config.ProfilesDir(), ".bak", targetName+"."+ts)
				if err := os.MkdirAll(filepath.Dir(bakDir), 0700); err != nil {
					return fmt.Errorf("creating backup dir: %w", err)
				}
				if err := os.Rename(config.ProfileDir(targetName), bakDir); err != nil {
					return fmt.Errorf("backing up existing profile: %w", err)
				}
			} else {
				if err := os.RemoveAll(config.ProfileDir(targetName)); err != nil {
					return fmt.Errorf("removing existing profile: %w", err)
				}
			}
		}

		// Create target profile dir and copy data
		if err := config.EnsureProfile(targetName, ""); err != nil {
			return err
		}
		if err := copyProfileDataFromDir(bak.Path, targetName); err != nil {
			return fmt.Errorf("restoring profile data: %w", err)
		}

		fmt.Printf("restored profile %q from backup %s\n", targetName, bak.Timestamp)
		return nil
	},
}

func init() {
	profileRestoreCmd.Flags().String("as", "", "Restore as a different profile name")
	profileRestoreCmd.Flags().BoolP("force", "f", false, "Overwrite existing profile")
	profileRestoreCmd.Flags().String("backup", "", "Specific backup timestamp to restore (default: latest)")
}

// copyProfileDataFromDir copies links, aliases, groups, and config from a backup dir.
func copyProfileDataFromDir(srcDir, dstProfile string) error {
	files := map[string]string{
		"links.yaml":   config.ProfileLinksFile(dstProfile),
		"aliases.yaml": config.ProfileAliasesFile(dstProfile),
		"groups.yaml":  config.ProfileGroupsFile(dstProfile),
		"config.yaml":  config.ProfileConfigFile(dstProfile),
	}
	for srcFile, dst := range files {
		data, err := os.ReadFile(filepath.Join(srcDir, srcFile))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			return err
		}
	}
	return nil
}
