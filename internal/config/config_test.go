package config

import (
	"testing"
)

func TestApplyConfigDefaults(t *testing.T) {
	cfg := &GlobalConfig{}
	applyConfigDefaults(cfg)

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"browser", cfg.Browser, "chrome"},
		{"variable_prefix", cfg.VariablePrefix, "@"},
		{"variable_display", cfg.VariableDisplay, "named"},
		{"open_mode", cfg.OpenMode, "new_tab"},
		{"open_default", cfg.OpenDefault, "link"},
		{"profile_delete_mode", cfg.ProfileDeleteMode, "backup"},
		{"profile_view_mode", cfg.ProfileViewMode, "summary"},
		{"history_dedup", cfg.HistoryDedup, "consecutive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
	if cfg.HistorySize != 10000 {
		t.Errorf("history_size: got %d, want 10000", cfg.HistorySize)
	}
}

func TestApplyConfigDefaultsPreservesExisting(t *testing.T) {
	cfg := &GlobalConfig{
		Browser:        "brave",
		VariablePrefix: "^",
		HistorySize:    500,
	}
	applyConfigDefaults(cfg)
	if cfg.Browser != "brave" {
		t.Errorf("browser should not be overwritten: got %q", cfg.Browser)
	}
	if cfg.VariablePrefix != "^" {
		t.Errorf("variable_prefix should not be overwritten: got %q", cfg.VariablePrefix)
	}
	if cfg.HistorySize != 500 {
		t.Errorf("history_size should not be overwritten: got %d", cfg.HistorySize)
	}
}

func TestGlobalConfigGet(t *testing.T) {
	cfg := &GlobalConfig{}
	applyConfigDefaults(cfg)

	val, err := cfg.Get("browser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "chrome" {
		t.Errorf("got %q, want %q", val, "chrome")
	}

	_, err = cfg.Get("nonexistent_key")
	if err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestGlobalConfigSet(t *testing.T) {
	cfg := &GlobalConfig{}
	applyConfigDefaults(cfg)

	if err := cfg.Set("browser", "brave"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Browser != "brave" {
		t.Errorf("got %q, want %q", cfg.Browser, "brave")
	}

	// Invalid variable_display
	if err := cfg.Set("variable_display", "invalid"); err == nil {
		t.Error("expected error for invalid variable_display, got nil")
	}

	// Valid variable_display
	if err := cfg.Set("variable_display", "positional"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.VariableDisplay != "positional" {
		t.Errorf("got %q, want %q", cfg.VariableDisplay, "positional")
	}

	// Invalid history_size
	if err := cfg.Set("history_size", "0"); err == nil {
		t.Error("expected error for history_size=0, got nil")
	}
	if err := cfg.Set("history_size", "abc"); err == nil {
		t.Error("expected error for non-numeric history_size, got nil")
	}

	// Valid history_size
	if err := cfg.Set("history_size", "500"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HistorySize != 500 {
		t.Errorf("got %d, want %d", cfg.HistorySize, 500)
	}

	// Unknown key
	if err := cfg.Set("unknown_key", "val"); err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestProfileConfigGetSet(t *testing.T) {
	pc := &ProfileConfig{Name: "test"}

	// description is profile-only
	if err := pc.Set("description", "My profile"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, err := pc.Get("description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "My profile" {
		t.Errorf("got %q, want %q", val, "My profile")
	}

	// Invalid open_default
	if err := pc.Set("open_default", "invalid"); err == nil {
		t.Error("expected error for invalid open_default, got nil")
	}

	// Valid open_default
	if err := pc.Set("open_default", "group"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.OpenDefault != "group" {
		t.Errorf("got %q, want %q", pc.OpenDefault, "group")
	}

	// Unknown key
	if err := pc.Set("unknown_key", "val"); err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestValidateConfigValue(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"variable_display", "named", false},
		{"variable_display", "positional", false},
		{"variable_display", "invalid", true},
		{"open_mode", "new_tab", false},
		{"open_mode", "new_window", false},
		{"open_mode", "full_screen", true},
		{"open_default", "link", false},
		{"open_default", "group", false},
		{"open_default", "url", false},
		{"open_default", "direct", true},
		{"profile_delete_mode", "backup", false},
		{"profile_delete_mode", "permanent", false},
		{"profile_delete_mode", "trash", true},
		{"profile_view_mode", "summary", false},
		{"profile_view_mode", "detail", false},
		{"profile_view_mode", "full", true},
		{"history_dedup", "none", false},
		{"history_dedup", "consecutive", false},
		{"history_dedup", "all", false},
		{"history_dedup", "erasedup", true},
		{"history_size", "100", false},
		{"history_size", "-1", false},
		{"history_size", "0", true},
		{"history_size", "abc", true},
		{"browser", "chrome", false},
		{"browser", "any-browser", false},
		{"description", "any text", false},
	}

	for _, tt := range tests {
		t.Run(tt.key+"/"+tt.value, func(t *testing.T) {
			err := validateConfigValue(tt.key, tt.value)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for key=%q value=%q, got nil", tt.key, tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for key=%q value=%q: %v", tt.key, tt.value, err)
			}
		})
	}
}

func TestValidateVariablePrefix(t *testing.T) {
	validPrefixes := []string{"@", "^", ":", "~", "+", "=", ",", ".", ";", "-", "_"}
	for _, p := range validPrefixes {
		if err := validateVariablePrefix(p); err != nil {
			t.Errorf("valid prefix %q rejected: %v", p, err)
		}
	}

	invalidPrefixes := []string{"", "@@", "a", "1", "/", "$", "!", "?", "#", "%"}
	for _, p := range invalidPrefixes {
		if err := validateVariablePrefix(p); err == nil {
			t.Errorf("invalid prefix %q accepted", p)
		}
	}
}
