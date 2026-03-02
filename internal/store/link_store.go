package store

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// LinkEntry is the on-disk value for a link (key is the map key).
type LinkEntry struct {
	URL         string   `yaml:"url"`
	Description string   `yaml:"description,omitempty"`
	Params      []string `yaml:"params,omitempty"`
}

// LinkFile is the on-disk format for links.yaml.
type LinkFile struct {
	Version string               `yaml:"version"`
	Links   map[string]LinkEntry `yaml:"links"`
}

// Link is the runtime representation combining key and entry.
type Link struct {
	Key         string
	URL         string
	Description string
	Params      []string
}

// LoadLinks reads the links file for a profile.
func LoadLinks(path string) (*LinkFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LinkFile{Version: "1", Links: map[string]LinkEntry{}}, nil
		}
		return nil, fmt.Errorf("reading links: %w", err)
	}

	var lf LinkFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing links: %w", err)
	}
	if lf.Links == nil {
		lf.Links = map[string]LinkEntry{}
	}
	return &lf, nil
}

// SaveLinks writes the links file.
func SaveLinks(path string, lf *LinkFile) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("serializing links: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// AddLink adds or updates a link.
func AddLink(path string, link Link) error {
	lf, err := LoadLinks(path)
	if err != nil {
		return err
	}
	lf.Links[link.Key] = LinkEntry{
		URL:         link.URL,
		Description: link.Description,
		Params:      link.Params,
	}
	return SaveLinks(path, lf)
}

// GetLink retrieves a link by key.
func GetLink(path, key string) (*Link, error) {
	lf, err := LoadLinks(path)
	if err != nil {
		return nil, err
	}
	entry, ok := lf.Links[key]
	if !ok {
		return nil, fmt.Errorf("link %q not found", key)
	}
	return &Link{Key: key, URL: entry.URL, Description: entry.Description, Params: entry.Params}, nil
}

// RemoveLink deletes a link by key.
func RemoveLink(path, key string) error {
	lf, err := LoadLinks(path)
	if err != nil {
		return err
	}
	if _, ok := lf.Links[key]; !ok {
		return fmt.Errorf("link %q not found", key)
	}
	delete(lf.Links, key)
	return SaveLinks(path, lf)
}

// ListLinks returns all links sorted by key.
func ListLinks(path string) ([]Link, error) {
	lf, err := LoadLinks(path)
	if err != nil {
		return nil, err
	}
	links := make([]Link, 0, len(lf.Links))
	for key, entry := range lf.Links {
		links = append(links, Link{Key: key, URL: entry.URL, Description: entry.Description, Params: entry.Params})
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Key < links[j].Key })
	return links, nil
}
