// Package render replaces {{PLACEHOLDER}} tokens in template content and file names.
package render

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Vars maps a placeholder name (without braces) to its value.
type Vars map[string]string

var (
	keyPattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	tokenPattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
)

// Builtins returns the placeholders Foundry always provides.
func Builtins(projectName, author, license string, now time.Time) Vars {
	words := splitWords(projectName)
	return Vars{
		"PROJECT_NAME":        projectName,
		"PROJECT_NAME_LOWER":  strings.ToLower(projectName),
		"PROJECT_NAME_UPPER":  strings.ToUpper(projectName),
		"PROJECT_NAME_SNAKE":  strings.ToLower(strings.Join(words, "_")),
		"PROJECT_NAME_KEBAB":  strings.ToLower(strings.Join(words, "-")),
		"PROJECT_NAME_PASCAL": pascal(words),
		"PROJECT_NAME_CAMEL":  camel(words),
		"AUTHOR":              author,
		"LICENSE":             license,
		"YEAR":                fmt.Sprintf("%d", now.Year()),
		"DATE":                now.Format("2006-01-02"),
	}
}

// BuiltinNames lists the names of the built-in placeholders, sorted.
func BuiltinNames() []string {
	names := make([]string, 0, 11)
	for k := range Builtins("", "", "", time.Time{}) {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ValidKey reports whether name can be used as a custom placeholder name.
func ValidKey(name string) bool { return keyPattern.MatchString(name) }

// Merge returns builtins overridden by custom variables. Custom variables win, which is deliberate:
// a user may want --var AUTHOR="Someone Else" for one project.
func Merge(builtins, custom Vars) Vars {
	out := make(Vars, len(builtins)+len(custom))
	for k, v := range builtins {
		out[k] = v
	}
	for k, v := range custom {
		out[k] = v
	}
	return out
}

// Apply replaces every known {{NAME}} token in s in a single pass, so a value that itself contains
// a token is never expanded again. Unknown tokens are left untouched. Spaces inside the braces are
// tolerated: {{ NAME }}.
func Apply(s string, vars Vars) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	return tokenPattern.ReplaceAllStringFunc(s, func(tok string) string {
		name := tokenPattern.FindStringSubmatch(tok)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return tok
	})
}

// Used returns the sorted, unique placeholder names that occur in s.
func Used(s string) []string {
	seen := map[string]struct{}{}
	for _, m := range tokenPattern.FindAllStringSubmatch(s, -1) {
		seen[m[1]] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Unresolved returns the placeholder names in s that vars does not define.
func Unresolved(s string, vars Vars) []string {
	var out []string
	for _, name := range Used(s) {
		if _, ok := vars[name]; !ok {
			out = append(out, name)
		}
	}
	return out
}

// splitWords splits on non-alphanumerics and on lower-to-upper camel case boundaries.
func splitWords(s string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = cur[:0]
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

func capitalize(w string) string {
	r := []rune(strings.ToLower(w))
	if len(r) == 0 {
		return ""
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func pascal(words []string) string {
	var b strings.Builder
	for _, w := range words {
		b.WriteString(capitalize(w))
	}
	return b.String()
}

func camel(words []string) string {
	if len(words) == 0 {
		return ""
	}
	return strings.ToLower(words[0]) + pascal(words[1:])
}
