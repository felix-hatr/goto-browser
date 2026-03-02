package cli

import (
	"fmt"
	"os"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "Manage profiles — isolated sets of links, aliases, and groups.",
}

func init() {
	profileCmd.AddCommand(profileListCmd, profileViewCmd, profileCreateCmd, profileDeleteCmd, profileUseCmd)
	profileCreateCmd.Flags().StringP("description", "d", "", "Profile description")
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")

		if config.ProfileExists(name) {
			return fmt.Errorf("profile %q already exists", name)
		}

		if err := config.EnsureProfile(name, desc); err != nil {
			return err
		}
		fmt.Printf("created profile %q\n", name)
		return nil
	},
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

var profileViewCmd = &cobra.Command{
	Use:               "view [name]",
	Short:             "Show profile details",
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

		fmt.Printf("name:        %s\n", p.Name)
		fmt.Printf("description: %s\n", p.Description)
		if p.Browser != "" {
			fmt.Printf("browser:     %s\n", p.Browser)
		}
		active := ""
		if name == cfg.ActiveProfile {
			active = " (active)"
		}
		fmt.Printf("dir:         %s%s\n", config.ProfileDir(name), active)
		return nil
	},
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
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProfileNames,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		if err := os.RemoveAll(config.ProfileDir(name)); err != nil {
			return fmt.Errorf("removing profile: %w", err)
		}
		fmt.Printf("removed profile %q\n", name)
		return nil
	},
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
