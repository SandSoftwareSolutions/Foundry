package post

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStepsDependOnFilesPresent(t *testing.T) {
	dir := t.TempDir()
	if got := StepsFor("Go", dir); got != nil {
		t.Fatalf("no go.mod: expected no steps, got %v", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	got := StepsFor("Go", dir)
	if len(got) != 1 || got[0].String() != "go mod tidy" {
		t.Fatalf("got %v", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644)
	got = StepsFor("TypeScript", dir)
	if len(got) != 1 || got[0].String() != "npm install" {
		t.Fatalf("got %v", got)
	}
}

func TestNeverStartsDevServers(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.py"), []byte(""), 0o644)
	for _, lang := range []string{"JavaScript", "React", "Python"} {
		for _, s := range StepsFor(lang, dir) {
			if strings.Contains(s.String(), "run dev") || strings.Contains(s.String(), "main.py") {
				t.Fatalf("%s: unexpected long-running step %q", lang, s)
			}
		}
	}
}

func TestProjectNameWithShellCharactersIsHarmless(t *testing.T) {
	// The directory is passed as cmd.Dir, never interpolated into a command line.
	base := t.TempDir()
	dir := filepath.Join(base, `a"; touch pwned; "b`)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skip("file system does not allow this name:", err)
	}
	step := Step{Name: "echo", Cmd: "go", Args: []string{"version"}}
	var out, errOut bytes.Buffer
	if err := Run(context.Background(), dir, []Step{step}, &out, &errOut); err != nil {
		t.Skip("go not available:", err)
	}
	if _, err := os.Stat(filepath.Join(base, "pwned")); err == nil {
		t.Fatal("command injection through the directory name")
	}
}

func TestMissingToolIsReported(t *testing.T) {
	err := Run(context.Background(), t.TempDir(), []Step{{Name: "x", Cmd: "definitely-not-installed-xyz"}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v", err)
	}
}

func TestNextSteps(t *testing.T) {
	if got := NextSteps("Go"); !reflect.DeepEqual(got, []string{"go build ./...", "go test ./..."}) {
		t.Fatalf("got %v", got)
	}
	if NextSteps("Cobol") != nil {
		t.Fatal("unknown language has no next steps")
	}
}
