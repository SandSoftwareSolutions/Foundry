# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/),
and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.5.0] - 2026-10-16

### Added
- **Placeholders are asked for.** A template can declare them in `.foundry.yaml` (`vars:` with a prompt and a
  default); in a terminal Foundry asks, with `--non-interactive` the default is used, and any other used-but-missing
  placeholder is asked for too. `--strict` stops before writing anything when a placeholder has no value, which is
  what you want in CI.
- `foundry template init <folder>` creates the skeleton of a new template (`--add` saves it right away).
- `foundry template add <name> <git-address>` keeps a copy of a template from Git (`--branch`);
  `template update` fast-forwards it, `template remove` deletes the copy. Setup commands of such templates are
  never run.
- `foundry template show` and `template vars` show the source of a template and the placeholders that are asked for.

## [0.4.0] - 2026-10-16

### Added
- Templates can declare their own setup steps and next steps in a `.foundry.yaml` (see `docs/templates.md`),
  for example `pnpm install` instead of the default `npm install`. `post: []` means "no setup at all". Placeholders
  such as `{{PROJECT_NAME_KEBAB}}` work in these lines. The file is not copied into projects, and the commands are
  not run for templates fetched with `--git`.

## [0.3.0] - 2026-10-16

### Added
- **Built-in templates** that ship inside the binary: `go-basic`, `go-http`, `node-basic`, `node-http`,
  `typescript-basic`, `python-basic`, `python-http`, `rust-basic`, `static-site` and `blank`. `foundry new` works
  right after installing; `--language` uses the built-in template of that language unless you set another.
- `foundry template export <name> <folder>` copies a built-in template so you can change it.
- `foundry template list --builtin` and `--saved`; built-in templates can be the default of a language.
- `scripts/test-templates.sh` and a CI job that create a project from every built-in template and run its
  own build and tests.

### Changed
- `foundry template list` shows built-in and saved templates (before: only saved ones). A template you saved
  hides a built-in one with the same name.
- The example templates under `examples/` became built-in templates.

## [0.2.0] - 2026-10-16

### Security
- Post-create steps no longer go through `bash -lc` with the project directory pasted into the
  command line. A project name containing quotes could run arbitrary commands. Steps now start
  the tool directly with the project as working directory.
- `--git` and `--branch` values starting with `-` are refused, and `git clone` is called with `--`.
- File names are checked after placeholder replacement: a value that would write outside the project
  (`/`, `\` or `..`) is refused. Project names with path separators are rejected.
- Symbolic links in a template are skipped (before, the content of the link target was copied).

### Fixed
- The first run never created the config or detected tools; it does now.
- `foundry new` no longer runs `npm run dev` or `python main.py` after creating a project (these
  blocked the command forever).
- `foundry config <language> ""` clears a default, as the README always said.
- Removing a template that was a language default no longer leaves a dangling default.
- Placeholder replacement is deterministic and single-pass; `--var` values override built-ins in a
  fixed way. Before, the result depended on map iteration order.
- `.foundryignore` supports what its documentation promised: `/dist/`, `**`, `!` and `?`.
  Leading slashes did not work before.
- Language detection no longer treats a `Makefile` as a language and breaks ties alphabetically.
- A failed run removes the partial project instead of leaving it behind.
- `--git` now uses the repository as a template (placeholders, ignore rules, fresh history).
- Commands no longer read or create the config at start-up, so `--version` and `--help` have no
  side effects. The config file is written atomically.
- The first commit works on machines where git has no `user.name` set.
- `.gitignore` files are built in. No `curl` and no network are needed.

### Added
- CI/CD: test matrix (Linux, macOS, Windows; oldest and newest Go) with the race detector and coverage,
  gofmt, vet, `go mod tidy` check, staticcheck, govulncheck, cross-compile check, end-to-end smoke
  test, CodeQL, Dependabot, and a tag-driven release workflow that builds archives and `.deb`
  packages, attests build provenance and creates a draft release. See `docs/releasing.md`.
- `foundry template update`, `edit` and `vars`; `foundry version`; `foundry config --json --path`.
- `foundry new --dry-run --json`, `--branch`, `--no-open`; warnings for placeholders without a value.
- Placeholders `{{LICENSE}}`, `{{YEAR}}`, `{{DATE}}` and `{{PROJECT_NAME_SNAKE|KEBAB|PASCAL|CAMEL}}`.
- Placeholders in file and directory names.
- `foundry template list --json`.
- Case-insensitive language and template lookup.
- Tests for every package and for the command line; CI workflow; release workflow that creates a draft.
- Cross-platform release script (`scripts/dist.sh`) and `Makefile`.

### Changed
- The program now lives in `cmd/foundry`, so `go install github.com/SandSoftwareSolutions/Foundry/cmd/foundry@latest` installs a binary called `foundry` (before, it would have been called `Foundry`). The command line code moved to `internal/cli`.
- Minimum Go version is now 1.25 (needed for the patched `golang.org/x/sys`; older Go releases are out of support).
- Dependencies updated: `golang.org/x/text` and `golang.org/x/sys` (govulncheck reported vulnerabilities in the old versions; the code did not call them).
- The Go module path is now `github.com/SandSoftwareSolutions/Foundry`.
- Library code lives under `internal/`; commands return errors instead of exiting.
- Build scripts no longer rewrite source files or contain machine specific paths.
- `.foundryignore` is no longer copied into new projects.
