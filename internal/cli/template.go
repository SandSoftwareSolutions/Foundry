package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/SandSoftwareSolutions/Foundry/internal/gitutil"
	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/SandSoftwareSolutions/Foundry/internal/template"
	"github.com/SandSoftwareSolutions/Foundry/templates"
	"github.com/spf13/cobra"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates", "t"},
		Short:   "Manage saved templates",
		Long: `Save directories as templates, list them, inspect them and keep them up to date.

Foundry ships with built-in templates for common starts (see "foundry template list"); you can use them
right away and save your own next to them. A template is a normal directory. Put a .foundryignore file in it to leave things out
(same rules as .gitignore). node_modules, .git, vendor, dist, build, .venv and __pycache__ are
always left out unless you re-include them with a rule such as "!build/".`,
	}
	cmd.AddCommand(newTemplateAddCmd(), newTemplateListCmd(), newTemplateShowCmd(), newTemplateRemoveCmd(),
		newTemplateUpdateCmd(), newTemplateEditCmd(), newTemplateVarsCmd(), newTemplateExportCmd(), newTemplateInitCmd())
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
	for _, n := range everyName(cfg) {
		if strings.HasPrefix(strings.ToLower(n), strings.ToLower(toComplete)) {
			out = append(out, n)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func newTemplateAddCmd() *cobra.Command {
	var description, language, branch string
	cmd := &cobra.Command{
		Use:   "add <name> <path-or-git-url>",
		Short: "Save a directory or a Git repository as a template",
		Long: `Scan a directory and save it as a template. The language is detected from the files;
override it with --language, for example to tag a framework such as React or Vue.
Adding a name that exists replaces that template.

Give a Git address instead of a directory and Foundry keeps its own copy in its config folder; refresh it with
"foundry template update". Setup commands (.foundry.yaml "post") of a template from Git are never run.`,
		Example: `  foundry template add go-cli ./my-go-template
  foundry template add react-starter ~/templates/react --description "React with TypeScript" --language React
  foundry template add team-api https://github.com/acme/api-template --branch main`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, path := args[0], args[1]
			if err := template.ValidateName(name); err != nil {
				return usagef("%v", err)
			}
			source := ""
			if gitutil.IsURL(path) {
				if !gitutil.Available() {
					return fmt.Errorf("git is required to add a template from a Git address but was not found on PATH")
				}
				dest, err := templateFolder(name)
				if err != nil {
					return err
				}
				if err := os.RemoveAll(dest); err != nil {
					return err
				}
				infof(cmd.OutOrStdout(), "Fetching %s ...", path)
				if err := gitutil.Clone(cmd.Context(), path, branch, dest); err != nil {
					return fmt.Errorf("cannot fetch the template: %w", err)
				}
				source, path = path, dest
			} else if branch != "" {
				return usagef("--branch only works with a Git address")
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
			cfg.UpsertTemplate(config.Template{Name: name, Path: info.Path, Language: info.Language, Description: description, Files: info.Files, Source: source, Ref: branch})
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
	cmd.Flags().StringVar(&branch, "branch", "", "Branch or tag to fetch (with a Git address)")
	return cmd
}

// templateFolder is where Foundry keeps its own copies of templates fetched from Git.
func templateFolder(name string) (string, error) {
	p, err := config.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), "templates", name), nil
}

func newTemplateListCmd() *cobra.Command {
	var sortBy string
	var quiet, asJSON, onlySaved, onlyBuiltin bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the built-in templates and your saved ones",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sortBy != "name" && sortBy != "language" {
				return usagef("--sort must be name or language")
			}
			if onlySaved && onlyBuiltin {
				return usagef("choose at most one of --saved and --builtin")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			var list []entry
			for _, e := range everyTemplate(cfg) {
				if (onlySaved && e.Builtin) || (onlyBuiltin && !e.Builtin) {
					continue
				}
				list = append(list, e)
			}
			if sortBy == "language" {
				sort.SliceStable(list, func(i, j int) bool { return list[i].Language < list[j].Language })
			}
			out := cmd.OutOrStdout()
			if asJSON {
				type item struct {
					config.Template
					Builtin    bool     `json:"builtin"`
					DefaultFor []string `json:"default_for"`
					Exists     bool     `json:"exists"`
				}
				items := make([]item, 0, len(list))
				for _, e := range list {
					exists := true
					if !e.Builtin {
						_, err := os.Stat(e.Path)
						exists = err == nil
					}
					d := cfg.DefaultFor(e.Name)
					if d == nil {
						d = []string{}
					}
					e.Files = nil
					items = append(items, item{Template: e.Template, Builtin: e.Builtin, DefaultFor: d, Exists: exists})
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}
			if len(list) == 0 {
				if !quiet {
					fmt.Fprintln(out, "No templates here.\nSave one with: foundry template add <name> <path>")
				}
				return nil
			}
			if quiet {
				for _, e := range list {
					fmt.Fprintln(out, e.Name)
				}
				return nil
			}
			printGroup := func(title string, builtin bool) {
				var group []entry
				for _, e := range list {
					if e.Builtin == builtin {
						group = append(group, e)
					}
				}
				if len(group) == 0 {
					return
				}
				boldf(out, "%s (%d)", title, len(group))
				for _, e := range group {
					where := e.Path
					if e.Builtin {
						where = e.Description
					}
					fmt.Fprintf(out, "  %-20s %-12s %s\n", e.Name, e.Language, where)
					if !e.Builtin && e.Description != "" {
						fmt.Fprintf(out, "      %s\n", e.Description)
					}
					if d := cfg.DefaultFor(e.Name); len(d) > 0 {
						infof(out, "      default for: %s", strings.Join(d, ", "))
					}
					if !e.Builtin {
						if _, err := os.Stat(e.Path); err != nil {
							warnf(out, "      path no longer exists")
						}
					}
				}
				fmt.Fprintln(out)
			}
			printGroup("Your templates", false)
			printGroup("Built-in templates", true)
			fmt.Fprintln(out, "Use one with: foundry new <name> --template <template>   (or --language <Language>)")
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort", "name", "Sort by name or language")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Only print the names")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print as JSON")
	cmd.Flags().BoolVar(&onlySaved, "saved", false, "Only the templates you saved")
	cmd.Flags().BoolVar(&onlyBuiltin, "builtin", false, "Only the templates that ship with Foundry")
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
			e, ok := lookup(cfg, args[0])
			if !ok {
				return notFound("template", args[0], everyName(cfg))
			}
			t := &e.Template
			if e.Builtin {
				files, err := templates.Files(e.Name)
				if err != nil {
					return err
				}
				t.Files = files
			}
			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(t)
			}
			if !filesOnly {
				if e.Builtin {
					fmt.Fprintf(out, "Template: %s (built-in)\nLanguage: %s\n", t.Name, t.Language)
				} else {
					fmt.Fprintf(out, "Template: %s\nLanguage: %s\nPath:     %s\n", t.Name, t.Language, t.Path)
					if t.Source != "" {
						fmt.Fprintf(out, "Source:   %s\n", t.Source)
					}
				}
				if t.Description != "" {
					fmt.Fprintf(out, "About:    %s\n", t.Description)
				}
				if d := cfg.DefaultFor(t.Name); len(d) > 0 {
					infof(out, "Default for: %s", strings.Join(d, ", "))
				}
				if _, err := os.Stat(t.Path); !e.Builtin && err != nil {
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
				if e, found := lookup(cfg, args[0]); found && e.Builtin {
					return builtinChangeError(e.Name)
				}
				return notFound("template", args[0], everyName(cfg))
			}
			name := t.Name
			if d := cfg.DefaultFor(name); len(d) > 0 && !force {
				return usagef("template %q is the default for: %s\nUse --force to remove it anyway (the defaults are cleared).", name, strings.Join(d, ", "))
			}
			removedCopy := ""
			if t.Source != "" {
				if base, err := templateFolder(""); err == nil && isInside(base, t.Path) {
					removedCopy = t.Path
				}
			}
			cleared, err := cfg.RemoveTemplate(name)
			if err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			if removedCopy != "" {
				_ = os.RemoveAll(removedCopy)
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
					if e, found := lookup(cfg, n); found && e.Builtin {
						return builtinChangeError(e.Name)
					}
					return notFound("template", n, templateNames(cfg))
				}
				if t.Source != "" {
					if err := gitutil.Pull(cmd.Context(), t.Path); err != nil {
						warnf(cmd.ErrOrStderr(), "%s: cannot update from %s: %v", t.Name, t.Source, err)
						failed++
						continue
					}
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
				if e, found := lookup(cfg, args[0]); found && e.Builtin {
					return builtinChangeError(e.Name)
				}
				return notFound("template", args[0], everyName(cfg))
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
			e, ok := lookup(cfg, args[0])
			if !ok {
				return notFound("template", args[0], everyName(cfg))
			}
			t, cleanup, err := materialize(e)
			if err != nil {
				return err
			}
			defer cleanup()
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
			declared := map[string]template.VarSpec{}
			if meta, err := template.LoadMeta(t.Path); err == nil {
				for _, v := range meta.Vars {
					declared[v.Name] = v
				}
			}
			for _, n := range names {
				switch spec, isDeclared := declared[n]; {
				case builtin[n]:
					fmt.Fprintf(out, "  %-24s built in\n", n)
				case isDeclared && spec.Default != "":
					fmt.Fprintf(out, "  %-24s asked for; default %q (or --var %s=VALUE)\n", n, spec.Default, n)
				case isDeclared:
					fmt.Fprintf(out, "  %-24s asked for (or --var %s=VALUE)\n", n, n)
				default:
					fmt.Fprintf(out, "  %-24s needs --var %s=VALUE\n", n, n)
				}
			}
			return nil
		},
	}
}

func newTemplateExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <name> <folder>",
		Short: "Copy a built-in template into a folder so you can change it",
		Long: `Write the files of a built-in template to a new folder. Edit them as you like, then save the folder
as your own template with "foundry template add".`,
		Example: `  foundry template export go-basic ./my-go
  foundry template add my-go ./my-go --language Go`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: templateCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			e, ok := lookup(cfg, args[0])
			if !ok {
				return notFound("template", args[0], everyName(cfg))
			}
			if !e.Builtin {
				return usagef("%q is one of your own templates; it already is a folder: %s", e.Name, e.Path)
			}
			if err := templates.Extract(e.Name, args[1]); err != nil {
				return err
			}
			okf(cmd.OutOrStdout(), "Wrote the %q template to %s", e.Name, args[1])
			fmt.Fprintf(cmd.OutOrStdout(), "  Save it as your own with: foundry template add <name> %s --language %s\n", args[1], e.Language)
			return nil
		},
	}
}

