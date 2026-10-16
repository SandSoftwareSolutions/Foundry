# Writing templates

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

## Language

Foundry guesses the language from the files (`go.mod`, `package.json`, `.py` files ...). Override it
when you add the template: `foundry template add react-starter ./react --language React`.
The language picks the `.gitignore`, the setup step (`go mod tidy` for Go, `npm install` for
JavaScript, TypeScript, React and Vue) and the hints shown at the end.

## Git repositories as templates

`foundry new my-app --git https://github.com/you/template [--branch main]` fetches a shallow clone,
uses it like a template (placeholders, `.foundryignore`) and starts a fresh history. The history of
the template is not copied.
