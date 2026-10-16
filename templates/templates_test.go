package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCatalogMatchesDirectories(t *testing.T) {
	var inCatalog, onDisk []string
	for _, e := range Catalog() {
		inCatalog = append(inCatalog, e.Name)
		if e.Language == "" || e.Description == "" {
			t.Errorf("%s: language and description are required", e.Name)
		}
	}
	entries, err := fs.ReadDir(files, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range entries {
		if d.IsDir() {
			onDisk = append(onDisk, d.Name())
		}
	}
	sort.Strings(inCatalog)
	sort.Strings(onDisk)
	if strings.Join(inCatalog, ",") != strings.Join(onDisk, ",") {
		t.Fatalf("catalog %v does not match directories %v", inCatalog, onDisk)
	}
	if len(inCatalog) < 8 {
		t.Fatalf("expected a useful set of templates, got %d", len(inCatalog))
	}
}

func TestOneDefaultPerLanguageAtMost(t *testing.T) {
	seen := map[string]string{}
	for _, e := range Catalog() {
		if !e.Default {
			continue
		}
		lang := strings.ToLower(e.Language)
		if other, dup := seen[lang]; dup {
			t.Errorf("%s and %s are both the default for %s", other, e.Name, e.Language)
		}
		seen[lang] = e.Name
	}
}

func TestEveryTemplateHasAReadmeAndFiles(t *testing.T) {
	for _, e := range Catalog() {
		list, err := Files(e.Name)
		if err != nil || len(list) == 0 {
			t.Fatalf("%s: %v, %d files", e.Name, err, len(list))
		}
		hasReadme := false
		for _, f := range list {
			if strings.EqualFold(f, "README.md") {
				hasReadme = true
			}
		}
		if !hasReadme {
			t.Errorf("%s has no README.md", e.Name)
		}
	}
}

func TestGetIsCaseInsensitiveAndDefaultFor(t *testing.T) {
	if e, ok := Get("GO-BASIC"); !ok || e.Name != "go-basic" {
		t.Fatalf("Get: %+v %v", e, ok)
	}
	if e, ok := DefaultFor("go"); !ok || e.Name != "go-basic" {
		t.Fatalf("DefaultFor: %+v %v", e, ok)
	}
	if _, ok := DefaultFor("Cobol"); ok {
		t.Fatal("no default expected")
	}
	if _, ok := Get("nope"); ok {
		t.Fatal("unknown template found")
	}
}

func TestExtract(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out")
	if err := Extract("python-basic", dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "pyproject.toml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "src", "{{PROJECT_NAME_SNAKE}}", "cli.py")); err != nil {
		t.Fatal("file names with placeholders must be extracted as they are:", err)
	}
	goDest := filepath.Join(t.TempDir(), "go")
	if err := Extract("go-basic", goDest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(goDest, "go.mod")); err != nil {
		t.Fatal("go.mod.txt must be extracted as go.mod:", err)
	}
	if list, _ := Files("go-http"); !contains(list, "go.mod") || contains(list, "go.mod.txt") {
		t.Fatalf("Files must list go.mod, got %v", list)
	}
	if err := Extract("python-basic", dest); err == nil {
		t.Fatal("extracting into a non-empty directory must fail")
	}
	if err := Extract("nope", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("unknown template must fail")
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
