package cli

import (
	"fmt"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/spf13/cobra"
)

var configGlobalFlag bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage configuration for zebro.

By default, commands operate on the current profile's config.
Use -g to read or write global config instead.

PROFILE SETTINGS
  - description:      Profile description
  - browser:          Browser to use for this profile {chrome|brave|edge|arc|safari|whale}
  - variable_prefix:  Variable prefix character for this profile (e.g. @, :, ^)
  - open_mode:        How to open links for this profile {new_tab|new_window}

GLOBAL SETTINGS  (zebro config <cmd> -g)
  - browser:          Default browser {chrome|brave|edge|arc|safari|whale} (default: chrome)
  - variable_prefix:  Default variable prefix character (default: @)
  - open_mode:        Default open mode {new_tab|new_window} (default: new_tab)`,
}

func init() {
	configCmd.PersistentFlags().BoolVarP(&configGlobalFlag, "global", "g", false, "Operate on global config instead of current profile")
	configCmd.AddCommand(configGetCmd, configSetCmd, configListCmd)
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List config values",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configGlobalFlag {
			return runConfigListGlobal()
		}
		return runConfigListProfile(cmd)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if configGlobalFlag {
			return []string{"browser", "variable_prefix", "open_mode"}, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{"description", "browser", "variable_prefix", "open_mode"}, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if configGlobalFlag {
			return runConfigGetGlobal(args[0])
		}
		return runConfigGetProfile(cmd, args[0])
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if configGlobalFlag {
			if len(args) == 0 {
				return []string{"browser", "variable_prefix", "open_mode"}, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 && args[0] == "browser" {
				return []string{"chrome", "brave", "edge", "arc", "safari", "whale"}, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 && args[0] == "open_mode" {
				return []string{"new_tab", "new_window"}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 0 {
			return []string{"description", "browser", "variable_prefix", "open_mode"}, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 && args[0] == "browser" {
			return []string{"chrome", "brave", "edge", "arc", "safari", "whale"}, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 && args[0] == "open_mode" {
			return []string{"new_tab", "new_window"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if configGlobalFlag {
			return runConfigSetGlobal(args[0], args[1])
		}
		return runConfigSetProfile(cmd, args[0], args[1])
	},
}

func runConfigListGlobal() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	activeProfile, _ := config.GetActiveProfile()
	fmt.Printf("%-20s %s\n", "active_profile", activeProfile)
	keys := []string{"browser", "variable_prefix", "open_mode"}
	for _, k := range keys {
		val, _ := cfg.Get(k)
		fmt.Printf("%-20s %s\n", k, val)
	}
	return nil
}

func runConfigListProfile(cmd *cobra.Command) error {
	profile, cfg, err := currentProfile()
	if err != nil {
		return err
	}
	p, err := config.LoadProfile(profile)
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %s\n", "profile", profile)

	keys := []string{"description", "browser", "variable_prefix", "open_mode"}
	for _, k := range keys {
		profileVal, _ := p.Get(k)
		if profileVal != "" {
			fmt.Printf("%-20s %s\n", k, profileVal)
		} else {
			globalVal, _ := cfg.Get(k)
			fmt.Printf("%-20s %-20s (global)\n", k, globalVal)
		}
	}
	return nil
}

func runConfigGetGlobal(key string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	val, err := cfg.Get(key)
	if err != nil {
		return err
	}
	fmt.Println(val)
	return nil
}

func runConfigGetProfile(cmd *cobra.Command, key string) error {
	profile, cfg, err := currentProfile()
	if err != nil {
		return err
	}
	p, err := config.LoadProfile(profile)
	if err != nil {
		return err
	}
	val, err := p.Get(key)
	if err != nil {
		return err
	}
	if val != "" {
		fmt.Println(val)
		return nil
	}
	// fall back to global
	globalVal, _ := cfg.Get(key)
	fmt.Printf("%s  (global)\n", globalVal)
	return nil
}

func runConfigSetGlobal(key, value string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.Set(key, value); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("set %s = %s  (global)\n", key, value)
	return nil
}

func runConfigSetProfile(cmd *cobra.Command, key, value string) error {
	profile, _, err := currentProfile()
	if err != nil {
		return err
	}
	p, err := config.LoadProfile(profile)
	if err != nil {
		return err
	}
	if err := p.Set(key, value); err != nil {
		return err
	}
	if err := config.SaveProfile(profile, p); err != nil {
		return err
	}
	fmt.Printf("set %s = %s  (profile: %s)\n", key, value, profile)
	return nil
}
