package store

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExportFileMarshalRoundTrip(t *testing.T) {
	original := &ExportFile{
		Version: "1",
		Links: map[string]LinkEntry{
			"github": {URL: "https://github.com", Description: "GitHub"},
			"jira/<vp>1": {URL: "https://jira.example.com/browse/<vp>1", Params: []string{"ticket"}},
		},
		Groups: map[string]GroupEntry{
			"morning": {
				Description: "Morning links",
				URLs:        []string{"https://github.com", "https://slack.com"},
			},
		},
		Config: map[string]string{
			"browser":          "chrome",
			"variable_prefix":  "@",
		},
	}

	// Marshal
	data, err := MarshalExportFile(original)
	if err != nil {
		t.Fatalf("MarshalExportFile: %v", err)
	}

	// Unmarshal
	var restored ExportFile
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if restored.Version != original.Version {
		t.Errorf("version: got %q, want %q", restored.Version, original.Version)
	}
	if len(restored.Links) != len(original.Links) {
		t.Errorf("links count: got %d, want %d", len(restored.Links), len(original.Links))
	}
	if len(restored.Groups) != len(original.Groups) {
		t.Errorf("groups count: got %d, want %d", len(restored.Groups), len(original.Groups))
	}
	if len(restored.Config) != len(original.Config) {
		t.Errorf("config count: got %d, want %d", len(restored.Config), len(original.Config))
	}

	// Check specific values
	gh, ok := restored.Links["github"]
	if !ok {
		t.Error("github link not found after round-trip")
	} else if gh.URL != "https://github.com" {
		t.Errorf("github URL: got %q, want %q", gh.URL, "https://github.com")
	}

	morning, ok := restored.Groups["morning"]
	if !ok {
		t.Error("morning group not found after round-trip")
	} else if len(morning.URLs) != 2 {
		t.Errorf("morning URLs count: got %d, want 2", len(morning.URLs))
	}
}

func TestExportFileOmitEmpty(t *testing.T) {
	ef := &ExportFile{
		Version: "1",
		// Links, Groups, Config all nil
	}

	data, err := MarshalExportFile(ef)
	if err != nil {
		t.Fatalf("MarshalExportFile: %v", err)
	}

	content := string(data)

	// With omitempty, nil maps should not appear in YAML output
	if contains(content, "links:") {
		t.Error("expected 'links:' to be omitted when nil")
	}
	if contains(content, "groups:") {
		t.Error("expected 'groups:' to be omitted when nil")
	}
	if contains(content, "config:") {
		t.Error("expected 'config:' to be omitted when nil")
	}
	if !contains(content, "version:") {
		t.Error("expected 'version:' to be present")
	}
}

func TestExportFileLinksOnly(t *testing.T) {
	ef := &ExportFile{
		Version: "1",
		Links: map[string]LinkEntry{
			"test": {URL: "https://example.com"},
		},
	}

	data, err := MarshalExportFile(ef)
	if err != nil {
		t.Fatalf("MarshalExportFile: %v", err)
	}

	var restored ExportFile
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if len(restored.Links) != 1 {
		t.Errorf("links count: got %d, want 1", len(restored.Links))
	}
	if restored.Groups != nil {
		t.Errorf("groups should be nil, got %v", restored.Groups)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
