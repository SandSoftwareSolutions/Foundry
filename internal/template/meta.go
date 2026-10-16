package template

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MetaFile is an optional file at the root of a template that says what to do after a project was created
// from it. It is never copied into the project.
const MetaFile = ".foundry.yaml"

// VarSpec declares a placeholder a template needs, so Foundry can ask for it (or use the default when it is
// not allowed to ask).
type VarSpec struct {
	Name    string `yaml:"name"`
	Prompt  string `yaml:"prompt"`
	Default string `yaml:"default"`
}

// Meta is the content of MetaFile.
//
//	vars:            # placeholders to ask for
//	  - name: PORT
//	    prompt: Port the service listens on
//	    default: "8080"
//	post:            # commands to run in the new project, one per line, no shell syntax
//	  - pnpm install
//	next:            # what to tell the user to do next (shown at the end)
//	  - pnpm dev
type Meta struct {
	Vars []VarSpec `yaml:"vars"`
	Post []string  `yaml:"post"`
	Next []string  `yaml:"next"`
}

// LoadMeta reads the template's MetaFile. A missing file gives an empty Meta.
func LoadMeta(dir string) (Meta, error) {
	data, err := os.ReadFile(filepath.Join(dir, MetaFile))
	if errors.Is(err, fs.ErrNotExist) {
		return Meta{}, nil
	}
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Meta{}, fmt.Errorf("cannot parse %s: %w", MetaFile, err)
	}
	return m, nil
}
