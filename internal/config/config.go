package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

const defaultVariablePrefix = "@"

// GlobalConfig holds the global zebro configuration.
// active_profile is intentionally excluded — stored separately in .current_profile.
type GlobalConfig struct {
	Version           string `yaml:"version"`
	Browser           string `yaml:"browser"`
	VariablePrefix    string `yaml:"variable_prefix"`
	VariableDisplay   string `yaml:"variable_display"`
	OpenMode          string `yaml:"open_mode"`
	OpenDefault       string `yaml:"open_default"`
	ProfileDeleteMode string `yaml:"profile_delete_mode"`
	ProfileViewMode   string `yaml:"profile_view_mode"`
	HistorySize  int    `yaml:"history_size,omitempty"`
	HistoryDedup string `yaml:"history_dedup,omitempty"`

	// Runtime-only: loaded from .current_profile, not written to config.yaml.
	ActiveProfile string `yaml:"-"`
}

// ProfileConfig holds per-profile configuration.
// Non-empty fields override the corresponding global config values.
type ProfileConfig struct {
	Name              string `yaml:"name"`
	Description       string `yaml:"description,omitempty"`
	Browser           string `yaml:"browser,omitempty"`
	VariablePrefix    string `yaml:"variable_prefix,omitempty"`
	VariableDisplay   string `yaml:"variable_display,omitempty"`
	OpenMode          string `yaml:"open_mode,omitempty"`
	OpenDefault       string `yaml:"open_default,omitempty"`
	ProfileDeleteMode string `yaml:"profile_delete_mode,omitempty"`
	ProfileViewMode   string `yaml:"profile_view_mode,omitempty"`
	HistorySize  int    `yaml:"history_size,omitempty"`
	HistoryDedup string `yaml:"history_dedup,omitempty"`
}

// applyConfigDefaults fills in zero-value fields with their defaults.
func applyConfigDefaults(cfg *GlobalConfig) {
	if cfg.VariablePrefix == "" {
		cfg.VariablePrefix = defaultVariablePrefix
	}
	if cfg.VariableDisplay == "" {
		cfg.VariableDisplay = "named"
	}
	if cfg.OpenMode == "" {
		cfg.OpenMode = "new_tab"
	}
	if cfg.OpenDefault == "" {
		cfg.OpenDefault = "link"
	}
	if cfg.ProfileDeleteMode == "" {
		cfg.ProfileDeleteMode = "backup"
	}
	if cfg.ProfileViewMode == "" {
		cfg.ProfileViewMode = "summary"
	}
}

// LoadGlobal loads the raw global config without applying profile overrides.
// Use this when you need the actual stored global value, not the effective (profile-merged) value.
func LoadGlobal() (*GlobalConfig, error) {
	cfgPath := ConfigFile()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	applyConfigDefaults(&cfg)
	return &cfg, nil
}

// Load loads the global config, auto-initializing if not present.
func Load() (*GlobalConfig, error) {
	cfgPath := ConfigFile()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return autoInit()
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply defaults
	applyConfigDefaults(&cfg)

	// Load active profile from .current_profile
	cfg.ActiveProfile, err = GetActiveProfile()
	if err != nil {
		cfg.ActiveProfile = "default"
	}

	// Apply profile-level overrides onto global config
	if pc, err := LoadProfile(cfg.ActiveProfile); err == nil {
		applyProfileOverrides(&cfg, pc)
	}

	return &cfg, nil
}

// applyProfileOverrides overlays non-empty profile config values onto global config.
func applyProfileOverrides(global *GlobalConfig, profile *ProfileConfig) {
	if profile.Browser != "" {
		global.Browser = profile.Browser
	}
	if profile.VariablePrefix != "" {
		global.VariablePrefix = profile.VariablePrefix
	}
	if profile.VariableDisplay != "" {
		global.VariableDisplay = profile.VariableDisplay
	}
	if profile.OpenMode != "" {
		global.OpenMode = profile.OpenMode
	}
	if profile.OpenDefault != "" {
		global.OpenDefault = profile.OpenDefault
	}
	if profile.ProfileDeleteMode != "" {
		global.ProfileDeleteMode = profile.ProfileDeleteMode
	}
	if profile.ProfileViewMode != "" {
		global.ProfileViewMode = profile.ProfileViewMode
	}
	if profile.HistorySize != 0 {
		global.HistorySize = profile.HistorySize
	}
	if profile.HistoryDedup != "" {
		global.HistoryDedup = profile.HistoryDedup
	}
}

