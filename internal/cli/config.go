package cli

import (
	"fmt"
	"strings"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/spf13/cobra"
)

// configKeyDef describes a single config key — its name, help text, and valid values.
// Adding a new key here automatically updates help, tab completion, and list output.
type configKeyDef struct {
	key         string
	desc        string
	values      []string // nil = free-form input
	profileOnly bool
}

// sharedConfigKeys are available for both global and profile config.
// Profile values override the global value.
var sharedConfigKeys = []configKeyDef{
	{"browser", "Browser {chrome|brave|edge|arc|safari|whale} (default: chrome)", []string{"chrome", "brave", "edge", "arc", "safari", "whale"}, false},
	{"variable_prefix", "Variable prefix character (default: @)", nil, false},
	{"variable_display", "How variables are shown {named|positional} (default: named)", []string{"named", "positional"}, false},
	{"open_mode", "How to open links {new_tab|new_window} (default: new_tab)", []string{"new_tab", "new_window"}, false},
	{"open_default", "Default open target when no flag given {link|group|url} (default: link)", []string{"link", "group", "url"}, false},
	{"profile_delete_mode", "How to delete profiles {backup|permanent} (default: backup)", []string{"backup", "permanent"}, false},
	{"profile_view_mode", "Default view mode for profile view {summary|detail} (default: summary)", []string{"summary", "detail"}, false},
	{"description", "Profile description (profile only)", nil, true},
	{"history_size", "Max history entries to keep (default: 10000, -1=unlimited)", nil, false},
	{"history_dedup", "Dedup strategy {none|consecutive|all} (default: none)", []string{"none", "consecutive", "all"}, false},
}

func globalConfigKeys() []configKeyDef {
	var keys []configKeyDef
	for _, k := range sharedConfigKeys {
		if !k.profileOnly {
			keys = append(keys, k)
		}
	}
	return keys
}

func profileConfigKeys() []configKeyDef {
	return sharedConfigKeys
}

func keyNames(defs []configKeyDef) []string {
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.key
	}
	return names
}

func buildConfigLong() string {
	var sb strings.Builder
	sb.WriteString("Manage configuration for zebro.\n\n")
	sb.WriteString("By default, commands operate on the current profile's config.\n")
	sb.WriteString("Profile values override global values. Use -g for global config.\n\n")
	sb.WriteString("SETTINGS\n")
	for _, k := range sharedConfigKeys {
		fmt.Fprintf(&sb, "  - %-22s %s\n", k.key+":", k.desc)
	}
	return strings.TrimRight(sb.String(), "\n")
}

var configGlobalFlag bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  buildConfigLong(),
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
			return keyNames(globalConfigKeys()), cobra.ShellCompDirectiveNoFileComp
		}
		return keyNames(profileConfigKeys()), cobra.ShellCompDirectiveNoFileComp
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
		keys := profileConfigKeys()
		if configGlobalFlag {
			keys = globalConfigKeys()
		}
		if len(args) == 0 {
			return keyNames(keys), cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			for _, k := range keys {
				if k.key != args[0] || k.values == nil {
					continue
				}
				// Load current value to mark it with (*)
				var currentVal string
				if configGlobalFlag {
					if cfg, err := config.LoadGlobal(); err == nil {
						currentVal, _ = cfg.Get(k.key)
					}
				} else if cfg, err := config.Load(); err == nil {
					currentVal, _ = cfg.Get(k.key)
				}
				completions := make([]string, len(k.values))
				for i, v := range k.values {
					if v == currentVal {
						completions[i] = v + "\t(*)"
					} else {
						completions[i] = v
					}
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}
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
	for _, k := range globalConfigKeys() {
		val, _ := cfg.Get(k.key)
		fmt.Printf("%-20s %s\n", k.key, val)
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

	for _, k := range profileConfigKeys() {
		profileVal, _ := p.Get(k.key)
		if profileVal != "" {
			fmt.Printf("%-20s %s\n", k.key, profileVal)
		} else {
			globalVal, _ := cfg.Get(k.key)
			fmt.Printf("%-20s %-20s (global)\n", k.key, globalVal)
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
	// LoadGlobal reads raw values without profile overrides (for accurate old value)
	rawCfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	oldVal, _ := rawCfg.Get(key)

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
	if oldVal != "" && oldVal != value {
		fmt.Printf("set %s: %q → %q  (global)\n", key, oldVal, value)
	} else {
		fmt.Printf("set %s: %q  (global)\n", key, value)
	}
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
	oldVal, _ := p.Get(key)
	if err := p.Set(key, value); err != nil {
		return err
	}
	if err := config.SaveProfile(profile, p); err != nil {
		return err
	}
	if oldVal != "" && oldVal != value {
		fmt.Printf("set %s: %q → %q  (profile: %s)\n", key, oldVal, value, profile)
	} else {
		fmt.Printf("set %s: %q  (profile: %s)\n", key, value, profile)
	}
	return nil
}
