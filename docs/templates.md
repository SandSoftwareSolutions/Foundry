# Writing templates

Foundry ships with built-in templates (`foundry template list --builtin`); everything below applies to them too,
and to your own. To start from a built-in one, run `foundry template export go-basic ./my-go`.

A Foundry template is an ordinary directory. Everything in it is copied into the new project, with
placeholders filled in.

## Placeholders

Write `{{NAME}}` anywhere in a text file, and in file or directory names.

| Placeholder | Example for `my-cool_App` |
| --- | --- |
| `{{PROJECT_NAME}}` | `my-cool_App` |
| `{{PROJECT_NAME_LOWER}}` | `my-cool_app` |
| `{{PROJECT_NAME_UPPER}}` | `MY-COOL_APP` |
| `{{PROJECT_NAME_SNAKE}}` | `my_cool_app` |
| `{{PROJECT_NAME_KEBAB}}` | `my-cool-app` |
| `{{PROJECT_NAME_PASCAL}}` | `MyCoolApp` |
| `{{PROJECT_NAME_CAMEL}}` | `myCoolApp` |
| `{{AUTHOR}}` | the name from `foundry config --user` |
| `{{LICENSE}}` | the license from `foundry config --license` |
| `{{YEAR}}`, `{{DATE}}` | `2026`, `2026-03-09` |

Your own placeholders work too: write `{{PORT}}` in the template and run
`foundry new app --template web --var PORT=3000`. A `--var` with the same name as a built-in wins.

Rules worth knowing:

- Replacement happens in one pass, so a value that contains `{{X}}` is not expanded again.
- Spaces inside the braces are fine: `{{ AUTHOR }}`.
- Unknown placeholders are left as they are, and Foundry lists them after creating the project
  (or in `--dry-run`). `foundry template vars <name>` lists them before you start.
- Binary files (anything with a NUL byte in the first 8000 bytes) are copied unchanged.
- A value that would put a `/`, `\` or `..` in a file name is refused.
- Symbolic links are skipped, with a warning.

## `.foundryignore`

Put a `.foundryignore` at the root of the template to leave things out. It works like `.gitignore`:

```gitignore
# comments and blank lines are fine
*.log            # any .log file at any depth
/dist/           # only the top-level dist directory
docs/*.md        # a pattern with a slash is anchored to the template root
**/cache/        # any cache directory
!keep.log        # re-include something an earlier rule excluded
```

These are always left out: `.git/`, `node_modules/`, `.venv/`, `__pycache__/`, `vendor/`, `dist/`,
`build/` and the `.foundryignore` file itself. A template that really needs one of them can
re-include it with a rule such as `!build/`.

## Starting a new template

`foundry template init ./my-template` creates the skeleton (README with placeholders, a commented `.foundry.yaml`,
a `.foundryignore` and an example file). Add `--add my-template --language Go` to save it right away.

## Templates from Git

`foundry template add team-api https://github.com/acme/api-template [--branch main]` keeps Foundry's own copy of
the repository in its config folder. `foundry template update` fetches new commits (fast-forward only), and
`foundry template remove` deletes the copy again. Setup commands (`post`) of such a template are **never run**;
its hints (`next`) and declared placeholders (`vars`) are used.

## Placeholders that are asked for

Declare them in `.foundry.yaml` so Foundry asks for them in a terminal, and uses the default when it may not ask:

```yaml
vars:
  - name: PORT
    prompt: Port the service listens on
    default: "8080"
```

A placeholder that is used but not declared is asked for too (in a terminal). `--var NAME=value` always wins,
`--non-interactive` never asks, and `--strict` stops before writing anything if a placeholder still has no value.
`foundry template vars <name>` shows what a template uses and what is asked.

## Setup steps: `.foundry.yaml`

A template can say what to do after a project was created from it. Put a `.foundry.yaml` at its root:

```yaml
post:            # commands to run in the new project, in order
  - pnpm install
next:            # what to tell the user to do next
  - pnpm dev
```

- `post` replaces the default setup of the language (`go mod tidy`, `npm install`). Each line is one program with
  arguments; there is no shell, so no `&&`, pipes, redirects or `$VARIABLES`. A line with shell syntax is refused
  and nothing from it runs.
- `post: []` (an empty list) means "no setup steps at all", for a template that needs none.
- `next` replaces the hints printed at the end.
- `--no-post` skips the commands.
- The file is never copied into the project.
- For a template fetched with `--git`, the `post` commands are **not run** (a repository you just downloaded is not
  trusted to run commands); the `next` hints are still shown.

## Language

Foundry guesses the language from the files (`go.mod`, `package.json`, `.py` files ...). Override it
when you add the template: `foundry template add react-starter ./react --language React`.
The language picks the `.gitignore`, the setup step (`go mod tidy` for Go, `npm install` for
JavaScript, TypeScript, React and Vue) and the hints shown at the end.

## Git repositories as templates

`foundry new my-app --git https://github.com/you/template [--branch main]` fetches a shallow clone,
uses it like a template (placeholders, `.foundryignore`) and starts a fresh history. The history of
the template is not copied.

## Built-in templates in the source

The built-in templates live in `templates/_builtin/<name>/` and are embedded in the binary. They are listed,
with a language and a description, in `templates/_builtin/catalog.json`. Two details:

- The directory starts with an underscore so that the Go tools ignore it: the Go sources of the templates are
  not compiled as part of Foundry.
- A `go.mod` would make a directory a separate module that cannot be embedded, so templates keep it as
  `go.mod.txt`; Foundry gives it its real name when it writes a project.

To add a template: create the directory, add it to `catalog.json`, add a line to `scripts/test-templates.sh`
that builds and tests the generated project, and run `make templates`.
