package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/SandSoftwareSolutions/Foundry/internal/config"
	"github.com/SandSoftwareSolutions/Foundry/internal/gitutil"
	"github.com/SandSoftwareSolutions/Foundry/internal/post"
	"github.com/SandSoftwareSolutions/Foundry/internal/project"
	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/SandSoftwareSolutions/Foundry/internal/template"
	"github.com/SandSoftwareSolutions/Foundry/templates"
	"github.com/spf13/cobra"
)

const pageSize = 10

type newOptions struct {
	language, templateName, gitURL, branch, path string
	noGit, noPost, noOpen, nonInteractive        bool
	dryRun, asJSON, strict                       bool
	vars                                         []string
}

func newNewCmd() *cobra.Command {
	var o newOptions
	cmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Create a new project from a template",
		Long: `Create a new project from a saved template or from a Git repository.

The template is chosen with --template, with --language (the default template of that language),
with --git (a repository used as a template), or interactively.

Foundry then:
  - copies the files, skipping what .foundryignore excludes
  - replaces placeholders such as {{PROJECT_NAME}} and {{AUTHOR}} in file contents and file names
  - runs the setup step of the language (go mod tidy, npm install), unless --no-post
  - initialises a git repository with a .gitignore and a first commit, unless --no-git

Placeholders: {{PROJECT_NAME}}, {{PROJECT_NAME_LOWER}}, {{PROJECT_NAME_UPPER}}, {{PROJECT_NAME_SNAKE}},
{{PROJECT_NAME_KEBAB}}, {{PROJECT_NAME_PASCAL}}, {{PROJECT_NAME_CAMEL}}, {{AUTHOR}}, {{LICENSE}},
{{YEAR}}, {{DATE}}, plus anything you pass with --var KEY=VALUE.`,
		Example: `  foundry new my-api --language Go
  foundry new my-app --template react-starter --var PORT=3000
  foundry new my-tool --git https://github.com/you/template-repo --branch main
  foundry new my-app --template go-cli --path ~/projects --no-git
  foundry new my-app --template go-cli --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(cmd, args[0], o)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.language, "language", "l", "", "Use the default template of this language")
	f.StringVarP(&o.templateName, "template", "t", "", "Use this saved template")
	f.StringVarP(&o.gitURL, "git", "g", "", "Use a Git repository as the template")
	f.StringVar(&o.branch, "branch", "", "Branch or tag to fetch with --git")
	f.StringVarP(&o.path, "path", "p", "", "Parent directory of the new project (default: current directory)")
	f.BoolVar(&o.noGit, "no-git", false, "Do not initialise a git repository")
	f.BoolVar(&o.noPost, "no-post", false, "Skip the language setup step (go mod tidy, npm install)")
	f.BoolVar(&o.noOpen, "no-open", false, "Do not open the project in VS Code")
	f.BoolVar(&o.nonInteractive, "non-interactive", false, "Never prompt; require --template, --language or --git")
	f.StringArrayVar(&o.vars, "var", nil, "Placeholder value as KEY=VALUE (repeatable)")
	f.BoolVar(&o.dryRun, "dry-run", false, "Show what would be created without writing anything")
	f.BoolVar(&o.strict, "strict", false, "Stop (before writing anything) when a placeholder has no value")
	f.BoolVar(&o.asJSON, "json", false, "With --dry-run: print the plan as JSON")
	return cmd
}

// source is where the files of the new project come from.
type source struct {
	fromGit  bool   // fetched with --git: its setup commands are not trusted
	dir      string // template directory
	language string
	label    string // shown to the user
}

func runNew(cmd *cobra.Command, name string, o newOptions) error {
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	if err := project.ValidateName(name); err != nil {
		return usagef("%v", err)
	}
	chosen := 0
	for _, s := range []string{o.templateName, o.language, o.gitURL} {
		if s != "" {
			chosen++
		}
	}
	if chosen > 1 {
		return usagef("choose only one of --template, --language and --git")
	}
	if o.asJSON && !o.dryRun {
		return usagef("--json only works together with --dry-run")
	}
	vars, err := parseVars(o.vars)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	target := filepath.Join(o.path, name)

	var src source
	if o.gitURL != "" {
		tmp, err := os.MkdirTemp("", "foundry-clone-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		if !gitutil.Available() {
			return fmt.Errorf("git is required for --git but was not found on PATH")
		}
		dir := filepath.Join(tmp, "repo")
		if !o.asJSON {
			infof(out, "Fetching %s ...", o.gitURL)
		}
		if err := gitutil.Clone(cmd.Context(), o.gitURL, o.branch, dir); err != nil {
			return fmt.Errorf("cannot fetch template: %w", err)
		}
		lang, err := template.DetectLanguage(dir)
		if err != nil {
			return err
		}
		src = source{fromGit: true, dir: dir, language: lang, label: o.gitURL}
	} else {
		tmpl, cleanup, err := selectTemplate(cfg, o)
		if err != nil {
			return err
		}
		defer cleanup()
		label := tmpl.Name
		if tmpl.Description != "" && isBuiltinName(tmpl.Name, cfg) {
			label = tmpl.Name + " (built-in)"
		}
		src = source{fromGit: tmpl.Source != "", dir: tmpl.Path, language: tmpl.Language, label: label}
	}

	meta := loadMeta(cmd, src)
	// Ask only on a terminal, and never when the output must be machine readable.
	var ask asker
	if !o.nonInteractive && !o.asJSON && cfg.Interactive && interactive() {
		ask = surveyAsk
	}
	vars, err = fillDeclared(meta.Vars, vars, ask)
	if err != nil {
		return err
	}
	build := func(v render.Vars) (*project.Plan, error) {
		return project.BuildPlan(project.Options{
			TemplateDir: src.dir, Name: name, TargetDir: target,
			Author: cfg.Author, License: cfg.License, Vars: v,
		})
	}
	plan, err := build(vars)
	if err != nil {
		return err
	}
	if missing := unresolvedNames(plan); len(missing) > 0 {
		switch {
		case ask != nil:
			if vars, err = askMissing(missing, vars, ask); err != nil {
				return err
			}
			if plan, err = build(vars); err != nil {
				return err
			}
		case o.strict:
			return strictError(plan)
		}
	}
	if o.strict && len(plan.Unresolved) > 0 {
		return strictError(plan)
	}
	meta = renderMeta(meta, plan.Vars)

	if o.dryRun {
		return printDryRun(out, plan, src, target, o.asJSON)
	}

	boldf(out, "Creating project %q from %s", name, src.label)
	fmt.Fprintf(out, "  Language: %s\n  Target:   %s\n", src.language, target)
	if err := plan.Execute(); err != nil {
		return fmt.Errorf("cannot create project: %w", err)
	}
	okf(out, "Created %d files in %s", len(plan.Files()), target)
	for _, w := range plan.Warnings {
		warnf(errOut, "%s", w)
	}
	if lines := plan.UnresolvedSummary(); len(lines) > 0 {
		warnf(errOut, "Some placeholders have no value and were left as they are (set them with --var KEY=VALUE):")
		for _, l := range lines {
			fmt.Fprintf(errOut, "    %s\n", l)
		}
	}

	runPost(cmd, o, src.language, target, meta)
	setupGit(cmd, o, cfg, src.language, target)
	maybeOpenEditor(cmd, o, cfg, target)
	printNextSteps(out, target, src.language, meta)
	return nil
}

// renderMeta fills placeholders such as {{PROJECT_NAME_KEBAB}} into the declared commands and hints, so a
// template can say "git remote add origin git@github.com:org/{{PROJECT_NAME_KEBAB}}.git". An absent list stays
// absent: an empty "post: []" is different from no "post" at all.
func renderMeta(m template.Meta, vars render.Vars) template.Meta {
	apply := func(in []string) []string {
		if in == nil {
			return nil
		}
		out := make([]string, len(in))
		for i, l := range in {
			out[i] = render.Apply(l, vars)
		}
		return out
	}
	return template.Meta{Post: apply(m.Post), Next: apply(m.Next)}
}

// loadMeta reads the setup steps a template declares. A repository fetched with --git is not trusted to run
// commands, so its declaration is ignored (with a note).
func loadMeta(cmd *cobra.Command, src source) template.Meta {
	meta, err := template.LoadMeta(src.dir)
	if err != nil {
		warnf(cmd.ErrOrStderr(), "%v", err)
		return template.Meta{}
	}
	if src.fromGit && meta.Post != nil {
		infof(cmd.OutOrStdout(), "The template repository lists setup commands; they are not run for a template fetched with --git.")
		return template.Meta{Vars: meta.Vars, Next: meta.Next}
	}
	return meta
}

func runPost(cmd *cobra.Command, o newOptions, language, target string, meta template.Meta) {
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	if o.noPost {
		infof(out, "Skipped the setup step (--no-post).")
		return
	}
	steps := post.StepsFor(language, target)
	if meta.Post != nil { // an empty list ("post: []") means: no setup steps at all
		declared, err := post.FromCommands(meta.Post)
		if err != nil {
			warnf(errOut, "The template's setup commands are not valid: %v", err)
			return
		}
		steps = declared
	}
	for _, s := range steps {
		infof(out, "%s: %s", s.Name, s.String())
	}
	if len(steps) == 0 {
		return
	}
	if err := post.Run(cmd.Context(), target, steps, out, errOut); err != nil {
		warnf(errOut, "Setup step failed: %v", err)
		return
	}
	okf(out, "Setup finished.")
}

func setupGit(cmd *cobra.Command, o newOptions, cfg *config.Config, language, target string) {
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	if o.noGit {
		infof(out, "Skipped git (--no-git).")
		return
	}
	if !gitutil.Available() {
		warnf(errOut, "git was not found on PATH; skipped git initialisation.")
		return
	}
	res, err := gitutil.InitAndCommit(cmd.Context(), target, language, cfg.Author, "Initial commit from Foundry")
	if err != nil {
		warnf(errOut, "Git setup failed: %v", err)
		return
	}
	okf(out, "Initialised a git repository.")
	if res.GitignoreWritten {
		if res.GitignoreSpecific {
			okf(out, "Added a .gitignore for %s.", language)
		} else {
			okf(out, "Added a generic .gitignore (no specific one for %q).", language)
		}
	}
	if res.Committed {
		okf(out, "Created the first commit.")
	}
	if res.UsedFallbackIdentity {
		warnf(errOut, "git has no user.name/user.email; the first commit used %q <foundry@localhost>.", fallbackName(cfg.Author))
	}
}

func fallbackName(author string) string {
	if strings.TrimSpace(author) == "" {
		return "Foundry"
	}
	return author
}

func maybeOpenEditor(cmd *cobra.Command, o newOptions, cfg *config.Config, target string) {
	if o.noOpen || o.nonInteractive || cfg.VSCodePath == "" || !interactive() {
		return
	}
	c := exec.Command(cfg.VSCodePath, target)
	if err := c.Start(); err != nil {
		warnf(cmd.ErrOrStderr(), "Could not open VS Code: %v", err)
		return
	}
	okf(cmd.OutOrStdout(), "Opened the project in VS Code.")
}

func printNextSteps(out io.Writer, target, language string, meta template.Meta) {
	fmt.Fprintln(out)
	boldf(out, "Next steps")
	fmt.Fprintf(out, "  cd %s\n", shellQuote(target))
	next := post.NextSteps(language)
	if len(meta.Next) > 0 {
		next = meta.Next
	}
	for _, s := range next {
		fmt.Fprintf(out, "  %s\n", s)
	}
}

func shellQuote(p string) string {
	if strings.ContainsAny(p, " \t'\"$&;()") {
		return `"` + strings.ReplaceAll(p, `"`, `\"`) + `"`
	}
	return p
}

type dryRunReport struct {
	Template   string              `json:"template"`
	Language   string              `json:"language"`
	Target     string              `json:"target"`
	Files      []string            `json:"files"`
	Warnings   []string            `json:"warnings"`
	Unresolved map[string][]string `json:"unresolved"`
}

func printDryRun(out io.Writer, plan *project.Plan, src source, target string, asJSON bool) error {
	files := plan.Files()
	if asJSON {
		warnings := plan.Warnings
		if warnings == nil {
			warnings = []string{}
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(dryRunReport{Template: src.label, Language: src.language, Target: target,
			Files: files, Warnings: warnings, Unresolved: plan.Unresolved})
	}
	boldf(out, "Dry run: nothing will be written")
	fmt.Fprintf(out, "  Template: %s\n  Language: %s\n  Target:   %s\n", src.label, src.language, target)
	fmt.Fprintf(out, "\n  Would create %d files:\n", len(files))
	const show = 50
	for i, f := range files {
		if i == show {
			fmt.Fprintf(out, "    ... and %d more\n", len(files)-show)
			break
		}
		fmt.Fprintf(out, "    %s\n", f)
	}
	for _, w := range plan.Warnings {
		warnf(out, "%s", w)
	}
	if lines := plan.UnresolvedSummary(); len(lines) > 0 {
		warnf(out, "Placeholders without a value (set them with --var KEY=VALUE):")
		for _, l := range lines {
			fmt.Fprintf(out, "    %s\n", l)
		}
	}
	return nil
}

// isBuiltinName reports whether name refers to a built-in template that the user has not shadowed.
func isBuiltinName(name string, cfg *config.Config) bool {
	e, ok := lookup(cfg, name)
	return ok && e.Builtin
}

// selectTemplate resolves the template from the flags, or asks the user. The returned function removes any
// temporary files and must always be called.
func selectTemplate(cfg *config.Config, o newOptions) (config.Template, func(), error) {
	none := func() {}
	pick := func(e entry) (config.Template, func(), error) { return materialize(e) }
	switch {
	case o.templateName != "":
		e, ok := lookup(cfg, o.templateName)
		if !ok {
			return config.Template{}, none, notFound("template", o.templateName, everyName(cfg))
		}
		return pick(e)
	case o.language != "":
		if name := cfg.LanguageDefault(o.language); name != "" {
			e, ok := lookup(cfg, name)
			if !ok {
				return config.Template{}, none, fmt.Errorf("the default template %q for %s no longer exists; set a new one with: foundry config %s <template-name>", name, o.language, o.language)
			}
			return pick(e)
		}
		if b, ok := templates.DefaultFor(o.language); ok {
			e, _ := lookup(cfg, b.Name)
			return pick(e)
		}
		return config.Template{}, none, fmt.Errorf("there is no template for language %q yet\n  See what is available:  foundry template list\n  Or choose one:            foundry new <name> --template <template-name>", o.language)
	}
	list := everyTemplate(cfg)
	if o.nonInteractive || !cfg.Interactive || !interactive() {
		var b strings.Builder
		b.WriteString("choose a template with --template, --language or --git. Available templates:\n")
		for _, t := range list {
			fmt.Fprintf(&b, "  %-24s %s\n", t.Name, t.Language)
		}
		return config.Template{}, none, usagef("%s", strings.TrimRight(b.String(), "\n"))
	}
	e, err := pickInteractively(cfg, list)
	if err != nil {
		return config.Template{}, none, err
	}
	return pick(e)
}

func pickInteractively(cfg *config.Config, list []entry) (entry, error) {
	langs := languagesOf(list)
	if len(langs) == 0 {
		return entry{}, fmt.Errorf("none of the templates has a language")
	}
	lang := langs[0]
	if len(langs) > 1 {
		if err := survey.AskOne(&survey.Select{Message: "Select a language:", Options: langs, PageSize: min(len(langs), pageSize)}, &lang); err != nil {
			return entry{}, fmt.Errorf("selection cancelled")
		}
	}
	var candidates []entry
	for _, t := range list {
		if t.Language == lang {
			candidates = append(candidates, t)
		}
	}
	labels := make([]string, len(candidates))
	for i, t := range candidates {
		labels[i] = t.Name
		if len(cfg.DefaultFor(t.Name)) > 0 {
			labels[i] += " (default)"
		}
		if t.Builtin {
			labels[i] += " (built-in)"
		}
	}
	var picked string
	if err := survey.AskOne(&survey.Select{Message: fmt.Sprintf("Select a %s template:", lang), Options: labels, PageSize: min(len(labels), pageSize)}, &picked); err != nil {
		return entry{}, fmt.Errorf("selection cancelled")
	}
	for i, l := range labels {
		if l == picked {
			return candidates[i], nil
		}
	}
	return entry{}, fmt.Errorf("template not found")
}

func templateNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Templates))
	for _, t := range cfg.Templates {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return names
}

func notFound(kind, name string, available []string) error {
	if len(available) == 0 {
		return fmt.Errorf("%s %q not found (none saved yet)", kind, name)
	}
	return fmt.Errorf("%s %q not found; available: %s", kind, name, strings.Join(available, ", "))
}
