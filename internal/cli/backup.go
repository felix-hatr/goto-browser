package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type profileBackup struct {
	ProfileName string
	Timestamp   string
	Path        string
}

// parseBackupEntry parses a backup directory name into profile name and timestamp.
// Format: {profileName}.{YYYYMMDD-HHMMSS}
func parseBackupEntry(name string) (profileName, timestamp string, ok bool) {
	const tsLen = 15 // "20060102-150405"
	if len(name) <= tsLen+1 {
		return "", "", false
	}
	ts := name[len(name)-tsLen:]
	if ts[8] != '-' {
		return "", "", false
	}
	return name[:len(name)-tsLen-1], ts, true
}

// listAllBackups returns all backups sorted by profile name asc, timestamp desc.
func listAllBackups() ([]profileBackup, error) {
	bakDir := filepath.Join(config.ProfilesDir(), ".bak")
	entries, err := os.ReadDir(bakDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var backups []profileBackup
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		profileName, ts, ok := parseBackupEntry(e.Name())
		if !ok {
			continue
		}
		backups = append(backups, profileBackup{
			ProfileName: profileName,
			Timestamp:   ts,
			Path:        filepath.Join(bakDir, e.Name()),
		})
	}
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].ProfileName != backups[j].ProfileName {
			return backups[i].ProfileName < backups[j].ProfileName
		}
		return backups[i].Timestamp > backups[j].Timestamp // newest first
	})
	return backups, nil
}

// findBackupsFor returns backups for a specific profile, newest first.
func findBackupsFor(profileName string) ([]profileBackup, error) {
	all, err := listAllBackups()
	if err != nil {
		return nil, err
	}
	var result []profileBackup
	for _, b := range all {
		if b.ProfileName == profileName {
			result = append(result, b)
		}
	}
	return result, nil
}

// copyFilesBetweenDirs copies named files from srcDir to dstDir, skipping missing files.
func copyFilesBetweenDirs(srcDir, dstDir string, names []string) error {
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.WriteFile(filepath.Join(dstDir, name), data, 0600); err != nil {
			return err
		}
	}
	return nil
}

// copyProfileDataFromDir copies links, aliases, groups, and config from a backup dir to a profile.
func copyProfileDataFromDir(srcDir, dstProfile string) error {
	return copyFilesBetweenDirs(srcDir, config.ProfileDir(dstProfile),
		[]string{"links.yaml", "aliases.yaml", "groups.yaml", "config.yaml"})
}

// backupProfileToDir copies all profile files to a backup directory.
func backupProfileToDir(profileName, bakDir string) error {
	return copyFilesBetweenDirs(config.ProfileDir(profileName), bakDir,
		[]string{"links.yaml", "aliases.yaml", "groups.yaml", "config.yaml"})
}

// completeBackupProfileNames returns profile names that have backups, for tab completion.
// Unlike completeProfileNames, this includes deleted profiles that still have backups.
func completeBackupProfileNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	backups, err := listAllBackups()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	seen := map[string]bool{}
	var names []string
	for _, b := range backups {
		if !seen[b.ProfileName] {
			seen[b.ProfileName] = true
			names = append(names, b.ProfileName)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

var profileBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage profile backups",
	Long:  "List, view, create, restore, or delete profile backups.",
}

func init() {
	profileBackupCmd.AddCommand(
		profileBackupListCmd,
		profileBackupViewCmd,
		profileBackupCreateCmd,
		profileBackupRestoreCmd,
		profileBackupDeleteCmd,
	)
	profileBackupViewCmd.Flags().BoolP("detail", "d", false, "Show full lists of links, aliases, and groups")
	profileBackupRestoreCmd.Flags().String("from", "", "Backup timestamp to restore from (default: latest)")
	profileBackupRestoreCmd.Flags().String("as", "", "Restore as a different profile name")
	profileBackupRestoreCmd.Flags().BoolP("force", "f", false, "Overwrite if profile already exists")
	_ = profileBackupRestoreCmd.RegisterFlagCompletionFunc("from",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			baks, _ := findBackupsFor(args[0])
			ts := make([]string, len(baks))
			for i, b := range baks {
				ts[i] = b.Timestamp
			}
			return ts, cobra.ShellCompDirectiveNoFileComp
		})
}

