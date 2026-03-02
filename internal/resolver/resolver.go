package resolver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/felix-hatr/goto-browser/internal/store"
)


// Result holds the resolved URL and metadata.
type Result struct {
	URL     string
	Pattern string
	Vars    map[string]string // keyed by variable name (from Params if available)
}

// Resolver resolves input link keys to URLs.
type Resolver struct {
	variablePrefix string
}

// New creates a new Resolver with the given variable prefix (e.g., "@").
func New(variablePrefix string) *Resolver {
	if variablePrefix == "" {
		variablePrefix = "@"
	}
	return &Resolver{variablePrefix: variablePrefix}
}

// segment represents a parsed URL pattern segment.
type segment struct {
	isVar bool
	value string // literal value or variable name/number
}

// Resolve matches the input string against stored links, substitutes variables,
// and returns the final URL.
func (r *Resolver) Resolve(input string, links []store.Link) (*Result, error) {
	// Direct URL: strip trailing slash then pass through
	if strings.Contains(input, "://") {
		url := strings.TrimRight(input, "/")
		return &Result{URL: url, Pattern: url, Vars: map[string]string{}}, nil
	}

	// Normalize input: strip leading/trailing slashes
	input = strings.Trim(input, "/")

	// Parse input segments (empty segments from double-slash are dropped)
	rawSegs := strings.Split(input, "/")
	inputSegs := make([]string, 0, len(rawSegs))
	for _, s := range rawSegs {
		if s != "" {
			inputSegs = append(inputSegs, s)
		}
	}

	type candidate struct {
		link   store.Link
		score  int
		vars   map[string]string
		params []string
	}

	var candidates []candidate

	for _, raw := range links {
		// Denormalize stored token to actual prefix before matching
		link := store.Link{
			Key:         store.DenormalizeVars(raw.Key, r.variablePrefix),
			URL:         store.DenormalizeVars(raw.URL, r.variablePrefix),
			Description: raw.Description,
			Params:      raw.Params,
		}
		score, vars, ok := r.matchPattern(inputSegs, link.Key)
		if ok {
			candidates = append(candidates, candidate{link: link, score: score, vars: vars, params: raw.Params})
		}
	}

	if len(candidates) == 0 {
		suggestions := r.suggest(input, links)
		if len(suggestions) > 0 {
			return nil, fmt.Errorf("no match for %q. Did you mean: %s", input, strings.Join(suggestions, ", "))
		}
		return nil, fmt.Errorf("no match for %q", input)
	}

	// Pick highest score; ties broken by shorter key (more specific pattern)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return len(candidates[i].link.Key) < len(candidates[j].link.Key)
	})

	best := candidates[0]
	url := strings.TrimRight(r.substituteVars(best.link.URL, best.vars), "/")

	// Map positional vars back to named vars if params are available.
	namedVars := best.vars
	if len(best.params) > 0 {
		namedVars = make(map[string]string, len(best.vars))
		for posStr, val := range best.vars {
			pos, err := strconv.Atoi(posStr)
			if err == nil && pos >= 1 && pos <= len(best.params) {
				namedVars[best.params[pos-1]] = val
			} else {
				namedVars[posStr] = val
			}
		}
	}

	return &Result{URL: url, Pattern: best.link.Key, Vars: namedVars}, nil
}

// parsePattern parses a link key pattern into segments.
func (r *Resolver) parsePattern(key string) []segment {
	parts := strings.Split(key, "/")
	segs := make([]segment, len(parts))
	for i, p := range parts {
		if strings.HasPrefix(p, r.variablePrefix) {
			segs[i] = segment{isVar: true, value: p[len(r.variablePrefix):]}
		} else {
			segs[i] = segment{isVar: false, value: p}
		}
	}
	return segs
}

