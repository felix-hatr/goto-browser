package store

import (
	"reflect"
	"testing"
)

func TestNormalizeVars(t *testing.T) {
	tests := []struct {
		input  string
		prefix string
		want   string
	}{
		{"github/@account/@repo", "@", "github/<vp>account/<vp>repo"},
		{"https://github.com/@account/@repo", "@", "https://github.com/<vp>account/<vp>repo"},
		{"jira/@ticket", "@", "jira/<vp>ticket"},
		{"@var", "@", "<vp>var"},
		{"github", "@", "github"},
		{"no-vars-here", "@", "no-vars-here"},
		{"", "@", ""},
		{"gh/^account/^repo", "^", "gh/<vp>account/<vp>repo"},
		{"@account", "^", "@account"}, // @ not normalized when prefix is ^
	}
	for _, tt := range tests {
		got := NormalizeVars(tt.input, tt.prefix)
		if got != tt.want {
			t.Errorf("NormalizeVars(%q, %q) = %q, want %q", tt.input, tt.prefix, got, tt.want)
		}
	}
}

func TestDenormalizeVars(t *testing.T) {
	tests := []struct {
		input  string
		prefix string
		want   string
	}{
		{"github/<vp>account/<vp>repo", "@", "github/@account/@repo"},
		{"github/<vp>account/<vp>repo", "^", "github/^account/^repo"},
		{"jira/<vp>ticket", "@", "jira/@ticket"},
		{"github", "@", "github"},
		{"", "@", ""},
	}
	for _, tt := range tests {
		got := DenormalizeVars(tt.input, tt.prefix)
		if got != tt.want {
			t.Errorf("DenormalizeVars(%q, %q) = %q, want %q", tt.input, tt.prefix, got, tt.want)
		}
	}
}

func TestNormalizeDenormalizeRoundtrip(t *testing.T) {
	for _, prefix := range []string{"@", "^"} {
		input := "github/" + prefix + "account/" + prefix + "repo"
		normalized := NormalizeVars(input, prefix)
		got := DenormalizeVars(normalized, prefix)
		if got != input {
			t.Errorf("roundtrip with prefix %q: got %q, want %q", prefix, got, input)
		}
	}
}

func TestNormalizeToPositional(t *testing.T) {
	tests := []struct {
		input      string
		wantResult string
		wantParams []string
	}{
		{
			"github/<vp>account/<vp>repo",
			"github/<vp>1/<vp>2",
			[]string{"account", "repo"},
		},
		{
			"jira/<vp>ticket",
			"jira/<vp>1",
			[]string{"ticket"},
		},
		{
			// Same var used twice → same position
			"<vp>a/<vp>b/<vp>a",
			"<vp>1/<vp>2/<vp>1",
			[]string{"a", "b"},
		},
		{
			"github",
			"github",
			nil,
		},
	}
	for _, tt := range tests {
		gotResult, gotParams := NormalizeToPositional(tt.input)
		if gotResult != tt.wantResult {
			t.Errorf("NormalizeToPositional(%q) result = %q, want %q", tt.input, gotResult, tt.wantResult)
		}
		if !reflect.DeepEqual(gotParams, tt.wantParams) {
			t.Errorf("NormalizeToPositional(%q) params = %v, want %v", tt.input, gotParams, tt.wantParams)
		}
	}
}

func TestApplyPositional(t *testing.T) {
	nameToPos := map[string]int{"account": 1, "repo": 2}

	tests := []struct {
		input   string
		wantOut string
		wantErr bool
	}{
		{
			"github/<vp>account/<vp>repo",
			"github/<vp>1/<vp>2",
			false,
		},
		{
			"github/<vp>account",
			"github/<vp>1",
			false,
		},
		{
			"github",
			"github",
			false,
		},
		{
			"github/<vp>unknown",
			"",
			true, // undefined variable
		},
	}
	for _, tt := range tests {
		got, err := ApplyPositional(tt.input, nameToPos)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ApplyPositional(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ApplyPositional(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.wantOut {
				t.Errorf("ApplyPositional(%q) = %q, want %q", tt.input, got, tt.wantOut)
			}
		}
	}
}

func TestDenormalizeParams(t *testing.T) {
	params := []string{"account", "repo"}

	tests := []struct {
		input  string
		prefix string
		params []string
		want   string
	}{
		{
			"github/<vp>1/<vp>2",
			"@",
			params,
			"github/@account/@repo",
		},
		{
			"github/<vp>1",
			"@",
			params,
			"github/@account",
		},
		{
			"github/<vp>1/<vp>2",
			"^",
			params,
			"github/^account/^repo",
		},
		{
			// No params → falls back to DenormalizeVars
			"github/<vp>account",
			"@",
			nil,
			"github/@account",
		},
		{
			"github",
			"@",
			params,
			"github",
		},
	}
	for _, tt := range tests {
		got := DenormalizeParams(tt.input, tt.prefix, tt.params)
		if got != tt.want {
			t.Errorf("DenormalizeParams(%q, %q, %v) = %q, want %q", tt.input, tt.prefix, tt.params, got, tt.want)
		}
	}
}

func TestExtractVarNames(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"github/<vp>account/<vp>repo", []string{"account", "repo"}},
		{"jira/<vp>ticket", []string{"ticket"}},
		{"github", nil},
		{"<vp>b/<vp>a", []string{"a", "b"}}, // sorted
	}
	for _, tt := range tests {
		got := ExtractVarNames(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ExtractVarNames(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