var profileBackupListCmd = &cobra.Command{
	Use:   "list [name]",
	Short: "List backups",
	Long:  "List all backups, or backups for a specific profile.",
	Example: `  $ zebro profile backup list           # all backups
  $ zebro profile backup list work      # backups for 'work' profile`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeBackupProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		var backups []profileBackup
		var err error
		if len(args) > 0 {
			backups, err = findBackupsFor(args[0])
		} else {
			backups, err = listAllBackups()
		}
		if err != nil {
			return err
		}
		if len(backups) == 0 {
			fmt.Println("no backups found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if len(args) > 0 {
			// Profile-specific: show timestamp + path
			fmt.Fprintln(w, "TIMESTAMP\tPATH")
			fmt.Fprintln(w, "---------\t----")
			for i, b := range backups {
				suffix := ""
				if i == 0 {
					suffix = "  (latest)"
				}
				fmt.Fprintf(w, "%s%s\t%s\n", b.Timestamp, suffix, b.Path)
			}
		} else {
			// All backups: mark the newest entry per profile
			fmt.Fprintln(w, "PROFILE\tTIMESTAMP")
			fmt.Fprintln(w, "-------\t---------")
			lastProfile := ""
			for _, b := range backups {
				suffix := ""
				if b.ProfileName != lastProfile {
					suffix = "  (latest)"
					lastProfile = b.ProfileName
				}
				fmt.Fprintf(w, "%s\t%s%s\n", b.ProfileName, b.Timestamp, suffix)
			}
		}
		return w.Flush()
	},
}

var profileBackupViewCmd = &cobra.Command{
	Use:   "view <name> <ts>",
	Short: "Show contents of a backup",
	Long:  "Show the links, aliases, and groups stored in a specific backup.",
	Example: `  $ zebro profile backup view work 20260302-151524
  $ zebro profile backup view work 20260302-151524 -d`,
	Args: cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeBackupProfileNames(cmd, args, toComplete)
		}
		if len(args) == 1 {
			baks, _ := findBackupsFor(args[0])
			ts := make([]string, len(baks))
			for i, b := range baks {
				ts[i] = b.Timestamp
			}
			return ts, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name, ts := args[0], args[1]
		baks, err := findBackupsFor(name)
		if err != nil {
			return err
		}
		var bak *profileBackup
		for i := range baks {
			if baks[i].Timestamp == ts {
				bak = &baks[i]
				break
			}
		}
		if bak == nil {
			return fmt.Errorf("backup %q not found for profile %q", ts, name)
		}

		detail, _ := cmd.Flags().GetBool("detail")

		linksFile := filepath.Join(bak.Path, "links.yaml")
		aliasesFile := filepath.Join(bak.Path, "aliases.yaml")
		groupsFile := filepath.Join(bak.Path, "groups.yaml")

		links, _ := store.ListLinks(linksFile)
		// load aliases once; derive both the sorted slice and the lookup map
		aliasEntries, _ := store.ListAliases(aliasesFile)
		aliasesMap := make(map[string]string, len(aliasEntries))
		for _, a := range aliasEntries {
			aliasesMap[a.Name] = a.LinkKey
		}
		groups, _ := store.ListGroups(groupsFile)

		// Load variable prefix from backup's own config
		prefix := "@"
		var pc config.ProfileConfig
		if cfgData, err := os.ReadFile(filepath.Join(bak.Path, "config.yaml")); err == nil {
			if yaml.Unmarshal(cfgData, &pc) == nil && pc.VariablePrefix != "" {
				prefix = pc.VariablePrefix
			}
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "backup:\t%s (%s)\n", name, ts)
		fmt.Fprintf(w, "path:\t%s\n", bak.Path)

		if detail {
			fmt.Fprintf(w, "links (%d):\t\n", len(links))
			for _, l := range links {
				fmt.Fprintf(w, "  %s:\t%s\n", store.DenormalizeVars(l.Key, prefix), store.DenormalizeVars(l.URL, prefix))
			}
			fmt.Fprintf(w, "aliases (%d):\t\n", len(aliasEntries))
			for _, a := range aliasEntries {
				fmt.Fprintf(w, "  %s:\t%s\n", a.Name, a.LinkKey)
			}
			fmt.Fprintf(w, "groups (%d):\t\n", len(groups))
			for _, g := range groups {
				fmt.Fprintf(w, "  %s:\t\n", store.DenormalizeVars(g.Name, prefix))
				for _, ref := range g.Links {
					displayKey := store.DenormalizeVars(ref, prefix)
					url := resolveLinkURL(ref, links, aliasesMap, prefix)
					if url != "" {
						fmt.Fprintf(w, "    - %s:\t%s\n", displayKey, url)
					} else {
						fmt.Fprintf(w, "    - %s\t\n", displayKey)
					}
				}
			}
		} else {
			fmt.Fprintf(w, "links:\t%d\n", len(links))
			fmt.Fprintf(w, "aliases:\t%d\n", len(aliasEntries))
			fmt.Fprintf(w, "groups:\t%d\n", len(groups))
		}

		return w.Flush()
	},
}