// matchPattern tries to match inputSegs against the link key pattern.
// Returns (score, vars, matched).
func (r *Resolver) matchPattern(inputSegs []string, key string) (int, map[string]string, bool) {
	patternSegs := r.parsePattern(key)

	if len(inputSegs) != len(patternSegs) {
		return 0, nil, false
	}

	score := 0
	vars := map[string]string{}

	for i, ps := range patternSegs {
		input := inputSegs[i]
		if ps.isVar {
			vars[ps.value] = input
			score += 1
		} else {
			if ps.value != input {
				return 0, nil, false
			}
			score += 10
		}
	}

	return score, vars, true
}

// substituteVars replaces prefix+varname placeholders in the URL template.
func (r *Resolver) substituteVars(urlTemplate string, vars map[string]string) string {
	result := urlTemplate
	for name, val := range vars {
		result = strings.ReplaceAll(result, r.variablePrefix+name, val)
	}
	return result
}

// MatchGroup finds the group whose name pattern matches the input and returns
// the group along with the variable bindings.
func (r *Resolver) MatchGroup(input string, groups []store.Group) (*store.Group, map[string]string, error) {
	input = strings.Trim(input, "/")
	rawSegs := strings.Split(input, "/")
	inputSegs := make([]string, 0, len(rawSegs))
	for _, s := range rawSegs {
		if s != "" {
			inputSegs = append(inputSegs, s)
		}
	}

	for i := range groups {
		g := &groups[i]
		groupKey := store.DenormalizeVars(g.Name, r.variablePrefix)
		_, vars, ok := r.matchPattern(inputSegs, groupKey)
		if ok {
			// Return positional vars as-is (keys "1", "2", ...) so that
			// ResolveGroupLinks can directly substitute into positional templates.
			return g, vars, nil
		}
	}
	return nil, nil, fmt.Errorf("no group matching %q", input)
}

// ResolveGroupLinks resolves all link templates in a group to URLs.
// groupVars maps variable names (from group Params) to their bound values.
// Templates with variables are substituted first, then resolved against the link store.
func (r *Resolver) ResolveGroupLinks(linkTemplates []string, groupVars map[string]string, links []store.Link) ([]string, []error) {
	urls := make([]string, 0, len(linkTemplates))
	var errs []error

	for _, tmpl := range linkTemplates {
		// Denormalize: <vp>1 → @1 (or @account for named)
		denorm := store.DenormalizeVars(tmpl, r.variablePrefix)
		// Apply group variable bindings: @1 → value (positional) or @account → value (named)
		concreteKey := r.substituteVars(denorm, groupVars)
		// Resolve concrete key through the link store
		result, err := r.Resolve(concreteKey, links)
		if err != nil {
			errs = append(errs, fmt.Errorf("resolving %q: %w", store.DenormalizeVars(tmpl, r.variablePrefix), err))
			continue
		}
		urls = append(urls, result.URL)
	}

	return urls, errs
}

// suggest returns similar link keys using simple prefix/substring matching.
func (r *Resolver) suggest(input string, links []store.Link) []string {
	inputLower := strings.ToLower(input)
	// First segment for comparison
	firstSeg := strings.SplitN(inputLower, "/", 2)[0]

	type scored struct {
		key   string
		score int
	}

	var results []scored
	for _, l := range links {
		keyLower := strings.ToLower(store.DenormalizeVars(l.Key, r.variablePrefix))
		firstKeySegment := strings.SplitN(keyLower, "/", 2)[0]

		score := levenshtein(firstSeg, firstKeySegment)
		if score <= 3 || strings.HasPrefix(keyLower, firstSeg) || strings.HasPrefix(firstSeg, firstKeySegment) {
			results = append(results, scored{key: l.Key, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score < results[j].score
	})

	suggestions := make([]string, 0, min(3, len(results)))
	for i, r := range results {
		if i >= 3 {
			break
		}
		suggestions = append(suggestions, fmt.Sprintf("%q", r.key))
	}
	return suggestions
}

// levenshtein computes the edit distance between two strings using a single-row DP.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use a single row DP approach
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				curr[j-1]+1,
				min(prev[j]+1, prev[j-1]+cost),
			)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
