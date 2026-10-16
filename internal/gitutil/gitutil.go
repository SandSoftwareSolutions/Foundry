// Package gitutil wraps the few git operations Foundry needs, without a shell.
package gitutil

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed gitignores/*.gitignore
var gitignores embed.FS

// languageFile maps a template language tag to the embedded .gitignore.
var languageFile = map[string]string{
	"go": "Go", "javascript": "Node", "typescript": "Node", "react": "Node", "vue": "Node", "node.js": "Node",
	"python": "Python", "rust": "Rust", "java": "Java", "kotlin": "Java", "c#": "CSharp", "c": "C",
	"c++": "Cpp", "php": "PHP", "ruby": "Ruby", "swift": "Swift",
}

// GitignoreFor returns a starter .gitignore for a language. Unknown languages get a small generic one;
// the second result tells whether a language-specific file was used.
func GitignoreFor(language string) (content string, specific bool) {
	if name, ok := languageFile[strings.ToLower(strings.TrimSpace(language))]; ok {
		if b, err := gitignores.ReadFile("gitignores/" + name + ".gitignore"); err == nil {
			return string(b), true
		}
	}
	b, _ := gitignores.ReadFile("gitignores/Generic.gitignore")
	return string(b), false
}

// Available reports whether git is on the PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Never prompt for credentials; fail instead.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return text, fmt.Errorf("git %s: %w: %s", args[0], err, lastLine(text))
		}
		return text, fmt.Errorf("git %s: %w", args[0], err)
	}
	return text, nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// InitResult describes what InitAndCommit did.
type InitResult struct {
	GitignoreWritten  bool
	GitignoreSpecific bool
	Committed         bool
	// UsedFallbackIdentity is true when git had no user.name/user.email and Foundry supplied one.
	UsedFallbackIdentity bool
}

// InitAndCommit runs `git init` in dir, adds a .gitignore when there is none, stages everything and
// makes the first commit. If git has no identity configured, the commit uses author (or "Foundry")
// with the address foundry@localhost, so the first commit does not fail on a fresh machine.
func InitAndCommit(ctx context.Context, dir, language, author, message string) (InitResult, error) {
	var res InitResult
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if _, err := run(ctx, "", "init", "--quiet", dir); err != nil {
		return res, err
	}
	gi := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gi); errors.Is(err, os.ErrNotExist) {
		content, specific := GitignoreFor(language)
		if err := os.WriteFile(gi, []byte(content), 0o644); err != nil {
			return res, fmt.Errorf("cannot write .gitignore: %w", err)
		}
		res.GitignoreWritten, res.GitignoreSpecific = true, specific
	}
	if _, err := run(ctx, dir, "add", "--all"); err != nil {
		return res, err
	}
	args := []string{}
	if !hasIdentity(ctx, dir) {
		name := strings.TrimSpace(author)
		if name == "" {
			name = "Foundry"
		}
		args = append(args, "-c", "user.name="+name, "-c", "user.email=foundry@localhost")
		res.UsedFallbackIdentity = true
	}
	args = append(args, "commit", "--quiet", "--no-gpg-sign", "-m", message)
	if _, err := run(ctx, dir, args...); err != nil {
		return res, err
	}
	res.Committed = true
	return res, nil
}

func hasIdentity(ctx context.Context, dir string) bool {
	name, err1 := run(ctx, dir, "config", "user.name")
	email, err2 := run(ctx, dir, "config", "user.email")
	return err1 == nil && err2 == nil && name != "" && email != ""
}

// Clone fetches a repository into dest (a shallow clone). url must not start with "-".
func Clone(ctx context.Context, url, branch, dest string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("empty repository URL")
	}
	if strings.HasPrefix(url, "-") {
		return errors.New("repository URL must not start with '-'")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	args := []string{"clone", "--quiet", "--depth", "1"}
	if branch != "" {
		if strings.HasPrefix(branch, "-") {
			return errors.New("branch must not start with '-'")
		}
		args = append(args, "--branch", branch)
	}
	args = append(args, "--", url, dest)
	_, err := run(ctx, "", args...)
	return err
}

// GlobalUserName returns `git config --global user.name`, or "" when it is not set or git is missing.
func GlobalUserName() string {
	if !Available() {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := run(ctx, "", "config", "--global", "user.name")
	if err != nil {
		return ""
	}
	return out
}
