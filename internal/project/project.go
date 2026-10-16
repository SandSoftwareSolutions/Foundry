// Package project turns a template directory into a new project.
//
// Creation is split in two steps so that --dry-run and the real run cannot drift apart:
// BuildPlan decides what would be written (and collects warnings); Execute writes it.
package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/SandSoftwareSolutions/Foundry/internal/template"
)

const binarySniffBytes = 8000

// Options describe the project to create.
type Options struct {
	TemplateDir string // absolute or relative path of the template
	Name        string // project name, used for placeholders
	TargetDir   string // directory to create
	Author      string
	License     string
	Vars        render.Vars // custom --var values; they override built-ins
	Now         time.Time   // zero means time.Now()
}

// Kind is the type of a planned action.
type Kind int

const (
	// MkDir creates a directory.
	MkDir Kind = iota
	// WriteFile writes a (rendered) file.
	WriteFile
)

// Action is one planned filesystem change.
type Action struct {
	Kind Kind
	Src  string // absolute source path (empty for MkDir of the root)
	Dst  string // absolute destination path
	Rel  string // destination relative to the project root, slash separated
	Mode fs.FileMode
	// Text is true when placeholders are replaced in the content; binary files are copied byte for byte.
	Text bool
}

// Plan is the result of BuildPlan.
type Plan struct {
	Options  Options
	Vars     render.Vars
	Actions  []Action
	Warnings []string
	// Unresolved maps a destination path to the placeholder names in it that no variable defines.
	Unresolved map[string][]string
	// TargetExisted is true when the target directory was already there (and empty).
	TargetExisted bool
}

// Files returns the relative destination paths of the files that would be written, in order.
func (p *Plan) Files() []string {
	var out []string
	for _, a := range p.Actions {
		if a.Kind == WriteFile {
			out = append(out, a.Rel)
		}
	}
	return out
}

// ValidateName checks a project name. It must be usable as a single directory name; use --path to
// choose where the directory is created.
func ValidateName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("project name cannot be empty")
	case name != strings.TrimSpace(name):
		return errors.New("project name cannot start or end with spaces")
	case name == "." || name == "..":
		return errors.New("project name cannot be a single or double dot")
	case strings.ContainsAny(name, `/\`):
		return errors.New("project name cannot contain a path separator; use --path to choose the parent directory")
	case strings.ContainsAny(name, `:*?"<>|`+"\x00"):
		return errors.New(`project name cannot contain any of : * ? " < > |`)
	case len(name) > 100:
		return errors.New("project name is longer than 100 characters")
	case strings.HasSuffix(name, "."):
		return errors.New("project name cannot end with a dot (Windows cannot create such a folder)")
	case isWindowsReserved(name):
		return fmt.Errorf("%q is a reserved device name on Windows; choose another project name", name)
	}
	return nil
}

// isWindowsReserved reports whether name (ignoring any extension) is a Windows device name such as CON or COM1.
// Projects are meant to be portable, so the name is refused on every platform.
func isWindowsReserved(name string) bool {
	base := strings.ToUpper(name)
	if i := strings.Index(base, "."); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		return base[3] >= '1' && base[3] <= '9'
	}
	return false
}

// BuildPlan walks the template and decides what a project would contain. It writes nothing.
func BuildPlan(opts Options) (*Plan, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if err := ValidateName(opts.Name); err != nil {
		return nil, err
	}
	src, err := filepath.Abs(opts.TemplateDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve template path: %w", err)
	}
	st, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("template path no longer exists: %s", src)
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("template path is not a directory: %s", src)
	}
	dst, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve target path: %w", err)
	}
	plan := &Plan{Options: opts, Unresolved: map[string][]string{}}
	if entries, err := os.ReadDir(dst); err == nil {
		if len(entries) > 0 {
			return nil, fmt.Errorf("directory %q already exists and is not empty", opts.TargetDir)
		}
		plan.TargetExisted = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot use target %q: %w", opts.TargetDir, err)
	}
	// A target inside the template (for example the template is the current directory) is never copied into itself.
	realSrc, realDst := evalOrSelf(src), evalOrSelf(dst)
	targetInSource := isWithin(realSrc, realDst)

	matcher, err := template.Matcher(src)
	if err != nil {
		return nil, err
	}
	vars := render.Merge(render.Builtins(opts.Name, opts.Author, opts.License, opts.Now), opts.Vars)
	plan.Vars = vars

	walkErr := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if targetInSource && isWithin(realDst, evalOrSelf(p)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if matcher.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("skipped symbolic link %s (links are not copied)", filepath.ToSlash(rel)))
			return nil
		}
		if !d.IsDir() && !d.Type().IsRegular() {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("skipped %s (not a regular file)", filepath.ToSlash(rel)))
			return nil
		}
		destRel, err := renderPath(rel, vars)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if names := render.Unresolved(destRel, vars); len(names) > 0 {
			plan.Unresolved[destRel] = append(plan.Unresolved[destRel], names...)
		}
		a := Action{Src: p, Dst: filepath.Join(dst, filepath.FromSlash(destRel)), Rel: destRel}
		if d.IsDir() {
			a.Kind = MkDir
			a.Mode = 0o755
		} else {
			a.Kind = WriteFile
			a.Mode = info.Mode().Perm()
			if a.Mode&0o200 == 0 { // keep the copy writable so a later update or removal works
				a.Mode |= 0o200
			}
			text, names, err := inspect(p, vars)
			if err != nil {
				return err
			}
			a.Text = text
			if len(names) > 0 {
				plan.Unresolved[destRel] = append(plan.Unresolved[destRel], names...)
			}
		}
		plan.Actions = append(plan.Actions, a)
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("cannot read template: %w", walkErr)
	}
	for k := range plan.Unresolved {
		plan.Unresolved[k] = dedupe(plan.Unresolved[k])
	}
	if names := render.Unresolved(opts.Name, vars); len(names) > 0 {
		return nil, fmt.Errorf("project name contains unknown placeholder(s): %s", strings.Join(names, ", "))
	}
	return plan, nil
}

