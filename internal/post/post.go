// Package post runs the small setup commands that follow project creation.
//
// Commands are started directly (no shell), run inside the new project, have a time limit, and never
// start long-running processes such as dev servers.
package post

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Step is one command to run in the project directory.
type Step struct {
	Name string
	Cmd  string
	Args []string
}

// String renders the step like a shell command, for display.
func (s Step) String() string { return strings.TrimSpace(s.Cmd + " " + strings.Join(s.Args, " ")) }

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// StepsFor returns the setup steps that make sense for a language and the files present in dir.
func StepsFor(language, dir string) []Step {
	switch strings.ToLower(language) {
	case "go":
		if exists(dir, "go.mod") {
			return []Step{{Name: "Tidy Go modules", Cmd: "go", Args: []string{"mod", "tidy"}}}
		}
	case "javascript", "typescript", "react", "vue", "node.js":
		if exists(dir, "package.json") {
			return []Step{{Name: "Install npm packages", Cmd: "npm", Args: []string{"install"}}}
		}
	}
	return nil
}

// FromCommands turns command lines from a template into steps. A line is a program and its arguments separated
// by spaces; there is no shell, so quotes, pipes, redirects and && are not available.
func FromCommands(lines []string) ([]Step, error) {
	var steps []Step
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		for _, f := range fields {
			if strings.ContainsAny(f, ";|&<>`$()") {
				return nil, fmt.Errorf("%q: only one plain command per line is allowed; shell syntax such as %q is not supported", line, f)
			}
		}
		steps = append(steps, Step{Name: "Run", Cmd: fields[0], Args: fields[1:]})
	}
	return steps, nil
}

// NextSteps are commands the user can run themselves afterwards (nothing here is executed).
func NextSteps(language string) []string {
	switch strings.ToLower(language) {
	case "go":
		return []string{"go build ./...", "go test ./..."}
	case "javascript", "typescript", "react", "vue", "node.js":
		return []string{"npm run dev"}
	case "python":
		return []string{"python -m venv .venv", "pip install -r requirements.txt"}
	case "rust":
		return []string{"cargo build", "cargo run"}
	case "java":
		return []string{"mvn package"}
	}
	return nil
}

// Timeout is the limit for a single step.
const Timeout = 5 * time.Minute

// Run executes the steps in order and stops at the first failure. A tool that is not installed is
// reported as an error without running anything.
func Run(ctx context.Context, dir string, steps []Step, out, errOut io.Writer) error {
	for _, s := range steps {
		if _, err := exec.LookPath(s.Cmd); err != nil {
			return fmt.Errorf("%s: %q is not installed or not on PATH", s.Name, s.Cmd)
		}
		stepCtx, cancel := context.WithTimeout(ctx, Timeout)
		cmd := exec.CommandContext(stepCtx, s.Cmd, s.Args...)
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = out, errOut
		err := cmd.Run()
		cancel()
		if err != nil {
			return fmt.Errorf("%s (%s): %w", s.Name, s.String(), err)
		}
	}
	return nil
}
