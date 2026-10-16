// Package templates holds the templates that ship inside the Foundry binary.
//
// They are ordinary template directories (see docs/templates.md) under builtin/, embedded at build
// time, so they work the same for apt, go install and the release archives. A catalog file lists
// them with a language and a description.
package templates

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed all:_builtin
var files embed.FS

// root starts with an underscore on purpose: the Go tools skip such directories, so the Go sources of the
// templates are not compiled, vetted or tested as part of Foundry itself.
const root = "_builtin"

// A go.mod file would turn its directory into a separate module that //go:embed refuses to read, so
// templates keep it as go.mod.txt. It gets its real name when the template is read or extracted.
const embeddedGoMod = "go.mod.txt"

// outName returns the name a template file has in a project.
func outName(rel string) string {
	if path.Base(rel) == embeddedGoMod {
		return path.Join(path.Dir(rel), "go.mod")
	}
	return rel
}

// Entry describes one built-in template.
type Entry struct {
	Name        string `json:"name"`
	Language    string `json:"language"`
	Description string `json:"description"`
	// Default marks the template used for its language when the user has not chosen one.
	Default bool `json:"default"`
}

var (
	once    sync.Once
	catalog []Entry
	loadErr error
)

func load() {
	data, err := files.ReadFile(root + "/catalog.json")
	if err != nil {
		loadErr = err
		return
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		loadErr = fmt.Errorf("built-in catalog is damaged: %w", err)
		return
	}
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Name < catalog[j].Name })
}

// Catalog returns the built-in templates sorted by name.
func Catalog() []Entry {
	once.Do(load)
	if loadErr != nil {
		return nil
	}
	return append([]Entry(nil), catalog...)
}

// Get finds a built-in template by name (exact, then case-insensitive).
func Get(name string) (Entry, bool) {
	list := Catalog()
	for _, e := range list {
		if e.Name == name {
			return e, true
		}
	}
	for _, e := range list {
		if strings.EqualFold(e.Name, name) {
			return e, true
		}
	}
	return Entry{}, false
}

// DefaultFor returns the built-in default template of a language (case-insensitive).
func DefaultFor(language string) (Entry, bool) {
	for _, e := range Catalog() {
		if e.Default && strings.EqualFold(e.Language, language) {
			return e, true
		}
	}
	return Entry{}, false
}

// Files lists the files of a built-in template (relative, slash separated, sorted).
func Files(name string) ([]string, error) {
	e, ok := Get(name)
	if !ok {
		return nil, fmt.Errorf("no built-in template %q", name)
	}
	var out []string
	err := fs.WalkDir(files, path.Join(root, e.Name), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, outName(strings.TrimPrefix(p, path.Join(root, e.Name)+"/")))
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// Extract writes a built-in template to dest, which must not exist or must be empty.
func Extract(name, dest string) error {
	e, ok := Get(name)
	if !ok {
		return fmt.Errorf("no built-in template %q", name)
	}
	if entries, err := os.ReadDir(dest); err == nil && len(entries) > 0 {
		return fmt.Errorf("directory %q is not empty", dest)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	base := path.Join(root, e.Name)
	return fs.WalkDir(files, base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, base), "/")
		target := filepath.Join(dest, filepath.FromSlash(outName(rel)))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := files.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
}
