package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	var (
		user, license, defaultLang, clearDefault string
		docker, interactiveMode                  bool
		view, asJSON, showPath                   bool
	)
	cmd := &cobra.Command{
		Use:   "config [language [template]]",
		Short: "View or change settings and per-language default templates",
		Long: `View or change Foundry's settings.

Without arguments or flags the current configuration is shown. Set values with flags, or set the
default template of a language with positional arguments:

  foundry config <language> <template>     set the default template for a language
  foundry config <language> ""             clear it again`,
		Example: `  foundry config --user "Jane Doe" --license Apache-2.0
  foundry config Go go-cli
  foundry config Go ""
  foundry config --clear-default Python
  foundry config --json
  foundry config --path`,
		Args: cobra.MaximumNArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			cfg, err := config.Load()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var cands []string
			switch len(args) {
			case 0:
				set := map[string]bool{"Go": true, "Python": true, "JavaScript": true, "TypeScript": true, "React": true, "Rust": true, "Java": true}
				for _, l := range cfg.Languages() {
					set[l] = true
				}
				for l := range set {
					cands = append(cands, l)
				}
			case 1:
				cands = templateNames(cfg)
			}
			var out []string
			for _, c := range cands {
				if strings.HasPrefix(strings.ToLower(c), strings.ToLower(toComplete)) {
					out = append(out, c)
				}
			}
			sort.Strings(out)
			return out, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if showPath {
				p, err := config.Path()
				if err != nil {
					return err
				}
				fmt.Fprintln(out, p)
				return nil
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			f := cmd.Flags()
			changed := false
			if len(args) >= 1 && !view {
				template := ""
				if len(args) == 2 {
					template = args[1]
				}
				if len(args) == 1 {
					return usagef("give a template name after the language (use \"\" to clear): foundry config %s <template>", args[0])
				}
				if err := cfg.SetLanguageDefault(args[0], template); err != nil {
					return err
				}
				if template == "" {
					okf(out, "Cleared the default template for %s", args[0])
				} else {
					okf(out, "Default template for %s: %s", args[0], template)
				}
				changed = true
			}
			if clearDefault != "" {
				if err := cfg.SetLanguageDefault(clearDefault, ""); err != nil {
					return err
				}
				okf(out, "Cleared the default template for %s", clearDefault)
				changed = true
			}
			for _, s := range []struct{ flag, key, value string }{
				{"user", "author", user}, {"license", "license", license}, {"default-language", "default_language", defaultLang},
			} {
				if f.Changed(s.flag) {
					if err := cfg.Set(s.key, s.value); err != nil {
						return err
					}
					changed = true
				}
			}
			if f.Changed("docker") {
				cfg.Docker = docker
				changed = true
			}
			if f.Changed("interactive") {
				cfg.Interactive = interactiveMode
				changed = true
			}
			if changed {
				if err := config.Save(cfg); err != nil {
					return err
				}
			}
			if changed && !asJSON && !view {
				return nil // the confirmations above are enough; "foundry config" shows everything
			}
			return printConfig(cmd, cfg, asJSON)
		},
	}
	f := cmd.Flags()
	f.StringVar(&user, "user", "", "Set the author name ({{AUTHOR}})")
	f.StringVar(&license, "license", "", "Set the license ({{LICENSE}})")
	f.StringVar(&defaultLang, "default-language", "", "Set the default language")
	f.BoolVar(&docker, "docker", false, "Enable or disable Docker support (--docker=false)")
	f.BoolVar(&interactiveMode, "interactive", true, "Enable or disable interactive menus (--interactive=false)")
	f.StringVar(&clearDefault, "clear-default", "", "Clear the default template of a language")
	f.BoolVar(&view, "view", false, "Show the configuration (the default)")
	f.BoolVar(&asJSON, "json", false, "Print as JSON")
	f.BoolVar(&showPath, "path", false, "Print the location of the config file")
	return cmd
}

func printConfig(cmd *cobra.Command, cfg *config.Config, asJSON bool) error {
	out := cmd.OutOrStdout()
	if asJSON {
		type view struct {
			Author           string            `json:"author"`
			License          string            `json:"license"`
			DefaultLanguage  string            `json:"default_language"`
			Docker           bool              `json:"docker"`
			Interactive      bool              `json:"interactive"`
			Languages        []string          `json:"installed_languages"`
			PackageManagers  []string          `json:"installed_package_managers"`
			DevTools         []string          `json:"installed_dev_tools"`
			VSCodePath       string            `json:"vscode_path"`
			Templates        int               `json:"templates"`
			LanguageDefaults map[string]string `json:"language_defaults"`
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(view{cfg.Author, cfg.License, cfg.DefaultLanguage, cfg.Docker, cfg.Interactive,
			nz(cfg.InstalledLanguages), nz(cfg.InstalledPackageManagers), nz(cfg.InstalledDevTools), cfg.VSCodePath,
			len(cfg.Templates), cfg.LanguageDefaults})
	}
	p, _ := config.Path()
	fmt.Fprintf(out, "Config file:       %s\n", p)
	fmt.Fprintf(out, "Author:            %s\n", orDash(cfg.Author))
	fmt.Fprintf(out, "License:           %s\n", orDash(cfg.License))
	fmt.Fprintf(out, "Default language:  %s\n", orDash(cfg.DefaultLanguage))
	fmt.Fprintf(out, "Docker:            %t\n", cfg.Docker)
	fmt.Fprintf(out, "Interactive:       %t\n", cfg.Interactive)
	fmt.Fprintf(out, "Languages:         %s\n", orDash(strings.Join(cfg.InstalledLanguages, ", ")))
	fmt.Fprintf(out, "Package managers:  %s\n", orDash(strings.Join(cfg.InstalledPackageManagers, ", ")))
	fmt.Fprintf(out, "Dev tools:         %s\n", orDash(strings.Join(cfg.InstalledDevTools, ", ")))
	fmt.Fprintf(out, "VS Code:           %s\n", orDash(cfg.VSCodePath))
	fmt.Fprintf(out, "Templates:         %d saved\n", len(cfg.Templates))
	if len(cfg.LanguageDefaults) > 0 {
		fmt.Fprintln(out, "\nLanguage defaults:")
		langs := make([]string, 0, len(cfg.LanguageDefaults))
		for l := range cfg.LanguageDefaults {
			langs = append(langs, l)
		}
		sort.Strings(langs)
		for _, l := range langs {
			fmt.Fprintf(out, "  %-14s %s\n", l, cfg.LanguageDefaults[l])
		}
	}
	return nil
}

func nz(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
