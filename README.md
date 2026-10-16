# Foundry

**Start every project from your own template, in one command.**

Foundry turns any directory into a reusable project template. Create a new project from it with the
name, author and license filled in, setup commands run and a first git commit made.

```console
$ foundry template add go-cli ./examples/go-cli
✓ Template "go-cli" saved

$ foundry config Go go-cli            # make it the default for Go
$ foundry new my-api --language Go
Creating project "my-api" from go-cli
✓ Created 3 files in my-api
✓ Initialised a git repository.
✓ Added a .gitignore for Go.
✓ Created the first commit.
```

## Why Foundry

- **Your templates, not somebody else's.** A template is a normal directory you can edit, test and
  keep in git. No generator language to learn.
- **Safe by default.** Nothing is written outside the new directory, symbolic links are not followed,
  a failed run removes what it created, and `--dry-run` shows exactly what would happen.
- **Fast and dependency free.** One static binary for Linux, macOS and Windows.
- **Good defaults.** `.gitignore` per language, `go mod tidy` / `npm install` after creation, a first
  commit, and a `--non-interactive` mode with JSON output for scripts.

## Install

**With Go** (1.25 or newer):

```console
go install github.com/SandSoftwareSolutions/Foundry/cmd/foundry@latest
```

**From a release:** download the archive for your system from the
[releases page](https://github.com/SandSoftwareSolutions/Foundry/releases), unpack it and put the
`foundry` binary on your `PATH`. Check the download against `SHA256SUMS`. Linux releases also come
as `.deb` packages.

**From source:**

```console
git clone https://github.com/SandSoftwareSolutions/Foundry
cd Foundry
make build        # produces ./foundry
```

Check it works with `foundry version`.

## Quick start

```console
# 1. Save a directory as a template (the language is detected)
foundry template add go-cli ./examples/go-cli --description "Go CLI starter"
foundry template add node-cli ./examples/node-cli

# 2. Optional: pick a default template per language
foundry config Go go-cli

# 3. Create a project
foundry new my-api --language Go              # default template of a language
foundry new my-tool --template node-cli       # a specific template
foundry new my-app                            # choose from menus
foundry new my-site --git https://github.com/you/template   # a repository as template

# 4. See what would happen without writing anything
foundry new my-api --template go-cli --dry-run
```

On the first run Foundry creates `~/.foundry/config.yaml`, detects the tools on your machine and
takes your name from `git config --global user.name`. Change it with `foundry config --user "Your Name"`.

## Placeholders

Write `{{PROJECT_NAME}}`, `{{AUTHOR}}` and friends in your template files **and file names**:

| Placeholder | Meaning |
| --- | --- |
| `{{PROJECT_NAME}}` | the name you gave, as written |
| `{{PROJECT_NAME_LOWER}}` `_UPPER` `_SNAKE` `_KEBAB` `_PASCAL` `_CAMEL` | case variants |
| `{{AUTHOR}}`, `{{LICENSE}}` | from `foundry config` |
| `{{YEAR}}`, `{{DATE}}` | today |
| `{{ANYTHING}}` | whatever you pass with `--var ANYTHING=value` |

`foundry template vars <name>` lists the placeholders a template uses and which need `--var`.
Details, ignore rules and tips: [docs/templates.md](docs/templates.md).

## Commands

| Command | What it does |
| --- | --- |
| `foundry new <name>` | Create a project. Flags: `--template`, `--language`, `--git`, `--branch`, `--path`, `--var KEY=VALUE`, `--dry-run`, `--json` (with `--dry-run`), `--no-git`, `--no-post`, `--no-open`, `--non-interactive` |
| `foundry template add <name> <path>` | Save a directory as a template (`--description`, `--language`) |
| `foundry template list` | List templates (`--sort name\|language`, `--quiet`, `--json`) |
| `foundry template show <name>` | Show details and files (`--files-only`, `--summary`, `--json`) |
| `foundry template update [name...]` | Rescan templates so file lists stay current |
| `foundry template edit <name>` | Change description, language, path or name |
| `foundry template vars <name>` | List the placeholders a template uses |
| `foundry template remove <name>` | Forget a template (`--force` if it is a default); files stay |
| `foundry config` | Show settings; set with `--user`, `--license`, `--default-language`, `--docker`, `--interactive`; `--json`, `--path` |
| `foundry config <language> <template>` | Set the default template of a language (`""` clears it) |
| `foundry detect` | Detect languages, package managers and tools (`--json`, `--yes`) |
| `foundry completion <shell>` | Shell completion for bash, zsh, fish and PowerShell |
| `foundry version` | Print version information |

Global flags: `--config <file>`, `--no-color`, `--color`. `NO_COLOR` is respected.

Exit status is `0` on success and `1` on any error. Errors go to stderr; machine readable output
(`--json`) goes to stdout.

## What happens during `foundry new`

1. The name is checked (it must be a single directory name; use `--path` for the parent).
2. The template is resolved and a **plan** is built: which files are copied, which are skipped, which
   placeholders have no value. `--dry-run` stops here.
3. Files are written. If anything fails, what was created is removed again.
4. The language setup runs (`go mod tidy` for Go when there is a `go.mod`, `npm install` for
   JavaScript, TypeScript, React and Vue when there is a `package.json`). It is a plain command
   with a time limit, never a shell, and never starts a dev server. Skip it with `--no-post`.
5. `git init`, a `.gitignore` for the language when there is none, and a first commit. Skip it with
   `--no-git`. If git has no identity configured, the first commit uses your Foundry author name
   with the address `foundry@localhost`.
6. If VS Code was detected and you are in a terminal, the project opens there (`--no-open` skips it).

## Configuration

The file lives at `~/.foundry/config.yaml` (override with `--config` or see `foundry config --path`).
It is plain YAML and written atomically. It holds your author name, license, detected tools, saved
templates (name, path, language, description, file list) and the default template per language.

## Development

```console
make check      # gofmt, go vet, go test
make race       # tests with the race detector
make smoke      # end-to-end test with the real binary
make dist       # release archives, .deb packages and SHA256SUMS in dist/
```

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues: [SECURITY.md](SECURITY.md). Changes:
[CHANGELOG.md](CHANGELOG.md).

## License

Foundry is free software under the GNU Affero General Public License v3.0 or later. See
[LICENSE](LICENSE).
