package gitutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func needGit(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("git is not installed")
	}
}

// isolate keeps the developer's own git configuration out of the tests.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(home, "none"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func TestGitignoreForKnownAndUnknownLanguages(t *testing.T) {
	for _, lang := range []string{"Go", "go", "JavaScript", "TypeScript", "React", "Python", "Rust", "Java", "C#", "C", "C++", "PHP", "Ruby", "Swift"} {
		c, specific := GitignoreFor(lang)
		if !specific || strings.TrimSpace(c) == "" {
			t.Errorf("%s: expected a specific .gitignore", lang)
		}
	}
	c, specific := GitignoreFor("Cobol")
	if specific || !strings.Contains(c, ".env") {
		t.Fatalf("unknown language must give the generic file, specific=%v", specific)
	}
}

func TestInitAndCommitWithoutConfiguredIdentity(t *testing.T) {
	needGit(t)
	isolate(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	res, err := InitAndCommit(context.Background(), dir, "Go", "Kaj", "Initial commit from Foundry")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Committed || !res.GitignoreWritten || !res.UsedFallbackIdentity {
		t.Fatalf("unexpected result: %+v", res)
	}
	out, err := exec.Command("git", "-C", dir, "log", "--format=%an|%s").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "Kaj|Initial commit from Foundry" {
		t.Fatalf("log = %q", out)
	}
}

func TestExistingGitignoreIsKept(t *testing.T) {
	needGit(t)
	isolate(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("mine\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	res, err := InitAndCommit(context.Background(), dir, "Go", "", "init")
	if err != nil {
		t.Fatal(err)
	}
	if res.GitignoreWritten {
		t.Fatal("must not overwrite an existing .gitignore")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, ".gitignore")); string(b) != "mine\n" {
		t.Fatalf(".gitignore = %q", b)
	}
}

func TestCloneRejectsOptionLikeInput(t *testing.T) {
	if err := Clone(context.Background(), "--upload-pack=evil", "", t.TempDir()); err == nil {
		t.Fatal("URL starting with '-' must be refused")
	}
	if err := Clone(context.Background(), "https://example.invalid/x", "--evil", t.TempDir()); err == nil {
		t.Fatal("branch starting with '-' must be refused")
	}
	if err := Clone(context.Background(), " ", "", t.TempDir()); err == nil {
		t.Fatal("empty URL must be refused")
	}
}

func TestCloneLocalRepository(t *testing.T) {
	needGit(t)
	isolate(t)
	src := t.TempDir()
	_ = os.WriteFile(filepath.Join(src, "f.txt"), []byte("hi"), 0o644)
	if _, err := InitAndCommit(context.Background(), src, "Go", "T", "c"); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "clone")
	if err := Clone(context.Background(), src, "", dest); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dest, "f.txt")); err != nil || string(b) != "hi" {
		t.Fatalf("clone content: %q %v", b, err)
	}
}
