package project

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/SandSoftwareSolutions/Foundry/internal/template"
)

func write(t *testing.T, root, rel, content string, mode os.FileMode) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var fixedNow = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

func opts(src, dst string) Options {
	return Options{TemplateDir: src, Name: "my-app", TargetDir: dst, Author: "Kaj", License: "MIT", Now: fixedNow}
}

func TestCreatesProjectWithPlaceholdersInContentAndNames(t *testing.T) {
	src := t.TempDir()
	write(t, src, "README.md", "# {{PROJECT_NAME}} by {{AUTHOR}} ({{YEAR}}, {{LICENSE}})", 0o644)
	write(t, src, "cmd/{{PROJECT_NAME}}/main.go", "package main // {{PROJECT_NAME_PASCAL}}", 0o644)
	write(t, src, "blob.bin", "a\x00b{{PROJECT_NAME}}", 0o644)
	dst := filepath.Join(t.TempDir(), "out")

	plan, err := BuildPlan(opts(src, dst))
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dst, "README.md"); got != "# my-app by Kaj (2026, MIT)" {
		t.Fatalf("README = %q", got)
	}
	if got := read(t, dst, "cmd/my-app/main.go"); got != "package main // MyApp" {
		t.Fatalf("main.go = %q", got)
	}
	if got := read(t, dst, "blob.bin"); got != "a\x00b{{PROJECT_NAME}}" {
		t.Fatalf("binary file must be copied unchanged, got %q", got)
	}
}

func TestDryRunPlanMatchesWhatExecuteWrites(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.txt", "1", 0o644)
	write(t, src, "sub/b.txt", "2", 0o644)
	write(t, src, template.IgnoreFile, "secret.txt\n", 0o644)
	write(t, src, "secret.txt", "x", 0o644)
	write(t, src, "node_modules/x.js", "x", 0o644)
	dst := filepath.Join(t.TempDir(), "out")

	plan, err := BuildPlan(opts(src, dst))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err == nil {
		t.Fatal("BuildPlan must not create the target")
	}
	want := plan.Files()
	if err := plan.Execute(); err != nil {
		t.Fatal(err)
	}
	var got []string
	_ = filepath.WalkDir(dst, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dst, p)
			got = append(got, filepath.ToSlash(rel))
		}
		return nil
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("written %v, planned %v", got, want)
	}
	if !reflect.DeepEqual(want, []string{"a.txt", "sub/b.txt"}) {
		t.Fatalf("unexpected plan: %v", want)
	}
}

func TestRefusesNonEmptyTargetButAcceptsEmptyDirectory(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.txt", "1", 0o644)
	full := t.TempDir()
	write(t, full, "x", "x", 0o644)
	if _, err := BuildPlan(opts(src, full)); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected a not-empty error, got %v", err)
	}
	empty := t.TempDir()
	plan, err := BuildPlan(opts(src, empty))
	if err != nil {
		t.Fatal(err)
	}
	if !plan.TargetExisted {
		t.Fatal("TargetExisted should be true")
	}
	if err := plan.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTargetInsideTemplateIsNotCopiedIntoItself(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.txt", "1", 0o644)
	dst := filepath.Join(src, "generated", "proj")
	plan, err := BuildPlan(opts(src, dst))
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "generated")); err == nil {
		t.Fatal("the target was copied into itself")
	}
	if got := read(t, dst, "a.txt"); got != "1" {
		t.Fatalf("a.txt = %q", got)
	}
}

func TestUnsafeValueInFileNameIsRefused(t *testing.T) {
	src := t.TempDir()
	write(t, src, "{{X}}.txt", "1", 0o644)
	o := opts(src, filepath.Join(t.TempDir(), "out"))
	o.Vars = render.Vars{"X": "../../evil"}
	if _, err := BuildPlan(o); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected an unsafe file name error, got %v", err)
	}
}

func TestUnresolvedPlaceholdersAreReported(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.txt", "{{PROJECT_NAME}} {{PORT}} {{PORT}}", 0o644)
	write(t, src, "{{DIR}}/b.txt", "x", 0o644)
	plan, err := BuildPlan(opts(src, filepath.Join(t.TempDir(), "out")))
	if err != nil {
		t.Fatal(err)
	}
	got := plan.UnresolvedSummary()
	want := []string{"a.txt: PORT", "{{DIR}}: DIR", "{{DIR}}/b.txt: DIR"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unresolved = %v, want %v", got, want)
	}
}

func TestFailureCleansUpCreatedTarget(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.txt", "1", 0o644)
	write(t, src, "b.txt", "2", 0o644)
	dst := filepath.Join(t.TempDir(), "out")
	plan, err := BuildPlan(opts(src, dst))
	if err != nil {
		t.Fatal(err)
	}
	// Break the plan: the second source file disappears before it is written.
	_ = os.Remove(filepath.Join(src, "b.txt"))
	if err := plan.Execute(); err == nil {
		t.Fatal("expected a failure")
	}
	if _, err := os.Stat(dst); err == nil {
		t.Fatal("partial project must be removed after a failure")
	}
}

func TestSymlinksAreSkippedWithAWarning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	src := t.TempDir()
	write(t, src, "a.txt", "1", 0o644)
	secret := filepath.Join(t.TempDir(), "secret.txt")
	_ = os.WriteFile(secret, []byte("s3cret"), 0o644)
	if err := os.Symlink(secret, filepath.Join(src, "link.txt")); err != nil {
		t.Skip("cannot create symlink:", err)
	}
	plan, err := BuildPlan(opts(src, filepath.Join(t.TempDir(), "out")))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Files(), []string{"a.txt"}) || len(plan.Warnings) != 1 {
		t.Fatalf("files=%v warnings=%v", plan.Files(), plan.Warnings)
	}
}

func TestExecutableBitIsPreserved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no executable bit on Windows")
	}
	src := t.TempDir()
	write(t, src, "run.sh", "#!/bin/sh", 0o755)
	dst := filepath.Join(t.TempDir(), "out")
	plan, _ := BuildPlan(opts(src, dst))
	if err := plan.Execute(); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(filepath.Join(dst, "run.sh"))
	if st.Mode().Perm()&0o100 == 0 {
		t.Fatalf("mode = %v", st.Mode())
	}
}

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"app", "my-app", "My App", "a.b_c"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", " ", ".", "..", "a/b", `a\b`, "a:b", "a*b", " x", strings.Repeat("x", 101)} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
