package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

func okf(w io.Writer, format string, a ...any) {
	fmt.Fprintln(w, color.GreenString("✓ ")+fmt.Sprintf(format, a...))
}

func warnf(w io.Writer, format string, a ...any) {
	fmt.Fprintln(w, color.YellowString("⚠ ")+fmt.Sprintf(format, a...))
}

func infof(w io.Writer, format string, a ...any) {
	fmt.Fprintln(w, color.CyanString(format, a...))
}

func boldf(w io.Writer, format string, a ...any) {
	fmt.Fprintln(w, color.New(color.Bold).Sprintf(format, a...))
}

// interactive reports whether stdin and stdout are terminals, so prompts make sense.
func interactive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

// parseVars turns --var KEY=VALUE entries into variables and validates the names.
func parseVars(entries []string) (render.Vars, error) {
	vars := render.Vars{}
	for _, kv := range entries {
		key, value, ok := strings.Cut(kv, "=")
		key = strings.TrimSpace(key)
		if !ok {
			return nil, usagef("invalid --var %q: expected KEY=VALUE", kv)
		}
		if !render.ValidKey(key) {
			return nil, usagef("invalid --var name %q: use letters, digits and underscores, not starting with a digit", key)
		}
		vars[key] = value
	}
	return vars, nil
}
