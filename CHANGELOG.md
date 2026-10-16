# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/),
and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
- `foundry template update`, `edit` and `vars`; `foundry version`; `foundry config --json --path`.
- `foundry new --dry-run --json`, `--branch`, `--no-open`; warnings for placeholders without a value.
- Placeholders `{{LICENSE}}`, `{{YEAR}}`, `{{DATE}}` and `{{PROJECT_NAME_SNAKE|KEBAB|PASCAL|CAMEL}}`.
- Placeholders in file and directory names.
- `foundry template list --json`.
- Case-insensitive language and template lookup.
- Tests for every package and for the command line; CI workflow; release workflow that creates a draft.
- Cross-platform release script (`scripts/dist.sh`) and `Makefile`.

### Changed
- The Go module path is now `github.com/SandSoftwareSolutions/Foundry`.
- Library code lives under `internal/`; commands return errors instead of exiting.
- Build scripts no longer rewrite source files or contain machine specific paths.
- `.foundryignore` is no longer copied into new projects.
