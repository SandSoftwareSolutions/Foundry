package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/fatih/color"
)

// harness runs the real command tree against a throw-away config file and working directory.
type harness struct {
	t      *testing.T
	root   string // temp dir for everything
	config string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	color.NoColor = true
	root := t.TempDir()
	h := &harness{t: t, root: root, config: filepath.Join(root, "cfg", "config.yaml")}
	t.Cleanup(func() { config.SetPathOverride("") })
	home := filepath.Join(root, "home")
	_ = os.MkdirAll(home, 0o755)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("NO_COLOR", "1")
	return h
}

func (h *harness) run(args ...string) (stdout, stderr string, err error) {
	h.t.Helper()
	cmd := NewRootCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"--config", h.config}, args...))
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	out, errOut, err := h.run(args...)
	if err != nil {
		h.t.Fatalf("foundry %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, out, errOut)
	}
	return out
}

func (h *harness) writeTemplate(name string, files map[string]string) string {
	h.t.Helper()
	dir := filepath.Join(h.root, "templates", name)
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			h.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			h.t.Fatal(err)
		}
	}
	return dir
}

func goTemplate(h *harness) string {
	return h.writeTemplate("go-cli", map[string]string{
		"go.mod":                     "module example.com/{{PROJECT_NAME}}\n\ngo 1.22\n",
		"main.go":                    "package main\n\n// {{PROJECT_NAME}} by {{AUTHOR}} port {{PORT}}\nfunc main() {}\n",
		"README.md":                  "# {{PROJECT_NAME_PASCAL}}\n",
		"{{PROJECT_NAME_SNAKE}}.txt": "named after the project\n",
		".foundryignore":             "*.skip\nsecrets/\n",
		"notes.skip":                 "x",
		"secrets/key.txt":            "x",
		"node_modules/pkg/index.js":  "x",
	})
}

func TestFirstRunCreatesConfigAndStaysQuietAfterwards(t *testing.T) {
	h := newHarness(t)
	_, errOut, err := h.run("config", "--view")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut, "First run: created") {
		t.Fatalf("expected a first-run notice, got %q", errOut)
	}
	if _, err := os.Stat(h.config); err != nil {
		t.Fatalf("config not created: %v", err)
	}
	_, errOut, _ = h.run("config", "--view")
	if strings.Contains(errOut, "First run") {
		t.Fatalf("second run must be quiet, got %q", errOut)
	}
}

func TestVersionAndHelpDoNotCreateConfig(t *testing.T) {
	h := newHarness(t)
	h.mustRun("version")
	h.mustRun("--help")
	if _, err := os.Stat(h.config); err == nil {
		t.Fatal("version/help must not create the config file")
	}
}

func TestTemplateLifecycle(t *testing.T) {
	h := newHarness(t)
	dir := goTemplate(h)

	out := h.mustRun("template", "add", "go-cli", dir, "--description", "Go starter")
	if !strings.Contains(out, `Template "go-cli" saved`) || !strings.Contains(out, "Go") {
		t.Fatalf("add output: %s", out)
	}
	out = h.mustRun("template", "list", "--quiet", "--saved")
	if strings.TrimSpace(out) != "go-cli" {
		t.Fatalf("list --quiet: %q", out)
	}
	out = h.mustRun("template", "show", "go-cli", "--json")
	var shown config.Template
	if err := json.Unmarshal([]byte(out), &shown); err != nil {
		t.Fatalf("show --json: %v\n%s", err, out)
	}
	for _, f := range shown.Files {
		if strings.Contains(f, "secrets") || strings.HasSuffix(f, ".skip") || strings.Contains(f, "node_modules") {
			t.Fatalf("ignored file listed: %s", f)
		}
	}
	out = h.mustRun("template", "vars", "go-cli")
	if !strings.Contains(out, "PORT") || !strings.Contains(out, "needs --var PORT=VALUE") || !strings.Contains(out, "PROJECT_NAME") {
		t.Fatalf("vars output: %s", out)
	}
	h.mustRun("template", "edit", "go-cli", "--rename", "gocli", "--description", "renamed")
	out = h.mustRun("template", "list", "--json", "--saved")
	if !strings.Contains(out, `"gocli"`) || strings.Contains(out, `"go-cli"`) {
		t.Fatalf("rename not applied: %s", out)
	}
	h.mustRun("template", "remove", "gocli")
	if out := h.mustRun("template", "list", "--quiet", "--saved"); strings.TrimSpace(out) != "" {
		t.Fatalf("template still listed: %q", out)
	}
}