// renderPath replaces placeholders in every segment of rel. A value that would introduce a path
// separator or a parent reference is refused, so a --var cannot write outside the project.
func renderPath(rel string, vars render.Vars) (string, error) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, seg := range parts {
		out := render.Apply(seg, vars)
		if out == "" || out == "." || out == ".." || strings.ContainsAny(out, `/\`) || strings.ContainsRune(out, 0) {
			return "", fmt.Errorf("placeholder in %q would produce an unsafe file name %q", rel, out)
		}
		parts[i] = out
	}
	return strings.Join(parts, "/"), nil
}

// inspect reads a file once to decide whether it is text and which unknown placeholders it uses.
func inspect(path string, vars render.Vars) (text bool, unresolved []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if isBinary(data) {
		return false, nil, nil
	}
	return true, render.Unresolved(string(data), vars), nil
}

func isBinary(data []byte) bool {
	n := len(data)
	if n > binarySniffBytes {
		n = binarySniffBytes
	}
	return bytes.IndexByte(data[:n], 0) >= 0
}

// Execute writes the planned project. If anything fails after the target directory was created by
// this call, the partial result is removed again, so a failed run does not leave a half project.
func (p *Plan) Execute() (err error) {
	root, err := filepath.Abs(p.Options.TargetDir)
	if err != nil {
		return err
	}
	createdRoot := false
	if _, statErr := os.Stat(root); errors.Is(statErr, fs.ErrNotExist) {
		createdRoot = true
	}
	defer func() {
		if err != nil && createdRoot {
			_ = os.RemoveAll(root)
		}
	}()
	if err = os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", root, err)
	}
	for _, a := range p.Actions {
		switch a.Kind {
		case MkDir:
			if err = os.MkdirAll(a.Dst, a.Mode); err != nil {
				return fmt.Errorf("cannot create %s: %w", a.Rel, err)
			}
		case WriteFile:
			if err = os.MkdirAll(filepath.Dir(a.Dst), 0o755); err != nil {
				return fmt.Errorf("cannot create directory for %s: %w", a.Rel, err)
			}
			data, rerr := os.ReadFile(a.Src)
			if rerr != nil {
				err = fmt.Errorf("cannot read %s: %w", a.Src, rerr)
				return err
			}
			if a.Text {
				data = []byte(render.Apply(string(data), p.Vars))
			}
			if err = os.WriteFile(a.Dst, data, a.Mode); err != nil {
				return fmt.Errorf("cannot write %s: %w", a.Rel, err)
			}
		}
	}
	return nil
}

// UnresolvedSummary returns "path: NAME, NAME" lines sorted by path, for display.
func (p *Plan) UnresolvedSummary() []string {
	paths := make([]string, 0, len(p.Unresolved))
	for k := range p.Unresolved {
		paths = append(paths, k)
	}
	sort.Strings(paths)
	out := make([]string, 0, len(paths))
	for _, k := range paths {
		out = append(out, fmt.Sprintf("%s: %s", k, strings.Join(p.Unresolved[k], ", ")))
	}
	return out
}

func evalOrSelf(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	// The target may not exist yet: resolve its closest existing parent.
	dir, base := filepath.Split(filepath.Clean(p))
	if dir == "" || dir == p {
		return p
	}
	return filepath.Join(evalOrSelf(filepath.Clean(dir)), base)
}

// isWithin reports whether child is parent itself or below it.
func isWithin(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
