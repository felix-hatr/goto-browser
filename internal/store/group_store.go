package store

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// GroupEntry is the on-disk value for a group (name is the map key).
type GroupEntry struct {
	Description string   `yaml:"description,omitempty"`
	Params      []string `yaml:"params,omitempty"`
	URLs        []string `yaml:"urls"`
}

// GroupFile is the on-disk format for groups.yaml.
type GroupFile struct {
	Version string                `yaml:"version"`
	Groups  map[string]GroupEntry `yaml:"groups"`
}

// Group is the runtime representation combining name and entry.
type Group struct {
	Name        string
	Description string
	Params      []string
	URLs        []string
}

// LoadGroups reads the groups file for a profile.
func LoadGroups(path string) (*GroupFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GroupFile{Version: "1", Groups: map[string]GroupEntry{}}, nil
		}
		return nil, fmt.Errorf("reading groups: %w", err)
	}

	var gf GroupFile
	if err := yaml.Unmarshal(data, &gf); err != nil {
		return nil, fmt.Errorf("parsing groups: %w", err)
	}
	if gf.Groups == nil {
		gf.Groups = map[string]GroupEntry{}
	}
	return &gf, nil
}

// SaveGroups writes the groups file.
func SaveGroups(path string, gf *GroupFile) error {
	data, err := yaml.Marshal(gf)
	if err != nil {
		return fmt.Errorf("serializing groups: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// GetGroup retrieves a group by name.
func GetGroup(path, name string) (*Group, error) {
	gf, err := LoadGroups(path)
	if err != nil {
		return nil, err
	}
	entry, ok := gf.Groups[name]
	if !ok {
		return nil, fmt.Errorf("group %q not found", name)
	}
	return &Group{Name: name, Description: entry.Description, Params: entry.Params, URLs: entry.URLs}, nil
}

// ListGroups returns all groups sorted by name.
func ListGroups(path string) ([]Group, error) {
	gf, err := LoadGroups(path)
	if err != nil {
		return nil, err
	}
	groups := make([]Group, 0, len(gf.Groups))
	for name, entry := range gf.Groups {
		groups = append(groups, Group{Name: name, Description: entry.Description, Params: entry.Params, URLs: entry.URLs})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups, nil
}

// InsertIntoGroup adds URL templates to an existing group at the given position (1-based).
// at=0 means append to end.
func InsertIntoGroup(path, name string, urlTemplates []string, at int) error {
	gf, err := LoadGroups(path)
	if err != nil {
		return err
	}

	entry, ok := gf.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	if at <= 0 || at > len(entry.URLs) {
		entry.URLs = append(entry.URLs, urlTemplates...)
	} else {
		idx := at - 1
		urls := make([]string, 0, len(entry.URLs)+len(urlTemplates))
		urls = append(urls, entry.URLs[:idx]...)
		urls = append(urls, urlTemplates...)
		urls = append(urls, entry.URLs[idx:]...)
		entry.URLs = urls
	}

	gf.Groups[name] = entry
	return SaveGroups(path, gf)
}