var profileBackupCreateCmd = &cobra.Command{
	Use:               "create <name>",
	Short:             "Create a manual backup of a profile",
	Long:              "Create a backup snapshot of a profile without removing it.",
	Example:           `  $ zebro profile backup create work`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !config.ProfileExists(name) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		ts := time.Now().Format("20060102-150405")
		bakDir := filepath.Join(config.ProfilesDir(), ".bak", name+"."+ts)
		if err := os.MkdirAll(bakDir, 0700); err != nil {
			return fmt.Errorf("creating backup dir: %w", err)
		}
		if err := backupProfileToDir(name, bakDir); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
		fmt.Printf("created backup of profile %q: %s\n", name, ts)
		return nil
	},
}

var profileBackupRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore a profile from backup",
	Long:  "Restore a profile from its most recent backup. Use --from to pick a specific timestamp.",
	Example: `  $ zebro profile backup restore work
  $ zebro profile backup restore work --from 20260302-150405
  $ zebro profile backup restore work --as work2
  $ zebro profile backup restore work --force`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeBackupProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		as, _ := cmd.Flags().GetString("as")
		force, _ := cmd.Flags().GetBool("force")
		fromTS, _ := cmd.Flags().GetString("from")

		targetName := name
		if as != "" {
			targetName = as
		}

		baks, err := findBackupsFor(name)
		if err != nil {
			return err
		}
		if len(baks) == 0 {
			return fmt.Errorf("no backups found for profile %q", name)
		}

		var bak *profileBackup
		if fromTS != "" {
			for i, b := range baks {
				if b.Timestamp == fromTS {
					bak = &baks[i]
					break
				}
			}
			if bak == nil {
				return fmt.Errorf("backup %q not found for profile %q", fromTS, name)
			}
		} else {
			bak = &baks[0] // newest first
		}

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
				fmt.Printf("backed up existing profile %q as %s\n", targetName, ts)
			} else {
				if err := os.RemoveAll(config.ProfileDir(targetName)); err != nil {
					return fmt.Errorf("removing existing profile: %w", err)
				}
			}
		}

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

var profileBackupDeleteCmd = &cobra.Command{
	Use:     "delete <name> <ts>",
	Short:   "Delete a specific backup",
	Long:    "Permanently delete a single backup entry.",
	Example: `  $ zebro profile backup delete work 20260302-151524`,
	Args:    cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeBackupProfileNames(cmd, args, toComplete)
		}
		if len(args) == 1 {
			baks, _ := findBackupsFor(args[0])
			ts := make([]string, len(baks))
			for i, b := range baks {
				ts[i] = b.Timestamp
			}
			return ts, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name, ts := args[0], args[1]
		baks, err := findBackupsFor(name)
		if err != nil {
			return err
		}
		var bak *profileBackup
		for i := range baks {
			if baks[i].Timestamp == ts {
				bak = &baks[i]
				break
			}
		}
		if bak == nil {
			return fmt.Errorf("backup %q not found for profile %q", ts, name)
		}
		if err := os.RemoveAll(bak.Path); err != nil {
			return fmt.Errorf("deleting backup: %w", err)
		}
		fmt.Printf("deleted backup %s for profile %q\n", ts, name)
		return nil
	},
}
