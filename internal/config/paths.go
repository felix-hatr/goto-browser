package config

import (
	"os"
	"path/filepath"
)

// GbroDir returns the root zebro config directory (~/.config/zebro).
// Respects XDG_CONFIG_HOME if set.
func GbroDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zebro")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "zebro")
	}
	return filepath.Join(home, ".config", "zebro")
}

// ConfigFile returns the path to the global config file.
func ConfigFile() string {
	return filepath.Join(GbroDir(), "config.yaml")
}

// CurrentProfileFile returns the path to the active profile state file.
func CurrentProfileFile() string {
	return filepath.Join(GbroDir(), ".current_profile")
}

// ProfilesDir returns the directory containing all profiles.
func ProfilesDir() string {
	return filepath.Join(GbroDir(), "profiles")
}

// ProfileDir returns the directory for a specific profile.
func ProfileDir(name string) string {
	return filepath.Join(ProfilesDir(), name)
}

// ProfileConfigFile returns the config.yaml path for a profile.
func ProfileConfigFile(name string) string {
	return filepath.Join(ProfileDir(name), "config.yaml")
}

// ProfileLinksFile returns the links.yaml path for a profile.
func ProfileLinksFile(name string) string {
	return filepath.Join(ProfileDir(name), "links.yaml")
}

// ProfileAliasesFile returns the aliases.yaml path for a profile.
func ProfileAliasesFile(name string) string {
	return filepath.Join(ProfileDir(name), "aliases.yaml")
}

// ProfileGroupsFile returns the groups.yaml path for a profile.
func ProfileGroupsFile(name string) string {
	return filepath.Join(ProfileDir(name), "groups.yaml")
}
