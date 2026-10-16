// Package cli contains the Foundry command line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/SandSoftwareSolutions/Foundry/internal/detect"
	"github.com/SandSoftwareSolutions/Foundry/internal/gitutil"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Set at build time with -ldflags "-X github.com/SandSoftwareSolutions/Foundry/cmd.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// VersionString describes this build.
func VersionString() string {
	return fmt.Sprintf("%s (commit %s, built %s, %s/%s)", version, commit, date, runtime.GOOS, runtime.GOARCH)
}

// Execute runs the CLI and exits with a non-zero status on failure.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root := NewRootCmd()
	if err := root.ExecuteContext(ctx); err != nil {
		var usage *usageError
		if !errors.As(err, &usage) || usage.show {
			fmt.Fprintln(root.ErrOrStderr(), color.RedString("Error: ")+err.Error())
		}
		stop()
		os.Exit(1)
	}
}

// usageError marks a mistake in how a command was called.
type usageError struct {
	msg  string
	show bool
}

func (e *usageError) Error() string { return e.msg }

func usagef(format string, a ...any) error {
	return &usageError{msg: fmt.Sprintf(format, a...), show: true}
}

// NewRootCmd builds the whole command tree. It has no global state, so tests can build many.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "foundry",
		Short: "Create new projects from your own templates",
		Long: `Foundry scaffolds new projects from templates that you save once and reuse.

  - Detects the languages and tools installed on your machine
  - Saves directories as templates and remembers a default per language
  - Creates a project with placeholders such as {{PROJECT_NAME}} filled in
  - Initialises git with a sensible .gitignore and a first commit

Colors:
  --no-color disables colors, --color forces them; NO_COLOR is respected.`,
		Example: `  foundry template add go-cli ./examples/go-cli
  foundry config Go go-cli
  foundry new my-api --language Go
  foundry new my-site --template react-starter --var PORT=3000
  foundry new my-tool --git https://github.com/you/template-repo
  foundry new my-app --template go-cli --dry-run`,
		Version:       VersionString(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("foundry {{.Version}}\n")
	root.PersistentFlags().Bool("no-color", false, "Disable colored output")
	root.PersistentFlags().Bool("color", false, "Force colored output (overrides NO_COLOR)")
	root.PersistentFlags().String("config", "", "Path of the config file (default ~/.foundry/config.yaml)")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		applyColor(cmd)
		if p, _ := cmd.Flags().GetString("config"); p != "" {
			config.SetPathOverride(p)
		}
		if skipsFirstRun(cmd) {
			return nil
		}
		return firstRun(cmd.ErrOrStderr())
	}

	root.AddCommand(newNewCmd(), newTemplateCmd(), newConfigCmd(), newDetectCmd(), newVersionCmd())
	root.CompletionOptions.HiddenDefaultCmd = false
	return root
}

// applyColor resolves --color, --no-color and NO_COLOR, in that order of precedence.
func applyColor(cmd *cobra.Command) {
	if v, ok := os.LookupEnv("NO_COLOR"); ok && strings.TrimSpace(v) != "" {
		color.NoColor = true
	}
	if f := cmd.Flags().Lookup("no-color"); f != nil && f.Changed {
		on, _ := cmd.Flags().GetBool("no-color")
		color.NoColor = on
	}
	if f := cmd.Flags().Lookup("color"); f != nil && f.Changed {
		on, _ := cmd.Flags().GetBool("color")
		if on {
			color.NoColor = false
		}
	}
}

func skipsFirstRun(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "version", "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return true
		}
	}
	return false
}

// firstRun creates the config file on the first use: it detects the installed tools and picks the
// author name from git when available. Nothing is changed when the file already exists.
func firstRun(errOut io.Writer) error {
	if config.Exists() {
		return nil
	}
	cfg := config.Default()
	detect.Apply(cfg, detect.System().Scan())
	cfg.Author = gitutil.GlobalUserName()
	if err := config.Save(cfg); err != nil {
		// A read-only home directory must not make every command fail.
		fmt.Fprintln(errOut, color.YellowString("Warning: could not create the config file: %v", err))
		return nil
	}
	p, _ := config.Path()
	fmt.Fprintf(errOut, "First run: created %s (detected %d languages and %d tools).\n",
		p, len(cfg.InstalledLanguages), len(cfg.InstalledDevTools))
	if cfg.Author == "" {
		fmt.Fprintln(errOut, "Set your name for {{AUTHOR}} with: foundry config --user \"Your Name\"")
	}
	return nil
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "foundry %s\n", VersionString())
		},
	}
}
