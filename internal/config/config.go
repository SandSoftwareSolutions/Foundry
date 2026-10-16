// Package config loads and saves Foundry's settings and saved templates.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template is a saved project template.
type Template struct {
	Name        string   `yaml:"name" json:"name"`
	Path        string   `yaml:"path" json:"path"`
	Language    string   `yaml:"language" json:"language"`
	Description string   `yaml:"description" json:"description"`
	Files       []string `yaml:"files,omitempty" json:"files,omitempty"`
}

// Config is the content of ~/.foundry/config.yaml.
type Config struct {
	Author          string `yaml:"author"`
	License         string `yaml:"license"`
	DefaultLanguage string `yaml:"default_language"`
	Docker          bool   `yaml:"docker"`
	Interactive     bool   `yaml:"interactive"`

	// Tools detected on this machine.
	InstalledLanguages       []string `yaml:"installed_languages"`
	InstalledPackageManagers []string `yaml:"installed_package_managers"`
	InstalledDevTools        []string `yaml:"installed_dev_tools"`
	VSCodePath               string   `yaml:"vscode_path,omitempty"`

	Templates []Template `yaml:"templates,omitempty"`

	// LanguageDefaults maps a language to the name of its default template.
	LanguageDefaults map[string]string `yaml:"language_defaults,omitempty"`
}

// Default returns the configuration used when no file exists yet.
func Default() *Config {
	return &Config{
		License:                  "MIT",
		Interactive:              true,
		InstalledLanguages:       []string{},
		InstalledPackageManagers: []string{},
		InstalledDevTools:        []string{},
		Templates:                []Template{},
		LanguageDefaults:         map[string]string{},
	}
}

var pathOverride string

// SetPathOverride makes Foundry use the given config file instead of ~/.foundry/config.yaml.
// An empty string restores the default.
func SetPathOverride(p string) { pathOverride = p }

// Path returns the config file location. It does not create anything.
func Path() (string, error) {
	if pathOverride != "" {
		return filepath.Abs(pathOverride)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".foundry", "config.yaml"), nil
}

// Exists reports whether the config file is present.
func Exists() bool {
	p, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Load reads the config file. A missing file gives Default().
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	cfg := Default()
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config %s: %w", p, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config %s: %w", p, err)
	}
	if cfg.LanguageDefaults == nil {
		cfg.LanguageDefaults = map[string]string{}
	}
	return cfg, nil
}

// Save writes the config atomically: it writes a temporary file next to the target and renames it,
// so a crash or a full disk never leaves a half-written config.
func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("cannot create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("cannot write config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("cannot flush config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		cleanup()
		return fmt.Errorf("cannot replace config: %w", err)
	}
	return nil
}

// Keys lists the settings that Set and Get understand, in display order.
var Keys = []string{"author", "license", "default_language", "docker", "interactive", "vscode_path"}

// Set changes one setting from its text form (booleans as true/false/yes/no/on/off/1/0).
func (c *Config) Set(key, value string) error {
	switch key {
	case "author":
		c.Author = value
	case "license":
		c.License = value
	case "default_language":
		c.DefaultLanguage = value
	case "vscode_path":
		c.VSCodePath = value
	case "docker", "interactive":
		b, err := ParseBool(value)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if key == "docker" {
			c.Docker = b
		} else {
			c.Interactive = b
		}
	default:
		return fmt.Errorf("unknown setting %q (known: %s)", key, strings.Join(Keys, ", "))
	}
	return nil
}

// Get returns one setting in text form.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "author":
		return c.Author, nil
	case "license":
		return c.License, nil
	case "default_language":
		return c.DefaultLanguage, nil
	case "vscode_path":
		return c.VSCodePath, nil
	case "docker":
		return strconv.FormatBool(c.Docker), nil
	case "interactive":
		return strconv.FormatBool(c.Interactive), nil
	}
	return "", fmt.Errorf("unknown setting %q (known: %s)", key, strings.Join(Keys, ", "))
}

// ParseBool accepts the usual spellings of true and false.
func ParseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "y", "yes", "on":
		return true, nil
	case "0", "f", "false", "n", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a yes/no value", s)
}

// HasTool reports whether a detected dev tool (for example "git") is recorded.
func (c *Config) HasTool(name string) bool {
	for _, t := range c.InstalledDevTools {
		if t == name {
			return true
		}
	}
	return false
}

// FindTemplate returns the saved template with the given name (exact match, then case-insensitive).
func (c *Config) FindTemplate(name string) (*Template, bool) {
	for i := range c.Templates {
		if c.Templates[i].Name == name {
			return &c.Templates[i], true
		}
	}
	for i := range c.Templates {
		if strings.EqualFold(c.Templates[i].Name, name) {
			return &c.Templates[i], true
		}
	}
	return nil, false
}

// UpsertTemplate adds a template or replaces the one with the same name.
func (c *Config) UpsertTemplate(t Template) {
	for i := range c.Templates {
		if c.Templates[i].Name == t.Name {
			c.Templates[i] = t
			return
		}
	}
	c.Templates = append(c.Templates, t)
}

// RemoveTemplate deletes a template and any language default that pointed at it.
// It returns the languages whose default was cleared.
func (c *Config) RemoveTemplate(name string) ([]string, error) {
	idx := -1
	for i := range c.Templates {
		if c.Templates[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("template %q not found", name)
	}
	c.Templates = append(c.Templates[:idx], c.Templates[idx+1:]...)
	cleared := c.DefaultFor(name)
	for _, lang := range cleared {
		delete(c.LanguageDefaults, lang)
	}
	return cleared, nil
}

// DefaultFor lists the languages for which the named template is the default, sorted.
func (c *Config) DefaultFor(templateName string) []string {
	var langs []string
	for lang, t := range c.LanguageDefaults {
		if t == templateName {
			langs = append(langs, lang)
		}
	}
	sort.Strings(langs)
	return langs
}

// LanguageDefault returns the default template name for a language (case-insensitive language match).
func (c *Config) LanguageDefault(language string) string {
	if t, ok := c.LanguageDefaults[language]; ok {
		return t
	}
	for lang, t := range c.LanguageDefaults {
		if strings.EqualFold(lang, language) {
			return t
		}
	}
	return ""
}

// SetLanguageDefault points a language at a saved template. An empty template name clears it.
func (c *Config) SetLanguageDefault(language, templateName string) error {
	if language == "" {
		return errors.New("language cannot be empty")
	}
	if c.LanguageDefaults == nil {
		c.LanguageDefaults = map[string]string{}
	}
	// Reuse the existing spelling of the language key so "go" and "Go" do not both exist.
	key := language
	for lang := range c.LanguageDefaults {
		if strings.EqualFold(lang, language) {
			key = lang
			break
		}
	}
	if templateName == "" {
		delete(c.LanguageDefaults, key)
		return nil
	}
	t, ok := c.FindTemplate(templateName)
	if !ok {
		return fmt.Errorf("template %q not found", templateName)
	}
	c.LanguageDefaults[key] = t.Name
	return nil
}

// Languages returns the sorted, de-duplicated languages of the saved templates.
func (c *Config) Languages() []string {
	set := map[string]struct{}{}
	for _, t := range c.Templates {
		if t.Language != "" {
			set[t.Language] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for l := range set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}
