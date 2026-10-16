// Package detect finds the languages, package managers and developer tools installed on this machine.
package detect

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
)

// Tool is a thing Foundry can look for.
type Tool struct {
	Name   string // display name, e.g. "Node.js"
	Binary string // executable looked up on PATH
}

// Catalog lists what is detected, per category.
var Catalog = struct{ Languages, PackageManagers, DevTools []Tool }{
	Languages: []Tool{
		{"Go", "go"}, {"Python", "python3"}, {"Node.js", "node"}, {"Rust", "rustc"}, {"Java", "javac"},
		{"C++", "g++"}, {"PHP", "php"}, {"Ruby", "ruby"}, {"Swift", "swift"}, {"Kotlin", "kotlinc"},
		{"C#", "dotnet"}, {"C", "gcc"}, {"TypeScript", "tsc"},
	},
	PackageManagers: []Tool{
		{"pip", "pip3"}, {"npm", "npm"}, {"yarn", "yarn"}, {"pnpm", "pnpm"}, {"cargo", "cargo"},
		{"maven", "mvn"}, {"gradle", "gradle"}, {"composer", "composer"}, {"make", "make"},
		{"cmake", "cmake"}, {"bundler", "bundle"}, {"brew", "brew"}, {"apt", "apt"},
	},
	DevTools: []Tool{
		{"git", "git"}, {"docker", "docker"}, {"kubectl", "kubectl"}, {"nginx", "nginx"},
		{"terraform", "terraform"}, {"ansible", "ansible"}, {"sqlite3", "sqlite3"}, {"mysql", "mysql"},
		{"psql", "psql"},
	},
}

// Result is what a scan found. The slices are sorted and hold only what is installed.
type Result struct {
	Languages       []string `json:"languages"`
	PackageManagers []string `json:"package_managers"`
	DevTools        []string `json:"dev_tools"`
	VSCodePath      string   `json:"vscode_path,omitempty"`
}

// Finder abstracts the machine so a scan can be tested.
type Finder struct {
	LookPath func(name string) (string, error)
	Stat     func(name string) (os.FileInfo, error)
	Getenv   func(key string) string
	GOOS     string
}

// System returns a Finder for the real machine.
func System() Finder {
	return Finder{LookPath: exec.LookPath, Stat: os.Stat, Getenv: os.Getenv, GOOS: runtime.GOOS}
}

func (f Finder) found(binary string) bool {
	_, err := f.LookPath(binary)
	return err == nil
}

func (f Finder) exists(path string) bool {
	_, err := f.Stat(path)
	return err == nil
}

// VSCode returns the path of the VS Code command line launcher, or "".
func (f Finder) VSCode() string {
	for _, name := range []string{"code", "code.cmd"} {
		if p, err := f.LookPath(name); err == nil {
			return p
		}
	}
	var candidates []string
	switch f.GOOS {
	case "windows":
		if local := f.Getenv("LOCALAPPDATA"); local != "" {
			candidates = append(candidates, filepath.Join(local, "Programs", "Microsoft VS Code", "bin", "code.cmd"))
		}
		if pf := f.Getenv("ProgramFiles"); pf != "" {
			candidates = append(candidates, filepath.Join(pf, "Microsoft VS Code", "bin", "code.cmd"))
		}
	case "darwin":
		candidates = []string{"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"}
	default:
		candidates = []string{"/usr/bin/code", "/usr/local/bin/code", "/snap/bin/code", "/usr/share/code/bin/code"}
	}
	for _, c := range candidates {
		if f.exists(c) {
			return c
		}
	}
	return ""
}

func (f Finder) collect(tools []Tool) []string {
	out := []string{}
	for _, t := range tools {
		if f.found(t.Binary) {
			out = append(out, t.Name)
		}
	}
	sort.Strings(out)
	return out
}

// Scan inspects the machine.
func (f Finder) Scan() Result {
	r := Result{
		Languages:       f.collect(Catalog.Languages),
		PackageManagers: f.collect(Catalog.PackageManagers),
		DevTools:        f.collect(Catalog.DevTools),
		VSCodePath:      f.VSCode(),
	}
	return r
}

// Names returns every catalog entry for a category with whether it was found, in catalog order.
func Names(tools []Tool, found []string) []Entry {
	set := map[string]bool{}
	for _, n := range found {
		set[n] = true
	}
	out := make([]Entry, 0, len(tools))
	for _, t := range tools {
		out = append(out, Entry{Name: t.Name, Found: set[t.Name]})
	}
	return out
}

// Entry is one catalog item and whether it was found.
type Entry struct {
	Name  string
	Found bool
}

// Apply stores a scan result in the config (it does not save the file).
func Apply(cfg *config.Config, r Result) {
	cfg.InstalledLanguages = r.Languages
	cfg.InstalledPackageManagers = r.PackageManagers
	cfg.InstalledDevTools = r.DevTools
	if r.VSCodePath != "" {
		cfg.VSCodePath = r.VSCodePath
	}
}
