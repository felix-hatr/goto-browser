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
	Links       []string `yaml:"links"`
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
	Links       []string
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

// AddGroup adds or updates a group.
func AddGroup(path string, group Group) error {
	gf, err := LoadGroups(path)
	if err != nil {
		return err
	}
	gf.Groups[group.Name] = GroupEntry{
		Description: group.Description,
		Params:      group.Params,
		Links:       group.Links,
	}
	return SaveGroups(path, gf)
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
	return &Group{Name: name, Description: entry.Description, Params: entry.Params, Links: entry.Links}, nil
}

// RemoveGroup deletes a group by name.
func RemoveGroup(path, name string) error {
	gf, err := LoadGroups(path)
	if err != nil {
		return err
	}
	if _, ok := gf.Groups[name]; !ok {
		return fmt.Errorf("group %q not found", name)
	}
	delete(gf.Groups, name)
	return SaveGroups(path, gf)
}

// ListGroups returns all groups sorted by name.
func ListGroups(path string) ([]Group, error) {
	gf, err := LoadGroups(path)
	if err != nil {
		return nil, err
	}
	groups := make([]Group, 0, len(gf.Groups))
	for name, entry := range gf.Groups {
		groups = append(groups, Group{Name: name, Description: entry.Description, Params: entry.Params, Links: entry.Links})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups, nil
}

// InsertIntoGroup adds link keys to an existing group at the given position (1-based).
// at=0 means append to end.
func InsertIntoGroup(path, name string, linkKeys []string, at int) error {
	gf, err := LoadGroups(path)
	if err != nil {
		return err
	}

	entry, ok := gf.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	if at <= 0 || at > len(entry.Links) {
		entry.Links = append(entry.Links, linkKeys...)
	} else {
		idx := at - 1
		links := make([]string, 0, len(entry.Links)+len(linkKeys))
		links = append(links, entry.Links[:idx]...)
		links = append(links, linkKeys...)
		links = append(links, entry.Links[idx:]...)
		entry.Links = links
	}

	gf.Groups[name] = entry
	return SaveGroups(path, gf)
}
