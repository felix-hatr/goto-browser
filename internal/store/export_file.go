package store

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ExportFile is the on-disk format for export/import files.
// Variable tokens are stored in their internal <vp>N form (prefix-independent).
type ExportFile struct {
	Version string                `yaml:"version"`
	Links   map[string]LinkEntry  `yaml:"links,omitempty"`
	Groups  map[string]GroupEntry `yaml:"groups,omitempty"`
	Config  map[string]string     `yaml:"config,omitempty"`
}

// LoadExportFile reads and parses an export file from disk.
func LoadExportFile(path string) (*ExportFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading export file: %w", err)
	}
	var ef ExportFile
	if err := yaml.Unmarshal(data, &ef); err != nil {
		return nil, fmt.Errorf("parsing export file: %w", err)
	}
	return &ef, nil
}

// SaveExportFile writes an export file to disk.
func SaveExportFile(path string, ef *ExportFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating export dir: %w", err)
	}
	data, err := MarshalExportFile(ef)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// MarshalExportFile serializes an ExportFile to YAML bytes.
func MarshalExportFile(ef *ExportFile) ([]byte, error) {
	data, err := yaml.Marshal(ef)
	if err != nil {
		return nil, fmt.Errorf("serializing export file: %w", err)
	}
	return data, nil
}