// Save writes the global config to disk (excludes ActiveProfile).
func Save(cfg *GlobalConfig) error {
	if err := os.MkdirAll(ZebroDir(), 0700); err != nil {
		return fmt.Errorf("creating zebro dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	return os.WriteFile(ConfigFile(), data, 0600)
}

// GetActiveProfile reads the active profile name from .current_profile.
func GetActiveProfile() (string, error) {
	data, err := os.ReadFile(CurrentProfileFile())
	if err != nil {
		if os.IsNotExist(err) {
			return "default", nil
		}
		return "", err
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "default", nil
	}
	return name, nil
}

// SetActiveProfile writes the active profile name to .current_profile.
func SetActiveProfile(name string) error {
	if err := os.MkdirAll(ZebroDir(), 0700); err != nil {
		return err
	}
	return os.WriteFile(CurrentProfileFile(), []byte(name+"\n"), 0600)
}

// autoInit creates the default config and profile on first run.
func autoInit() (*GlobalConfig, error) {
	fmt.Fprintln(os.Stderr, "zebro: no config found. initializing...")

	cfg := &GlobalConfig{
		Version:       "1",
		Browser:       "chrome",
		ActiveProfile: "default",
	}
	applyConfigDefaults(cfg)

	if err := Save(cfg); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "  created %s\n", ConfigFile())

	if err := SetActiveProfile("default"); err != nil {
		return nil, err
	}

	if err := EnsureProfile("default", "Default profile"); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "  created profile \"default\"\n")
	fmt.Fprintf(os.Stderr, "  active profile set to \"default\"\n")

	return cfg, nil
}

// EnsureProfile creates a profile directory and files if they don't exist.
func EnsureProfile(name, description string) error {
	dir := ProfileDir(name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating profile dir: %w", err)
	}

	profileCfg := ProfileConfig{
		Name:        name,
		Description: description,
	}

	cfgPath := ProfileConfigFile(name)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := SaveProfile(name, &profileCfg); err != nil {
			return err
		}
	}

	for path, content := range map[string]string{
		ProfileLinksFile(name):  "version: \"1\"\nlinks: {}\n",
		ProfileGroupsFile(name): "version: \"1\"\ngroups: {}\n",
	} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := writeEmptyFile(path, content); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeEmptyFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600)
}

// LoadProfile loads a profile's config.
func LoadProfile(name string) (*ProfileConfig, error) {
	data, err := os.ReadFile(ProfileConfigFile(name))
	if err != nil {
		return nil, fmt.Errorf("reading profile config: %w", err)
	}

	var cfg ProfileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing profile config: %w", err)
	}
	return &cfg, nil
}

// SaveProfile writes a profile config to disk.
func SaveProfile(name string, cfg *ProfileConfig) error {
	if err := os.MkdirAll(ProfileDir(name), 0700); err != nil {
		return fmt.Errorf("creating profile dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing profile config: %w", err)
	}

	return os.WriteFile(ProfileConfigFile(name), data, 0600)
}

