// Package template inspects template directories: it detects their language and lists their files.
package template

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/ignore"
)

// IgnoreFile is the name of the file that excludes paths from a template.
const IgnoreFile = ".foundryignore"

// DefaultIgnores are always excluded from templates. A rule in .foundryignore can re-include them,
// for example "!build/".
var DefaultIgnores = []string{".git/", "node_modules/", ".venv/", "__pycache__/", "vendor/", "dist/", "build/", IgnoreFile}

var extensionLanguage = map[string]string{
	".go": "Go", ".py": "Python", ".js": "JavaScript", ".mjs": "JavaScript", ".cjs": "JavaScript",
	".ts": "TypeScript", ".jsx": "React", ".tsx": "React", ".rs": "Rust", ".java": "Java", ".kt": "Kotlin",
	".cpp": "C++", ".cc": "C++", ".hpp": "C++", ".c": "C", ".h": "C", ".cs": "C#", ".php": "PHP",
	".rb": "Ruby", ".swift": "Swift", ".vue": "Vue",
}

// fileLanguage maps well-known file names to a language. They weigh more than a single source file.
var fileLanguage = map[string]string{
	"package.json": "JavaScript", "tsconfig.json": "TypeScript", "Cargo.toml": "Rust", "pom.xml": "Java",
	"build.gradle": "Java", "build.gradle.kts": "Kotlin", "Gemfile": "Ruby", "composer.json": "PHP",
	"requirements.txt": "Python", "pyproject.toml": "Python", "Pipfile": "Python", "go.mod": "Go",
	"Package.swift": "Swift",
}

// Matcher returns the ignore rules of a template directory: defaults first, then its .foundryignore.
func Matcher(dir string) (*ignore.Matcher, error) {
	user, err := ignore.Load(dir, IgnoreFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", IgnoreFile, err)
	}
	return user.Prepend(DefaultIgnores...), nil
}

// DetectLanguage returns the most likely language of the files in dir, or "Unknown".
// Ties are broken alphabetically so the result is stable.
func DetectLanguage(dir string) (string, error) {
	m, err := Matcher(dir)
	if err != nil {
		return "", err
	}
	counts := map[string]int{}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "." {
			return nil
		}
		if m.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if lang, ok := fileLanguage[d.Name()]; ok {
			counts[lang] += 5
			return nil
		}
		if lang, ok := extensionLanguage[strings.ToLower(filepath.Ext(d.Name()))]; ok {
			counts[lang]++
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("cannot scan %s: %w", dir, err)
	}
	best, bestN := "Unknown", 0
	langs := make([]string, 0, len(counts))
	for l := range counts {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	for _, l := range langs {
		if counts[l] > bestN {
			best, bestN = l, counts[l]
		}
	}
	return best, nil
}

// Info is the result of scanning a template directory.
type Info struct {
	Path     string
	Language string
	Files    []string // relative, slash separated, sorted
}

// Scan inspects dir: it resolves the absolute path, detects the language and lists the files that a
// project created from it would receive.
func Scan(dir string) (*Info, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve %s: %w", dir, err)
	}
	st, err := os.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("template directory does not exist: %s", abs)
	}
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
	}
	lang, err := DetectLanguage(abs)
	if err != nil {
		return nil, err
	}
	files, err := ListFiles(abs)
	if err != nil {
		return nil, err
	}
	return &Info{Path: abs, Language: lang, Files: files}, nil
}

// ListFiles returns the regular files of dir that are not ignored, relative and slash separated.
func ListFiles(dir string) ([]string, error) {
	m, err := Matcher(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "." {
			return nil
		}
		if m.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list %s: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}

// ValidateName checks a template name: it must be non-empty and safe to use as a file name.
func ValidateName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("template name cannot be empty")
	case name != strings.TrimSpace(name):
		return errors.New("template name cannot start or end with spaces")
	case strings.ContainsAny(name, `/\:*?"<>|`):
		return errors.New(`template name cannot contain any of / \ : * ? " < > |`)
	case name == "." || name == "..":
		return errors.New("template name cannot be . or ..")
	}
	return nil
}
