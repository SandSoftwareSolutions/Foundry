package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func useTempConfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sub", "config.yaml")
	SetPathOverride(p)
	t.Cleanup(func() { SetPathOverride("") })
	return p
}

func TestLoadMissingFileGivesDefaultsAndCreatesNothing(t *testing.T) {
	p := useTempConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.License != "MIT" || !cfg.Interactive || cfg.LanguageDefaults == nil {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if _, err := os.Stat(p); err == nil {
		t.Fatal("Load must not create the config file")
	}
	if Exists() {
		t.Fatal("Exists must be false")
	}
}

func TestSaveAndLoadRoundTripAndLeavesNoTempFiles(t *testing.T) {
	p := useTempConfig(t)
	cfg := Default()
	cfg.Author = "Kaj"
	cfg.UpsertTemplate(Template{Name: "go-cli", Path: "/x", Language: "Go"})
	if err := cfg.SetLanguageDefault("Go", "go-cli"); err != nil {
		t.Fatal(err)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Author != "Kaj" || len(got.Templates) != 1 || got.LanguageDefaults["Go"] != "go-cli" {
		t.Fatalf("round trip lost data: %+v", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temporary file left behind: %s", e.Name())
		}
	}
}

func TestLoadRejectsBrokenYAMLWithPath(t *testing.T) {
	p := useTempConfig(t)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("author: [unclosed"), 0o644)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("expected a parse error naming the file, got %v", err)
	}
}

func TestSetAndGetSettings(t *testing.T) {
	cfg := Default()
	for _, kv := range [][2]string{{"author", "A"}, {"license", "Apache-2.0"}, {"docker", "yes"}, {"interactive", "off"}} {
		if err := cfg.Set(kv[0], kv[1]); err != nil {
			t.Fatalf("set %s: %v", kv[0], err)
		}
	}
	if !cfg.Docker || cfg.Interactive || cfg.Author != "A" || cfg.License != "Apache-2.0" {
		t.Fatalf("unexpected: %+v", cfg)
	}
	if v, _ := cfg.Get("docker"); v != "true" {
		t.Fatalf("docker = %q", v)
	}
	if err := cfg.Set("nope", "x"); err == nil {
		t.Fatal("unknown key must fail")
	}
	if err := cfg.Set("docker", "maybe"); err == nil {
		t.Fatal("bad boolean must fail")
	}
}

func TestLanguageDefaultsAreCaseInsensitiveAndClearable(t *testing.T) {
	cfg := Default()
	cfg.UpsertTemplate(Template{Name: "Go-CLI", Language: "Go"})
	if err := cfg.SetLanguageDefault("Go", "go-cli"); err != nil { // template name matched case-insensitively
		t.Fatal(err)
	}
	if cfg.LanguageDefaults["Go"] != "Go-CLI" {
		t.Fatalf("must store the canonical template name, got %q", cfg.LanguageDefaults["Go"])
	}
	if cfg.LanguageDefault("go") != "Go-CLI" {
		t.Fatal("lookup must ignore case")
	}
	if err := cfg.SetLanguageDefault("go", ""); err != nil {
		t.Fatal(err)
	}
	if len(cfg.LanguageDefaults) != 0 {
		t.Fatalf("not cleared: %v", cfg.LanguageDefaults)
	}
	if err := cfg.SetLanguageDefault("Go", "missing"); err == nil {
		t.Fatal("unknown template must fail")
	}
}

func TestRemoveTemplateClearsDefaults(t *testing.T) {
	cfg := Default()
	cfg.UpsertTemplate(Template{Name: "a", Language: "Go"})
	cfg.UpsertTemplate(Template{Name: "b", Language: "Go"})
	_ = cfg.SetLanguageDefault("Go", "a")
	_ = cfg.SetLanguageDefault("Rust", "a")
	cleared, err := cfg.RemoveTemplate("a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cleared, []string{"Go", "Rust"}) {
		t.Fatalf("cleared = %v", cleared)
	}
	if len(cfg.LanguageDefaults) != 0 || len(cfg.Templates) != 1 {
		t.Fatalf("unexpected state: %+v", cfg)
	}
	if _, err := cfg.RemoveTemplate("a"); err == nil {
		t.Fatal("removing twice must fail")
	}
}

func TestUpsertReplacesSameName(t *testing.T) {
	cfg := Default()
	cfg.UpsertTemplate(Template{Name: "a", Description: "one"})
	cfg.UpsertTemplate(Template{Name: "a", Description: "two"})
	if len(cfg.Templates) != 1 || cfg.Templates[0].Description != "two" {
		t.Fatalf("got %+v", cfg.Templates)
	}
}
