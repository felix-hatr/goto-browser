package resolver

import (
	"testing"

	"github.com/felix-hatr/goto-browser/internal/store"
)

// makeLink is a helper that creates a Link with positional storage.
func makeLink(key, url string) store.Link {
	normKey := store.NormalizeVars(key, "@")
	posKey, params := store.NormalizeToPositional(normKey)
	normURL := store.NormalizeVars(url, "@")
	nameToPos := store.NameToPos(params)
	posURL, _ := store.ApplyPositional(normURL, nameToPos)
	return store.Link{Key: posKey, URL: posURL, Params: params}
}

func TestResolve_ExactMatch(t *testing.T) {
	r := New("@")
	links := []store.Link{
		{Key: "github", URL: "https://github.com"},
	}
	result, err := r.Resolve("github", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://github.com" {
		t.Errorf("expected https://github.com, got %q", result.URL)
	}
}

func TestResolve_VariableSubstitution(t *testing.T) {
	r := New("@")
	links := []store.Link{
		makeLink("github/@account/@repo", "https://github.com/@account/@repo"),
	}
	result, err := r.Resolve("github/me/my-repo", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://github.com/me/my-repo" {
		t.Errorf("expected https://github.com/me/my-repo, got %q", result.URL)
	}
	// Vars should be mapped back to named form via Params
	if result.Vars["account"] != "me" {
		t.Errorf("expected account=me, got %q", result.Vars["account"])
	}
	if result.Vars["repo"] != "my-repo" {
		t.Errorf("expected repo=my-repo, got %q", result.Vars["repo"])
	}
}

func TestResolve_DirectURL(t *testing.T) {
	r := New("@")
	links := []store.Link{}
	result, err := r.Resolve("https://example.com/path", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://example.com/path" {
		t.Errorf("expected direct URL pass-through, got %q", result.URL)
	}
}