// isInside reports whether path is below base (so Foundry never deletes a folder it did not create).
func isInside(base, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(path))
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}

func newTemplateInitCmd() *cobra.Command {
	var addAs, language, description string
	cmd := &cobra.Command{
		Use:   "init <folder>",
		Short: "Create the skeleton of a new template",
		Long: `Create a folder with the files every template needs: a README that uses placeholders, a .foundry.yaml that
shows the setup options (commented out), a .foundryignore and an example file. Fill it with your own files,
then save it with "foundry template add" (or pass --add <name> to do that right away).`,
		Example: `  foundry template init ./my-template
  foundry template init ./my-api --add my-api --language Go --description "Our Go API"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if addAs != "" {
				if err := template.ValidateName(addAs); err != nil {
					return usagef("%v", err)
				}
			}
			if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
				return usagef("%q already exists and is not empty", dir)
			}
			files := map[string]string{
				"README.md":         "# {{PROJECT_NAME}}\n\nCreated from a Foundry template by {{AUTHOR}} on {{DATE}}.\n\nDescribe what this project is for and how to run it.\n",
				"notes.txt":         "This file shows placeholders: the project is called {{PROJECT_NAME}} ({{PROJECT_NAME_KEBAB}}), made in {{YEAR}}.\nDelete it and add your own files.\n",
				template.IgnoreFile: "# Files and folders that are left out of the template, one pattern per line (same rules as .gitignore).\n# Always left out: .git, node_modules, .venv, __pycache__, vendor, dist, build.\n# Re-include one with a rule such as: !build/\n",
				template.MetaFile: `# Optional. What Foundry does after it created a project from this template.
#
# Placeholders to ask for. With --non-interactive the default is used; without a default it must be given with --var.
# vars:
#   - name: PORT
#     prompt: Port the service listens on
#     default: "8080"
#
# Setup commands, run in the new project. One program per line, no shell (no &&, pipes or $VARIABLES).
# "post: []" means: no setup at all. They are not run for templates that come from a Git address.
# post:
#   - go mod tidy
#
# What to tell the user to do next.
# next:
#   - go run .
`,
			}
			for rel, content := range files {
				p := filepath.Join(dir, rel)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					return err
				}
			}
			out := cmd.OutOrStdout()
			okf(out, "Created the template skeleton in %s", dir)
			if addAs == "" {
				fmt.Fprintf(out, "  Next: add your files, then  foundry template add <name> %s\n", dir)
				return nil
			}
			info, err := template.Scan(dir)
			if err != nil {
				return err
			}
			if strings.TrimSpace(language) != "" {
				info.Language = strings.TrimSpace(language)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.UpsertTemplate(config.Template{Name: addAs, Path: info.Path, Language: info.Language, Description: description, Files: info.Files})
			if err := config.Save(cfg); err != nil {
				return err
			}
			okf(out, "Saved it as template %q (language %s)", addAs, info.Language)
			fmt.Fprintf(out, "  Try it: foundry new demo --template %s --dry-run\n", addAs)
			return nil
		},
	}
	cmd.Flags().StringVar(&addAs, "add", "", "Also save the new template under this name")
	cmd.Flags().StringVarP(&language, "language", "l", "", "Language tag for --add")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description for --add")
	return cmd
}
