package store

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// VarToken is the canonical placeholder stored in link keys and URLs.
// e.g., "github/<vp>1/<vp>2"
const VarToken = "<vp>"

var (
	// namedVarRe matches internal named tokens: <variable_prefix>account
	namedVarRe = regexp.MustCompile(regexp.QuoteMeta(VarToken) + `([A-Za-z_][A-Za-z0-9_]*)`)

	// positionalVarRe matches internal positional tokens: <variable_prefix>1
	positionalVarRe = regexp.MustCompile(regexp.QuoteMeta(VarToken) + `(\d+)`)
)

// buildVarPattern returns a regexp matching prefix+varname or prefix+digits.
// Supports both named (@account) and positional (@1, @2) variable input.
func buildVarPattern(prefix string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(prefix) + `(\d+|[A-Za-z_][A-Za-z0-9_]*)`)
}

// NormalizeVars replaces prefix+varname occurrences in s with VarToken+varname.
func NormalizeVars(s, prefix string) string {
	pattern := buildVarPattern(prefix)
	return pattern.ReplaceAllStringFunc(s, func(m string) string {
		return VarToken + m[len(prefix):]
	})
}

// DenormalizeVars replaces VarToken occurrences in s with the configured prefix.
func DenormalizeVars(s, prefix string) string {
	return strings.ReplaceAll(s, VarToken, prefix)
}

// ContainsVarToken reports whether s contains any variable token for the given prefix.
func ContainsVarToken(s, prefix string) bool {
	return buildVarPattern(prefix).MatchString(s)
}

// NormalizeToPositional replaces named var tokens (<vp>name) with positional tokens
// (<vp>1, <vp>2, ...) in order of first appearance.
// Returns the positional string and params where params[i] = name for position i+1.
// For purely positional input (<vp>1, <vp>2), result is unchanged and params is nil.
func NormalizeToPositional(s string) (result string, params []string) {
	nameToPos := map[string]int{}
	result = namedVarRe.ReplaceAllStringFunc(s, func(m string) string {
		name := m[len(VarToken):]
		if _, exists := nameToPos[name]; !exists {
			nameToPos[name] = len(params) + 1
			params = append(params, name)
		}
		return VarToken + strconv.Itoa(nameToPos[name])
	})
	return
}

// HasVars reports whether s contains any variable tokens (named or positional).
// Use this to distinguish variable patterns from concrete keys.
func HasVars(s string) bool {
	return strings.Contains(s, VarToken)
}

// ApplyPositional applies an existing name→position mapping to s.
// Returns an error if s contains named vars not found in nameToPos.
func ApplyPositional(s string, nameToPos map[string]int) (string, error) {
	var unknown []string
	result := namedVarRe.ReplaceAllStringFunc(s, func(m string) string {
		name := m[len(VarToken):]
		pos, ok := nameToPos[name]
		if !ok {
			unknown = append(unknown, name)
			return m
		}
		return VarToken + strconv.Itoa(pos)
	})
	if len(unknown) > 0 {
		return "", fmt.Errorf("undefined variable(s): %s", strings.Join(unknown, ", "))
	}
	return result, nil
}

// DenormalizeParams replaces positional tokens (<vp>N) with named vars using the params slice.
// params[0] = name for position 1, params[1] = name for position 2, etc.
// Falls back to DenormalizeVars for strings without positional tokens.
func DenormalizeParams(s, prefix string, params []string) string {
	if len(params) == 0 {
		return DenormalizeVars(s, prefix)
	}
	return positionalVarRe.ReplaceAllStringFunc(s, func(m string) string {
		numStr := m[len(VarToken):]
		n, _ := strconv.Atoi(numStr)
		if n >= 1 && n <= len(params) {
			return prefix + params[n-1]
		}
		return m
	})
}

// FillPositional substitutes positional tokens (<vp>1, <vp>2, ...) with values from vals.
// vals[0] = value for position 1, vals[1] = value for position 2, etc.
func FillPositional(s string, vals []string) string {
	return positionalVarRe.ReplaceAllStringFunc(s, func(m string) string {
		numStr := m[len(VarToken):]
		n, _ := strconv.Atoi(numStr)
		if n >= 1 && n <= len(vals) {
			return vals[n-1]
		}
		return m
	})
}

// NameToPos converts a params slice to a name→position map.
func NameToPos(params []string) map[string]int {
	m := map[string]int{}
	for i, name := range params {
		m[name] = i + 1
	}
	return m
}

// ExtractPositionalNums returns a sorted, deduplicated list of positional indices
// found in s (e.g., [1, 2] for "github/<vp>1/<vp>2").
func ExtractPositionalNums(s string) []int {
	seen := map[int]bool{}
	var nums []int
	for _, m := range positionalVarRe.FindAllStringSubmatch(s, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && !seen[n] {
			seen[n] = true
			nums = append(nums, n)
		}
	}
	sort.Ints(nums)
	return nums
}

// ExtractVarNames returns a sorted, deduplicated list of named variable names
// from a normalized string (containing VarToken+name tokens).
func ExtractVarNames(s string) []string {
	matches := namedVarRe.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	var names []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	sort.Strings(names)
	return names
}