func TestResolve_MoreSpecificPatternWins(t *testing.T) {
	r := New("@")
	links := []store.Link{
		makeLink("github/@account", "https://github.com/@account"),
		makeLink("github/@account/@repo", "https://github.com/@account/@repo"),
	}
	result, err := r.Resolve("github/me/my-repo", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Pattern should be the positional form of github/@account/@repo
	posKey, _ := store.NormalizeToPositional(store.NormalizeVars("github/@account/@repo", "@"))
	wantPattern := store.DenormalizeVars(posKey, "@")
	if result.Pattern != wantPattern {
		t.Errorf("expected pattern %q, got %q", wantPattern, result.Pattern)
	}
}

func TestResolve_LiteralBeatsVariable(t *testing.T) {
	r := New("@")
	links := []store.Link{
		makeLink("github/@account", "https://github.com/@account"),
		{Key: "github/octocat", URL: "https://github.com/octocat"},
	}
	result, err := r.Resolve("github/octocat", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Pattern != "github/octocat" {
		t.Errorf("expected literal pattern to win, got %q", result.Pattern)
	}
}

func TestResolve_NoMatch(t *testing.T) {
	r := New("@")
	links := []store.Link{
		{Key: "github", URL: "https://github.com"},
	}
	_, err := r.Resolve("notexist", links)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestResolve_CaretPrefix(t *testing.T) {
	r := New("^")
	normKey := store.NormalizeVars("jira/^ticket", "^")
	posKey, params := store.NormalizeToPositional(normKey)
	normURL := store.NormalizeVars("https://jira.example.com/browse/^ticket", "^")
	posURL, _ := store.ApplyPositional(normURL, store.NameToPos(params))
	links := []store.Link{{Key: posKey, URL: posURL, Params: params}}

	result, err := r.Resolve("jira/PROJ-123", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://jira.example.com/browse/PROJ-123" {
		t.Errorf("expected caret prefix substitution, got %q", result.URL)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"github", "githb", 1},
		{"github", "gitlub", 1},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestResolve_TrailingSlashInput(t *testing.T) {
	r := New("@")
	links := []store.Link{
		makeLink("github/@account/@repo", "https://github.com/@account/@repo"),
	}
	result, err := r.Resolve("github/octocat/hello-world/", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://github.com/octocat/hello-world" {
		t.Errorf("unexpected URL: %q", result.URL)
	}
}

func TestResolve_TrailingSlashURL(t *testing.T) {
	r := New("@")
	links := []store.Link{
		{Key: "github", URL: "https://github.com/"},
	}
	result, err := r.Resolve("github", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://github.com" {
		t.Errorf("trailing slash should be stripped, got %q", result.URL)
	}
}

func TestResolve_DirectURLTrailingSlash(t *testing.T) {
	r := New("@")
	result, err := r.Resolve("https://example.com/path/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://example.com/path" {
		t.Errorf("trailing slash should be stripped from direct URL, got %q", result.URL)
	}
}

func TestResolve_PositionalStorage(t *testing.T) {
	r := New("@")
	links := []store.Link{
		makeLink("github/@account/@repo", "https://github.com/@account/@repo"),
	}
	result, err := r.Resolve("github/octocat/hello-world", links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://github.com/octocat/hello-world" {
		t.Errorf("expected https://github.com/octocat/hello-world, got %q", result.URL)
	}
	// Named vars accessible via Params mapping
	if result.Vars["account"] != "octocat" {
		t.Errorf("expected account=octocat, got %q", result.Vars["account"])
	}
	if result.Vars["repo"] != "hello-world" {
		t.Errorf("expected repo=hello-world, got %q", result.Vars["repo"])
	}
}

func TestMatchGroup_Concrete(t *testing.T) {
	r := New("@")
	groups := []store.Group{
		{Name: "morning", URLs: []string{"https://github.com", "https://google.com"}},
	}
	g, vars, err := r.MatchGroup("morning", groups)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name != "morning" {
		t.Errorf("expected group morning, got %q", g.Name)
	}
	if len(vars) != 0 {
		t.Errorf("expected no vars, got %v", vars)
	}
}

func TestMatchGroup_WithVariables(t *testing.T) {
	r := New("@")
	// Group stored with positional vars
	normName := store.NormalizeVars("dev/@account/@repo", "@")
	posName, params := store.NormalizeToPositional(normName)
	groups := []store.Group{
		{Name: posName, Params: params, URLs: []string{}},
	}

	g, vars, err := r.MatchGroup("dev/myorg/myrepo", groups)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g == nil {
		t.Fatal("expected group, got nil")
	}
	// MatchGroup returns positional vars (keys "1", "2") for direct use in ResolveGroupLinks
	if vars["1"] != "myorg" {
		t.Errorf("expected @1=myorg, got %q", vars["1"])
	}
	if vars["2"] != "myrepo" {
		t.Errorf("expected @2=myrepo, got %q", vars["2"])
	}
}

func TestResolveGroupLinks_WithVariables(t *testing.T) {
	r := New("@")

	ghLink := makeLink("github/@account/@repo", "https://github.com/@account/@repo")
	googleLink := makeLink("google/@account", "https://google.com/search?q=@account")

	links := []store.Link{ghLink, googleLink}

	// Group link templates (positional form)
	ghTmpl := store.NormalizeVars("github/@account/@repo", "@")
	ghTmplPos, _ := store.NormalizeToPositional(ghTmpl)
	googleTmpl := store.NormalizeVars("google/@account", "@")
	googleTmplPos, _ := store.NormalizeToPositional(googleTmpl)

	// But these are group-level: @account → position 1, @repo → position 2 in the group
	// The group has params: [account, repo], so @1=account, @2=repo
	// Group link templates reference group positions
	groupLinks := []string{ghTmplPos, googleTmplPos}

	// Positional vars: "1" = @1, "2" = @2 (as returned by MatchGroup)
	groupVars := map[string]string{"1": "myorg", "2": "myrepo"}

	urls, errs := r.ResolveGroupLinks(groupLinks, groupVars, links)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
	if urls[0] != "https://github.com/myorg/myrepo" {
		t.Errorf("expected github URL, got %q", urls[0])
	}
	if urls[1] != "https://google.com/search?q=myorg" {
		t.Errorf("expected google URL, got %q", urls[1])
	}
}
