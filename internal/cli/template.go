package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/SandSoftwareSolutions/Foundry/internal/template"
	"github.com/spf13/cobra"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates", "t"},
		Short:   "Manage saved templates",
		Long: `Save directories as templates, list them, inspect them and keep them up to date.

A template is a normal directory. Put a .foundryignore file in it to leave things out
(same rules as .gitignore). node_modules, .git, vendor, dist, build, .venv and __pycache__ are
always left out unless you re-include them with a rule such as "!build/".`,
	}
	cmd.AddCommand(newTemplateAddCmd(), newTemplateListCmd(), newTemplateShowCmd(), newTemplateRemoveCmd(),
		newTemplateUpdateCmd(), newTemplateEditCmd(), newTemplateVarsCmd())
	return cmd
}

func templateCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, n := range templateNames(cfg) {
		if strings.HasPrefix(strings.ToLower(n), strings.ToLower(toComplete)) {
			out = append(out, n)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func newTemplateAddCmd() *cobra.Command {
	var description, language string
	cmd := &cobra.Command{
		Use:   "add <name> <path>",
		Short: "Save a directory as a template",
		Long: `Scan a directory and save it as a template. The language is detected from the files;
override it with --language, for example to tag a framework such as React or Vue.
Adding a name that exists replaces that template.`,
		Example: `  foundry template add go-cli ./examples/go-cli
  foundry template add react-starter ~/templates/react --description "React with TypeScript" --language React`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, path := args[0], args[1]
			if err := template.ValidateName(name); err != nil {
				return usagef("%v", err)
			}
			info, err := template.Scan(path)
			if err != nil {
				return err
			}
			if language = strings.TrimSpace(language); language != "" {
				info.Language = language
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			_, existed := cfg.FindTemplate(name)
			cfg.UpsertTemplate(config.Template{Name: name, Path: info.Path, Language: info.Language, Description: description, Files: info.Files})
			if err := config.Save(cfg); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			verb := "saved"
			if existed {
				verb = "updated"
			}
			okf(out, "Template %q %s", name, verb)
			fmt.Fprintf(out, "  Path:     %s\n  Language: %s\n  Files:    %d\n", info.Path, info.Language, len(info.Files))
			if description != "" {
				fmt.Fprintf(out, "  About:    %s\n", description)
			}
			if len(info.Files) == 0 {
				warnf(cmd.ErrOrStderr(), "The template has no files (check .foundryignore).")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the template")
	cmd.Flags().StringVarP(&language, "language", "l", "", "Override the detected language or framework tag")
	return cmd
}

func newTemplateListCmd() *cobra.Command {
	var sortBy string
	var quiet, asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List saved templates",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sortBy != "name" && sortBy != "language" {
				return usagef("--sort must be name or language")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			ts := sortedTemplates(cfg)
			if sortBy == "language" {
				sort.SliceStable(ts, func(i, j int) bool { return ts[i].Language < ts[j].Language })
			}
			out := cmd.OutOrStdout()
			if asJSON {
				type item struct {
					config.Template
					DefaultFor []string `json:"default_for"`
					Exists     bool     `json:"exists"`
				}
				items := make([]item, 0, len(ts))
				for _, t := range ts {
					_, err := os.Stat(t.Path)
					d := cfg.DefaultFor(t.Name)
					if d == nil {
						d = []string{}
					}
					t.Files = nil
					items = append(items, item{Template: t, DefaultFor: d, Exists: err == nil})
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}
			if len(ts) == 0 {
				if !quiet {
					fmt.Fprintln(out, "No templates saved yet.\nAdd one with: foundry template add <name> <path>")
				}
				return nil
			}
			if quiet {
				for _, t := range ts {
					fmt.Fprintln(out, t.Name)
				}
				return nil
			}
			boldf(out, "Saved templates (%d)\n", len(ts))
			for _, t := range ts {
				line := fmt.Sprintf("%-24s %-12s %s", t.Name, t.Language, t.Path)
				fmt.Fprintln(out, line)
				if t.Description != "" {
					fmt.Fprintf(out, "    %s\n", t.Description)
				}
				if d := cfg.DefaultFor(t.Name); len(d) > 0 {
					infof(out, "    default for: %s", strings.Join(d, ", "))
				}
				if _, err := os.Stat(t.Path); err != nil {
					warnf(out, "    path no longer exists")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort", "name", "Sort by name or language")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Only print the names")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print as JSON")
	return cmd
}

func newTemplateShowCmd() *cobra.Command {
	var filesOnly, summary, asJSON bool
	cmd := &cobra.Command{
		Use:               "show <name>",
		Short:             "Show the details of a template",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: templateCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			t, ok := cfg.FindTemplate(args[0])
			if !ok {
				return notFound("template", args[0], templateNames(cfg))
			}
			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(t)
			}
			if !filesOnly {
				fmt.Fprintf(out, "Template: %s\nLanguage: %s\nPath:     %s\n", t.Name, t.Language, t.Path)
				if t.Description != "" {
					fmt.Fprintf(out, "About:    %s\n", t.Description)
				}
				if d := cfg.DefaultFor(t.Name); len(d) > 0 {
					infof(out, "Default for: %s", strings.Join(d, ", "))
				}
				if _, err := os.Stat(t.Path); err != nil {
					warnf(out, "The template path no longer exists.")
				}
			}
			if summary {
				return nil
			}
			files := append([]string(nil), t.Files...)
			sort.Strings(files)
			fmt.Fprintf(out, "\nFiles (%d):\n", len(files))
			byDir := map[string][]string{}
			var dirs []string
			for _, f := range files {
				d := filepath.ToSlash(filepath.Dir(filepath.FromSlash(f)))
				if d == "." {
					d = "(root)"
				}
				if _, seen := byDir[d]; !seen {
					dirs = append(dirs, d)
				}
				byDir[d] = append(byDir[d], filepath.Base(filepath.FromSlash(f)))
			}
			sort.Strings(dirs)
			for _, d := range dirs {
				fmt.Fprintf(out, "  %s\n", d)
				for _, f := range byDir[d] {
					fmt.Fprintf(out, "    %s\n", f)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&filesOnly, "files-only", false, "Only print the files")
	cmd.Flags().BoolVar(&summary, "summary", false, "Only print the details, not the files")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print as JSON")
	return cmd
}

func newTemplateRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:               "remove <name>",
		Aliases:           []string{"rm"},
		Short:             "Remove a saved template (the files stay)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: templateCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			t, ok := cfg.FindTemplate(args[0])
			if !ok {
				return notFound("template", args[0], templateNames(cfg))
			}
			name := t.Name
			if d := cfg.DefaultFor(name); len(d) > 0 && !force {
				return usagef("template %q is the default for: %s\nUse --force to remove it anyway (the defaults are cleared).", name, strings.Join(d, ", "))
			}
			cleared, err := cfg.RemoveTemplate(name)
			if err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			okf(cmd.OutOrStdout(), "Template %q removed", name)
			if len(cleared) > 0 {
				infof(cmd.OutOrStdout(), "Cleared the default for: %s", strings.Join(cleared, ", "))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Remove even if it is the default for a language")
	return cmd
}

func newTemplateUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [name...]",
		Short: "Rescan templates so their file lists and languages are current",
		Long: `Rescan the directory of each named template (or of all templates when no name is given).
The description, and a language you set by hand, are kept.`,
		ValidArgsFunction: templateCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			names := args
			if len(names) == 0 {
				names = templateNames(cfg)
			}
			out := cmd.OutOrStdout()
			failed := 0
			for _, n := range names {
				t, ok := cfg.FindTemplate(n)
				if !ok {
					return notFound("template", n, templateNames(cfg))
				}
				info, err := template.Scan(t.Path)
				if err != nil {
					warnf(cmd.ErrOrStderr(), "%s: %v", t.Name, err)
					failed++
					continue
				}
				before := len(t.Files)
				t.Files = info.Files
				okf(out, "%s: %d files (was %d)", t.Name, len(info.Files), before)
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			if failed > 0 {
				return fmt.Errorf("%d template(s) could not be rescanned", failed)
			}
			return nil
		},
	}
}

func newTemplateEditCmd() *cobra.Command {
	var description, language, path, rename string
	cmd := &cobra.Command{
		Use:               "edit <name>",
		Short:             "Change the description, language, path or name of a template",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: templateCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()
			if !f.Changed("description") && !f.Changed("language") && !f.Changed("path") && !f.Changed("rename") {
				return usagef("nothing to change: use --description, --language, --path or --rename")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			t, ok := cfg.FindTemplate(args[0])
			if !ok {
				return notFound("template", args[0], templateNames(cfg))
			}
			if f.Changed("description") {
				t.Description = description
			}
			if f.Changed("language") {
				if strings.TrimSpace(language) == "" {
					return usagef("--language cannot be empty")
				}
				t.Language = strings.TrimSpace(language)
			}
			if f.Changed("path") {
				info, err := template.Scan(path)
				if err != nil {
					return err
				}
				t.Path, t.Files = info.Path, info.Files
			}
			if f.Changed("rename") {
				if err := template.ValidateName(rename); err != nil {
					return usagef("%v", err)
				}
				if other, exists := cfg.FindTemplate(rename); exists && other != t {
					return fmt.Errorf("a template named %q already exists", rename)
				}
				old := t.Name
				for lang, d := range cfg.LanguageDefaults {
					if d == old {
						cfg.LanguageDefaults[lang] = rename
					}
				}
				t.Name = rename
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			okf(cmd.OutOrStdout(), "Template %q updated", t.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&language, "language", "", "New language or framework tag")
	cmd.Flags().StringVar(&path, "path", "", "New directory (rescanned)")
	cmd.Flags().StringVar(&rename, "rename", "", "New name")
	return cmd
}

func newTemplateVarsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vars <name>",
		Short: "List the placeholders a template uses",
		Long: `Scan the files and file names of a template for {{PLACEHOLDERS}} and say which are built in and
which you must supply with --var KEY=VALUE.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: templateCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			t, ok := cfg.FindTemplate(args[0])
			if !ok {
				return notFound("template", args[0], templateNames(cfg))
			}
			files, err := template.ListFiles(t.Path)
			if err != nil {
				return err
			}
			builtin := map[string]bool{}
			for _, n := range render.BuiltinNames() {
				builtin[n] = true
			}
			used := map[string]bool{}
			for _, rel := range files {
				for _, n := range render.Used(rel) {
					used[n] = true
				}
				data, err := os.ReadFile(filepath.Join(t.Path, filepath.FromSlash(rel)))
				if err != nil || strings.IndexByte(string(data[:min(len(data), 8000)]), 0) >= 0 {
					continue
				}
				for _, n := range render.Used(string(data)) {
					used[n] = true
				}
			}
			var names []string
			for n := range used {
				names = append(names, n)
			}
			sort.Strings(names)
			out := cmd.OutOrStdout()
			if len(names) == 0 {
				fmt.Fprintln(out, "This template uses no placeholders.")
				return nil
			}
			for _, n := range names {
				if builtin[n] {
					fmt.Fprintf(out, "  %-24s built in\n", n)
				} else {
					fmt.Fprintf(out, "  %-24s needs --var %s=VALUE\n", n, n)
				}
			}
			return nil
		},
	}
}
