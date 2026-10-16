package template

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanExcludesIgnoredAndDefaultDirectories(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.go", "package main")
	write(t, root, "go.mod", "module x")
	write(t, root, "generated/out.txt", "x")
	write(t, root, "node_modules/a/b.js", "x")
	write(t, root, ".git/HEAD", "x")
	write(t, root, IgnoreFile, "generated/\n")
	info, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"go.mod", "main.go"}; !reflect.DeepEqual(info.Files, want) {
		t.Fatalf("files = %v, want %v", info.Files, want)
	}
	if info.Language != "Go" {
		t.Fatalf("language = %q", info.Language)
	}
}

func TestUserRuleCanReincludeADefaultDirectory(t *testing.T) {
	root := t.TempDir()
	write(t, root, "build/keep.txt", "x")
	write(t, root, IgnoreFile, "!build/\n")
	files, err := ListFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files, []string{"build/keep.txt"}) {
		t.Fatalf("files = %v", files)
	}
}

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  string
	}{
		{"go", []string{"main.go", "go.mod"}, "Go"},
		{"python", []string{"app.py", "requirements.txt"}, "Python"},
		{"node", []string{"index.js", "package.json"}, "JavaScript"},
		{"typescript beats plain package.json", []string{"package.json", "tsconfig.json", "a.ts", "b.ts"}, "TypeScript"},
		{"unknown", []string{"README.md"}, "Unknown"},
		{"tie is alphabetical", []string{"a.go", "b.py"}, "Go"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			for _, f := range c.files {
				write(t, root, f, "x")
			}
			got, err := DetectLanguage(root)
			if err != nil || got != c.want {
				t.Fatalf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestScanErrors(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing directory must fail")
	}
	f := filepath.Join(t.TempDir(), "file")
	_ = os.WriteFile(f, []byte("x"), 0o644)
	if _, err := Scan(f); err == nil {
		t.Fatal("a file is not a template directory")
	}
}

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"go-cli", "my template", "a.b"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", " ", "a/b", `a\b`, "a:b", "..", " x", "x "} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestMetaIsLoadedAndNeverListedAsAFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.go", "package main")
	write(t, root, MetaFile, "post:\n  - go mod tidy\nnext:\n  - go run .\n")
	m, err := LoadMeta(root)
	if err != nil || len(m.Post) != 1 || m.Post[0] != "go mod tidy" || len(m.Next) != 1 {
		t.Fatalf("meta = %+v, err = %v", m, err)
	}
	files, err := ListFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f == MetaFile {
			t.Fatal("the meta file must not be part of the template files")
		}
	}
	if m, err := LoadMeta(t.TempDir()); err != nil || len(m.Post) != 0 {
		t.Fatalf("missing file must give an empty Meta: %+v %v", m, err)
	}
	write(t, root, MetaFile, "post: [unclosed")
	if _, err := LoadMeta(root); err == nil {
		t.Fatal("broken YAML must be reported")
	}
}
