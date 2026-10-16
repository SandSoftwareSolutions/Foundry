package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/SandSoftwareSolutions/Foundry/internal/detect"
	"github.com/spf13/cobra"
)

func newDetectCmd() *cobra.Command {
	var asJSON, yes, nonInteractive bool
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect installed languages, package managers and tools",
		Long: `Look for common languages, package managers and developer tools on this machine and store the
result in the configuration (after you confirm). Foundry uses it, for example, to know whether git is available.`,
		Example: `  foundry detect
  foundry detect --json
  foundry detect --non-interactive --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			result := detect.System().Scan()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return err
				}
			} else {
				printDetected(out, result)
			}
			save := yes
			if !yes && !nonInteractive && !asJSON {
				fmt.Fprint(out, "\nSave this to the configuration? [y/N] ")
				save = askYes(cmd.InOrStdin())
			}
			if !save {
				if !asJSON {
					infof(out, "Nothing saved.")
				}
				return nil
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			detect.Apply(cfg, result)
			if err := config.Save(cfg); err != nil {
				return err
			}
			if !asJSON {
				okf(out, "Saved to the configuration.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print the result as JSON")
	cmd.Flags().BoolVar(&yes, "yes", false, "Save without asking")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Never ask; only save when --yes is given")
	return cmd
}

func askYes(in io.Reader) bool {
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "j", "ja":
		return true
	}
	return false
}

func printDetected(out io.Writer, r detect.Result) {
	section := func(title string, tools []detect.Tool, found []string) {
		boldf(out, "%s", title)
		for _, e := range detect.Names(tools, found) {
			mark := "✗"
			if e.Found {
				mark = "✓"
			}
			fmt.Fprintf(out, "  %s %s\n", mark, e.Name)
		}
		fmt.Fprintln(out)
	}
	section("Languages", detect.Catalog.Languages, r.Languages)
	section("Package managers", detect.Catalog.PackageManagers, r.PackageManagers)
	section("Development tools", detect.Catalog.DevTools, r.DevTools)
	if r.VSCodePath != "" {
		fmt.Fprintf(out, "VS Code: %s\n", r.VSCodePath)
	}
}
