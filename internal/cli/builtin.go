package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/SandSoftwareSolutions/Foundry/templates"
)

// entry is a template the user can pick: one they saved, or one that ships with Foundry.
type entry struct {
	config.Template
	Builtin bool
}

// lookup finds a template by name. Templates the user saved win over built-in ones with the same name.
func lookup(cfg *config.Config, name string) (entry, bool) {
	if t, ok := cfg.FindTemplate(name); ok {
		return entry{Template: *t}, true
	}
	if b, ok := templates.Get(name); ok {
		return entry{Template: config.Template{Name: b.Name, Language: b.Language, Description: b.Description}, Builtin: true}, true
	}
	return entry{}, false
}

// everyTemplate returns the saved and the built-in templates, sorted by name; a saved template hides a
// built-in one with the same name.
func everyTemplate(cfg *config.Config) []entry {
	saved := map[string]bool{}
	var out []entry
	for _, t := range cfg.Templates {
		saved[strings.ToLower(t.Name)] = true
		out = append(out, entry{Template: t})
	}
	for _, b := range templates.Catalog() {
		if saved[strings.ToLower(b.Name)] {
			continue
		}
		out = append(out, entry{Template: config.Template{Name: b.Name, Language: b.Language, Description: b.Description}, Builtin: true})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func everyName(cfg *config.Config) []string {
	list := everyTemplate(cfg)
	names := make([]string, len(list))
	for i, e := range list {
		names[i] = e.Name
	}
	return names
}

func languagesOf(list []entry) []string {
	set := map[string]bool{}
	for _, e := range list {
		if e.Language != "" {
			set[e.Language] = true
		}
	}
	out := make([]string, 0, len(set))
	for l := range set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// materialize makes sure the template exists as a directory. A built-in template is written to a
// temporary directory; the returned function removes it again.
func materialize(e entry) (config.Template, func(), error) {
	noop := func() {}
	if !e.Builtin {
		return e.Template, noop, nil
	}
	dir, err := os.MkdirTemp("", "foundry-builtin-*")
	if err != nil {
		return config.Template{}, noop, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := templates.Extract(e.Name, dir); err != nil {
		cleanup()
		return config.Template{}, noop, fmt.Errorf("cannot unpack the built-in template %q: %w", e.Name, err)
	}
	t := e.Template
	t.Path = dir
	return t, cleanup, nil
}

// builtinChangeError explains why a built-in template cannot be edited or removed.
func builtinChangeError(name string) error {
	return usagef("%q is a built-in template and cannot be changed or removed.\nMake a copy you can edit with: foundry template export %s <folder>, then: foundry template add <name> <folder>", name, name)
}