func TestNewCreatesProjectFromTemplate(t *testing.T) {
	h := newHarness(t)
	h.mustRun("config", "--user", "Kaj", "--license", "MIT")
	h.mustRun("template", "add", "go-cli", goTemplate(h))
	parent := filepath.Join(h.root, "work")

	out := h.mustRun("new", "my-api", "--template", "go-cli", "--path", parent, "--var", "PORT=8080", "--no-git", "--no-post")
	if !strings.Contains(out, "Created") {
		t.Fatalf("output: %s", out)
	}
	proj := filepath.Join(parent, "my-api")
	main, _ := os.ReadFile(filepath.Join(proj, "main.go"))
	if !strings.Contains(string(main), "my-api by Kaj port 8080") {
		t.Fatalf("main.go: %q", main)
	}
	mod, _ := os.ReadFile(filepath.Join(proj, "go.mod"))
	if !strings.Contains(string(mod), "module example.com/my-api") {
		t.Fatalf("go.mod: %q", mod)
	}
	if b, _ := os.ReadFile(filepath.Join(proj, "README.md")); string(b) != "# MyApi\n" {
		t.Fatalf("README: %q", b)
	}
	for _, rel := range []string{"my_api.txt"} {
		if _, err := os.Stat(filepath.Join(proj, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	for _, rel := range []string{".foundryignore", "notes.skip", "secrets", "node_modules", ".git"} {
		if _, err := os.Stat(filepath.Join(proj, rel)); err == nil {
			t.Fatalf("%s must not be in the project", rel)
		}
	}
}

func TestNewByLanguageUsesTheDefaultTemplate(t *testing.T) {
	h := newHarness(t)
	h.mustRun("template", "add", "go-cli", goTemplate(h))
	// No default set: the built-in default of the language is used.
	out := h.mustRun("new", "x", "--language", "Go", "--path", h.root, "--dry-run")
	if !strings.Contains(out, "go-basic (built-in)") {
		t.Fatalf("expected the built-in Go template, got: %s", out)
	}
	// Your own default wins.
	h.mustRun("config", "Go", "go-cli")
	out = h.mustRun("new", "x", "--language", "go", "--path", h.root, "--dry-run")
	if !strings.Contains(out, "Dry run") || !strings.Contains(out, "go-cli") || strings.Contains(out, "built-in") {
		t.Fatalf("output: %s", out)
	}
	if _, err := os.Stat(filepath.Join(h.root, "x")); err == nil {
		t.Fatal("dry run wrote files")
	}
	if _, _, err := h.run("new", "x", "--language", "Cobol", "--path", h.root, "--dry-run"); err == nil {
		t.Fatal("a language without any template must fail")
	}
}

func TestDryRunJSON(t *testing.T) {
	h := newHarness(t)
	h.mustRun("template", "add", "go-cli", goTemplate(h))
	out := h.mustRun("new", "demo", "--template", "go-cli", "--path", h.root, "--dry-run", "--json")
	var report dryRunReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if report.Template != "go-cli" || len(report.Files) == 0 || report.Unresolved["main.go"][0] != "PORT" {
		t.Fatalf("report: %+v", report)
	}
}

func TestNewRefusesBadInput(t *testing.T) {
	h := newHarness(t)
	h.mustRun("template", "add", "go-cli", goTemplate(h))
	cases := map[string][]string{
		"path in name":        {"new", "../evil", "--template", "go-cli"},
		"two sources":         {"new", "x", "--template", "go-cli", "--language", "Go"},
		"unknown template":    {"new", "x", "--template", "nope", "--path", h.root},
		"bad var":             {"new", "x", "--template", "go-cli", "--var", "NOEQUALS", "--path", h.root},
		"bad var name":        {"new", "x", "--template", "go-cli", "--var", "1A=b", "--path", h.root},
		"json without dryrun": {"new", "x", "--template", "go-cli", "--json", "--path", h.root},
	}
	for name, args := range cases {
		if _, _, err := h.run(args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if entries, _ := os.ReadDir(h.root); len(entries) > 0 {
		for _, e := range entries {
			if e.Name() == "evil" || e.Name() == "x" {
				t.Fatalf("a failed run left %s behind", e.Name())
			}
		}
	}
}

func TestNewRefusesExistingNonEmptyDirectory(t *testing.T) {
	h := newHarness(t)
	h.mustRun("template", "add", "go-cli", goTemplate(h))
	target := filepath.Join(h.root, "taken")
	_ = os.MkdirAll(target, 0o755)
	_ = os.WriteFile(filepath.Join(target, "keep.txt"), []byte("mine"), 0o644)
	if _, _, err := h.run("new", "taken", "--template", "go-cli", "--path", h.root, "--no-git", "--no-post"); err == nil {
		t.Fatal("expected an error")
	}
	if b, _ := os.ReadFile(filepath.Join(target, "keep.txt")); string(b) != "mine" {
		t.Fatal("existing content was touched")
	}
}

func TestRemoveDefaultTemplateNeedsForce(t *testing.T) {
	h := newHarness(t)
	h.mustRun("template", "add", "go-cli", goTemplate(h))
	h.mustRun("config", "Go", "go-cli")
	if _, _, err := h.run("template", "remove", "go-cli"); err == nil {
		t.Fatal("expected an error without --force")
	}
	h.mustRun("template", "remove", "go-cli", "--force")
	cfg, _ := config.Load()
	if len(cfg.LanguageDefaults) != 0 {
		t.Fatalf("dangling default left: %v", cfg.LanguageDefaults)
	}
}

func TestConfigSettersAndJSON(t *testing.T) {
	h := newHarness(t)
	h.mustRun("config", "--user", "Jane", "--docker", "--interactive=false")
	out := h.mustRun("config", "--json")
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if v["author"] != "Jane" || v["docker"] != true || v["interactive"] != false {
		t.Fatalf("config = %v", v)
	}
	if out := strings.TrimSpace(h.mustRun("config", "--path")); out != h.config {
		t.Fatalf("path = %q, want %q", out, h.config)
	}
}

func TestDetectJSONAndSave(t *testing.T) {
	h := newHarness(t)
	out := h.mustRun("detect", "--json", "--yes")
	var r struct {
		Languages []string `json:"languages"`
		DevTools  []string `json:"dev_tools"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	cfg, _ := config.Load()
	if strings.Join(cfg.InstalledLanguages, ",") != strings.Join(r.Languages, ",") {
		t.Fatalf("not saved: %v vs %v", cfg.InstalledLanguages, r.Languages)
	}
}

func TestNewFromGitRepositoryAndInitialCommit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git is not installed")
	}
	h := newHarness(t)
	h.mustRun("config", "--user", "Kaj")
	repo := h.writeTemplate("remote", map[string]string{
		"main.go": "package main // {{PROJECT_NAME}}\n",
		"go.mod":  "module {{PROJECT_NAME}}\n",
	})
	if _, err := gitInitCommit(repo); err != nil {
		t.Fatal(err)
	}
	out := h.mustRun("new", "from-git", "--git", repo, "--path", h.root, "--no-post")
	if !strings.Contains(out, "first commit") {
		t.Fatalf("output: %s", out)
	}
	proj := filepath.Join(h.root, "from-git")
	if b, _ := os.ReadFile(filepath.Join(proj, "main.go")); string(b) != "package main // from-git\n" {
		t.Fatalf("main.go: %q", b)
	}
	if _, err := os.Stat(filepath.Join(proj, ".git")); err != nil {
		t.Fatal("the new project must have its own git repository")
	}
	log, err := gitOutput(proj, "log", "--format=%s")
	if err != nil || strings.TrimSpace(log) != "Initial commit from Foundry" {
		t.Fatalf("log = %q, %v", log, err)
	}
	if n, _ := gitOutput(proj, "rev-list", "--count", "HEAD"); strings.TrimSpace(n) != "1" {
		t.Fatalf("the history of the template must not be inherited, commits = %q", n)
	}
}

func TestEveryBuiltInTemplateCreatesACleanProject(t *testing.T) {
	h := newHarness(t)
	var listed []struct {
		Name    string `json:"name"`
		Builtin bool   `json:"builtin"`
	}
	if err := json.Unmarshal([]byte(h.mustRun("template", "list", "--builtin", "--json")), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) < 8 {
		t.Fatalf("expected at least 8 built-in templates, got %d", len(listed))
	}
	for _, tpl := range listed {
		t.Run(tpl.Name, func(t *testing.T) {
			if !tpl.Builtin {
				t.Fatal("not marked built-in")
			}
			out := h.mustRun("new", "My Demo-App", "--template", tpl.Name, "--path", h.root, "--dry-run", "--json")
			var report dryRunReport
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatalf("not JSON: %v\n%s", err, out)
			}
			if len(report.Files) == 0 {
				t.Fatal("no files planned")
			}
			if len(report.Unresolved) != 0 {
				t.Fatalf("placeholders without a value: %v", report.Unresolved)
			}
			for _, f := range report.Files {
				if strings.Contains(f, "{{") || strings.HasSuffix(f, ".txt") && strings.HasPrefix(f, "go.mod") {
					t.Fatalf("unexpected file name %q", f)
				}
			}
		})
	}
}

func TestBuiltInTemplateIsWrittenWithPlaceholdersFilledIn(t *testing.T) {
	h := newHarness(t)
	h.mustRun("config", "--user", "Ada")
	h.mustRun("new", "hello-tool", "--template", "go-basic", "--path", h.root, "--no-git", "--no-post")
	proj := filepath.Join(h.root, "hello-tool")
	mod, err := os.ReadFile(filepath.Join(proj, "go.mod"))
	if err != nil || !strings.Contains(string(mod), "module example.com/hello-tool") {
		t.Fatalf("go.mod: %q %v", mod, err)
	}
	readme, _ := os.ReadFile(filepath.Join(proj, "README.md"))
	if !strings.Contains(string(readme), "hello-tool") || !strings.Contains(string(readme), "Ada") || strings.Contains(string(readme), "{{") {
		t.Fatalf("README: %q", readme)
	}
	py := h.mustRun("new", "My Py Tool", "--template", "python-basic", "--path", h.root, "--no-git", "--no-post")
	if !strings.Contains(py, "built-in") {
		t.Fatalf("output: %s", py)
	}
	if _, err := os.Stat(filepath.Join(h.root, "My Py Tool", "src", "my_py_tool", "cli.py")); err != nil {
		t.Fatalf("package directory from a file-name placeholder: %v", err)
	}
	entries, _ := os.ReadDir(os.TempDir())
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "foundry-builtin-") {
			t.Fatalf("temporary directory left behind: %s", e.Name())
		}
	}
}

func TestBuiltInTemplatesInListShowVarsAndExport(t *testing.T) {
	h := newHarness(t)
	out := h.mustRun("template", "list")
	if !strings.Contains(out, "Built-in templates") || !strings.Contains(out, "go-basic") || !strings.Contains(out, "python-http") {
		t.Fatalf("list: %s", out)
	}
	if out := h.mustRun("template", "show", "node-http", "--files-only"); !strings.Contains(out, "server.js") {
		t.Fatalf("show: %s", out)
	}
	if out := h.mustRun("template", "vars", "static-site"); !strings.Contains(out, "PROJECT_NAME") || strings.Contains(out, "needs --var") {
		t.Fatalf("vars: %s", out)
	}
	dest := filepath.Join(h.root, "copy")
	h.mustRun("template", "export", "go-http", dest)
	if _, err := os.Stat(filepath.Join(dest, "go.mod")); err != nil {
		t.Fatal("exported go.mod missing:", err)
	}
	if _, _, err := h.run("template", "export", "go-http", dest); err == nil {
		t.Fatal("export into a non-empty folder must fail")
	}
	// The exported copy can be saved as one's own template and then shadows nothing.
	h.mustRun("template", "add", "my-http", dest, "--language", "Go")
	if out := h.mustRun("template", "list", "--saved", "--quiet"); strings.TrimSpace(out) != "my-http" {
		t.Fatalf("saved: %q", out)
	}
}

func TestBuiltInTemplatesCannotBeChanged(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{
		{"template", "remove", "go-basic"},
		{"template", "edit", "go-basic", "--description", "x"},
		{"template", "update", "go-basic"},
	} {
		_, _, err := h.run(args...)
		if err == nil || !strings.Contains(err.Error(), "built-in") || !strings.Contains(err.Error(), "template export") {
			t.Errorf("%v: expected a helpful built-in error, got %v", args, err)
		}
	}
}

func TestBuiltInTemplateCanBeTheLanguageDefault(t *testing.T) {
	h := newHarness(t)
	h.mustRun("config", "go", "go-http")
	cfg, _ := config.Load()
	if cfg.LanguageDefaults["go"] != "go-http" && cfg.LanguageDefaults["Go"] != "go-http" {
		t.Fatalf("defaults: %v", cfg.LanguageDefaults)
	}
	out := h.mustRun("new", "svc", "--language", "Go", "--path", h.root, "--dry-run")
	if !strings.Contains(out, "go-http (built-in)") {
		t.Fatalf("output: %s", out)
	}
	if _, _, err := h.run("config", "Go", "does-not-exist"); err == nil {
		t.Fatal("unknown template must fail")
	}
}

func TestSavedTemplateShadowsBuiltInWithTheSameName(t *testing.T) {
	h := newHarness(t)
	h.mustRun("template", "add", "go-basic", goTemplate(h))
	out := h.mustRun("new", "x", "--template", "go-basic", "--path", h.root, "--dry-run")
	if strings.Contains(out, "built-in") {
		t.Fatalf("the saved template must win: %s", out)
	}
	list := h.mustRun("template", "list", "--quiet")
	if strings.Count(list, "go-basic") != 1 {
		t.Fatalf("the name must be listed once: %q", list)
	}
}