// ListProfiles returns all profile names.
func ListProfiles() ([]string, error) {
	entries, err := os.ReadDir(ProfilesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing profiles: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}


// ProfileExists checks if a profile exists.
func ProfileExists(name string) bool {
	_, err := os.Stat(ProfileConfigFile(name))
	return err == nil
}

// Get returns a profile config value by key.
func (c *ProfileConfig) Get(key string) (string, error) {
	switch key {
	case "description":
		return c.Description, nil
	case "browser":
		return c.Browser, nil
	case "variable_prefix":
		return c.VariablePrefix, nil
	case "variable_display":
		return c.VariableDisplay, nil
	case "open_mode":
		return c.OpenMode, nil
	case "open_default":
		return c.OpenDefault, nil
	case "profile_delete_mode":
		return c.ProfileDeleteMode, nil
	case "profile_view_mode":
		return c.ProfileViewMode, nil
	case "history_size":
		if c.HistorySize == 0 {
			return "", nil
		}
		return fmt.Sprintf("%d", c.HistorySize), nil
	case "history_dedup":
		return c.HistoryDedup, nil
	default:
		return "", fmt.Errorf("unknown profile config key: %q (valid keys: description, browser, variable_prefix, variable_display, open_mode, open_default, profile_delete_mode, profile_view_mode, history_size, history_dedup)", key)
	}
}

// Set updates a profile config value by key.
func (c *ProfileConfig) Set(key, value string) error {
	switch key {
	case "description":
		c.Description = value
	case "browser":
		c.Browser = value
	case "variable_prefix":
		if err := validateVariablePrefix(value); err != nil {
			return err
		}
		c.VariablePrefix = value
	case "variable_display":
		if value != "named" && value != "positional" {
			return fmt.Errorf("variable_display must be 'named' or 'positional'")
		}
		c.VariableDisplay = value
	case "open_mode":
		if value != "new_tab" && value != "new_window" {
			return fmt.Errorf("open_mode must be 'new_tab' or 'new_window'")
		}
		c.OpenMode = value
	case "open_default":
		if value != "link" && value != "group" && value != "url" {
			return fmt.Errorf("open_default must be 'link', 'group', or 'url'")
		}
		c.OpenDefault = value
	case "profile_delete_mode":
		if value != "backup" && value != "permanent" {
			return fmt.Errorf("profile_delete_mode must be 'backup' or 'permanent'")
		}
		c.ProfileDeleteMode = value
	case "profile_view_mode":
		if value != "summary" && value != "detail" {
			return fmt.Errorf("profile_view_mode must be 'summary' or 'detail'")
		}
		c.ProfileViewMode = value
	case "history_size":
		n, err := parseHistorySize(value)
		if err != nil {
			return err
		}
		c.HistorySize = n
	case "history_dedup":
		if value != "none" && value != "consecutive" && value != "all" {
			return fmt.Errorf("history_dedup must be 'none', 'consecutive', or 'all'")
		}
		c.HistoryDedup = value
	default:
		return fmt.Errorf("unknown profile config key: %q (valid keys: description, browser, variable_prefix, variable_display, open_mode, open_default, profile_delete_mode, profile_view_mode, history_size, history_dedup)", key)
	}
	return nil
}

// Get returns a global config value by key.
func (c *GlobalConfig) Get(key string) (string, error) {
	switch key {
	case "browser":
		return c.Browser, nil
	case "variable_prefix":
		return c.VariablePrefix, nil
	case "variable_display":
		return c.VariableDisplay, nil
	case "open_mode":
		return c.OpenMode, nil
	case "open_default":
		return c.OpenDefault, nil
	case "profile_delete_mode":
		return c.ProfileDeleteMode, nil
	case "profile_view_mode":
		return c.ProfileViewMode, nil
	case "history_size":
		if c.HistorySize == 0 {
			return "", nil
		}
		return fmt.Sprintf("%d", c.HistorySize), nil
	case "history_dedup":
		return c.HistoryDedup, nil
	default:
		return "", fmt.Errorf("unknown config key: %q (valid keys: browser, variable_prefix, variable_display, open_mode, open_default, profile_delete_mode, profile_view_mode, history_size, history_dedup)", key)
	}
}

// Set updates a config value by key.
func (c *GlobalConfig) Set(key, value string) error {
	switch key {
	case "browser":
		c.Browser = value
	case "variable_prefix":
		if err := validateVariablePrefix(value); err != nil {
			return err
		}
		c.VariablePrefix = value
	case "variable_display":
		if value != "named" && value != "positional" {
			return fmt.Errorf("variable_display must be 'named' or 'positional'")
		}
		c.VariableDisplay = value
	case "open_mode":
		if value != "new_tab" && value != "new_window" {
			return fmt.Errorf("open_mode must be 'new_tab' or 'new_window'")
		}
		c.OpenMode = value
	case "open_default":
		if value != "link" && value != "group" && value != "url" {
			return fmt.Errorf("open_default must be 'link', 'group', or 'url'")
		}
		c.OpenDefault = value
	case "profile_delete_mode":
		if value != "backup" && value != "permanent" {
			return fmt.Errorf("profile_delete_mode must be 'backup' or 'permanent'")
		}
		c.ProfileDeleteMode = value
	case "profile_view_mode":
		if value != "summary" && value != "detail" {
			return fmt.Errorf("profile_view_mode must be 'summary' or 'detail'")
		}
		c.ProfileViewMode = value
	case "history_size":
		n, err := parseHistorySize(value)
		if err != nil {
			return err
		}
		c.HistorySize = n
	case "history_dedup":
		if value != "none" && value != "consecutive" && value != "all" {
			return fmt.Errorf("history_dedup must be 'none', 'consecutive', or 'all'")
		}
		c.HistoryDedup = value
	default:
		return fmt.Errorf("unknown config key: %q (valid keys: browser, variable_prefix, variable_display, open_mode, open_default, profile_delete_mode, profile_view_mode, history_size, history_dedup)", key)
	}
	return nil
}

// parseHistorySize parses a history_size config value (must be integer > 0).
func parseHistorySize(value string) (int, error) {
	n := 0
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return 0, fmt.Errorf("history_size must be a positive integer (got %q)", value)
	}
	if n <= 0 {
		return 0, fmt.Errorf("history_size must be a positive integer (got %d)", n)
	}
	return n, nil
}

// validateVariablePrefix checks that the prefix is a single, safe character.
// Blocked: letters, digits, path separator, shell metacharacters, YAML special chars, URL%, quotes, whitespace.
func validateVariablePrefix(value string) error {
	const hint = "common choices: @ ^ : ~ + = , . ; - _"
	runes := []rune(value)
	if len(runes) != 1 {
		return fmt.Errorf("variable_prefix must be a single character (got %q)\n%s", value, hint)
	}
	c := runes[0]
	if unicode.IsLetter(c) || unicode.IsDigit(c) {
		return fmt.Errorf("variable_prefix cannot be a letter or digit\n%s", hint)
	}
	const blocked = `/$!` + "`" + `\*?#{}[]&|><% "'()`
	if strings.ContainsRune(blocked, c) {
		return fmt.Errorf("variable_prefix %q is not allowed (conflicts with shell, URL, or YAML syntax)\n%s", value, hint)
	}
	return nil
}
