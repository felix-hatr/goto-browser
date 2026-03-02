package cli

import (
	"fmt"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global configuration",
}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd, configListCmd)
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"browser", "browser_profile", "variable_prefix", "open_mode"}, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		val, err := cfg.Get(args[0])
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"browser", "browser_profile", "variable_prefix", "open_mode"}, cobra.ShellCompDirectiveNoFileComp
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.Set(args[0], args[1]); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("set %s = %s\n", args[0], args[1])
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		activeProfile, _ := config.GetActiveProfile()
		fmt.Printf("%-20s %s\n", "active_profile", activeProfile)

		keys := []string{"browser", "browser_profile", "variable_prefix", "open_mode"}
		for _, k := range keys {
			val, _ := cfg.Get(k)
			fmt.Printf("%-20s %s\n", k, val)
		}
		return nil
	},
}
