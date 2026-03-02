package store

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// AliasFile is the on-disk format for aliases.yaml.
type AliasFile struct {
	Version string            `yaml:"version"`
	Aliases map[string]string `yaml:"aliases"`
}

// LoadAliases reads the aliases file for a profile.
func LoadAliases(path string) (*AliasFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AliasFile{Version: "1", Aliases: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("reading aliases: %w", err)
	}

	var af AliasFile
	if err := yaml.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("parsing aliases: %w", err)
	}
	if af.Aliases == nil {
		af.Aliases = map[string]string{}
	}
	return &af, nil
}

// SaveAliases writes the aliases file.
func SaveAliases(path string, af *AliasFile) error {
	data, err := yaml.Marshal(af)
	if err != nil {
		return fmt.Errorf("serializing aliases: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// AddAlias adds or updates an alias.
func AddAlias(path, name, linkKey string) error {
	af, err := LoadAliases(path)
	if err != nil {
		return err
	}
	af.Aliases[name] = linkKey
	return SaveAliases(path, af)
}

// GetAlias retrieves an alias value.
func GetAlias(path, name string) (string, error) {
	af, err := LoadAliases(path)
	if err != nil {
		return "", err
	}
	val, ok := af.Aliases[name]
	if !ok {
		return "", fmt.Errorf("alias %q not found", name)
	}
	return val, nil
}

// RemoveAlias deletes an alias.
func RemoveAlias(path, name string) error {
	af, err := LoadAliases(path)
	if err != nil {
		return err
	}
	if _, ok := af.Aliases[name]; !ok {
		return fmt.Errorf("alias %q not found", name)
	}
	delete(af.Aliases, name)
	return SaveAliases(path, af)
}

// AliasEntry is a sorted alias entry for display.
type AliasEntry struct {
	Name    string
	LinkKey string
}

// ListAliases returns all aliases sorted by name.
func ListAliases(path string) ([]AliasEntry, error) {
	af, err := LoadAliases(path)
	if err != nil {
		return nil, err
	}

	entries := make([]AliasEntry, 0, len(af.Aliases))
	for name, key := range af.Aliases {
		entries = append(entries, AliasEntry{Name: name, LinkKey: key})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}
