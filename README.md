# Foundry

**Start every project from your own template, in one command.**

Foundry turns any directory into a reusable project template. Create a new project from it with the
name, author and license filled in, setup commands run and a first git commit made.

```console
$ foundry new my-api --language Go
Creating project "my-api" from go-basic (built-in)
✓ Created 5 files in my-api
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

Pick the one that fits your system. All of them give you the same `foundry` command; check with `foundry version`.

**Debian, Ubuntu and derivatives (apt)**: add the signed repository once, then install and update like any other package.

```console
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://sandsoftwaresolutions.github.io/apt/key.asc | sudo tee /etc/apt/keyrings/sandsoftwaresolutions.asc >/dev/null
echo "deb [signed-by=/etc/apt/keyrings/sandsoftwaresolutions.asc] https://sandsoftwaresolutions.github.io/apt stable main" | sudo tee /etc/apt/sources.list.d/sandsoftwaresolutions.list
sudo apt update
sudo apt install foundry
```

Updates arrive with `sudo apt upgrade`. More about the repository: [SandSoftwareSolutions/apt](https://github.com/SandSoftwareSolutions/apt).

**Linux and macOS (install script)**: downloads the release for your system, checks its SHA-256
checksum and installs to `~/.local/bin`. It never uses sudo. Read it first if you like:
[install.sh](install.sh).

```console
curl -fsSL https://raw.githubusercontent.com/SandSoftwareSolutions/Foundry/main/install.sh | sh
```

**Windows**: download `foundry_<version>_windows_amd64.zip` (or `arm64`) from the
[releases page](https://github.com/SandSoftwareSolutions/Foundry/releases), unzip it and put `foundry.exe`
in a folder that is on your `PATH`. Check the download against `SHA256SUMS`.

**With Go** (Go 1.25 or newer):

```console
go install github.com/SandSoftwareSolutions/Foundry/cmd/foundry@latest
```

**Other downloads**: every release has archives for Linux, macOS and Windows (amd64 and arm64), `.deb`
packages and `SHA256SUMS`. Build provenance can be checked with `gh attestation verify <file> --repo SandSoftwareSolutions/Foundry`.

**From source**:

```console
git clone https://github.com/SandSoftwareSolutions/Foundry
cd Foundry
make build        # produces ./foundry
```

## Built-in templates

Foundry comes with templates for common starts, so `foundry new` works the moment it is installed. Each one
builds and passes its own tests, and uses only the language's standard tooling (no frameworks).

| Template | Language | What you get |
| --- | --- | --- |
| `go-basic` | Go | Command line program with flags and tests |
| `go-http` | Go | HTTP service with health check, graceful shutdown and tests |
| `node-basic` | JavaScript | Node.js program, ES modules, built-in test runner |
| `node-http` | JavaScript | Node.js HTTP service with health check and tests |
| `typescript-basic` | TypeScript | TypeScript program compiled with `tsc`, built-in test runner |
| `python-basic` | Python | Command line program with `pyproject.toml` and `unittest` |
| `python-http` | Python | HTTP service with health check and `unittest`, standard library only |
| `rust-basic` | Rust | Command line program with a unit test |
| `static-site` | HTML | Plain HTML and CSS site |
| `blank` | any | An empty project: README and editor settings |

`foundry new my-app --language Go` uses the built-in template of that language (marked as default in
`foundry template list`) unless you chose another. Want to change one? `foundry template export go-basic ./my-go`
copies it to a folder; edit it and save it as your own with `foundry template add`. Your own templates win when a
name is the same.

## Quick start

```console
# 1. Create a project from a built-in template
foundry new my-api --language Go                  # the default template of a language
foundry new my-tool --template node-basic         # a specific template
foundry new my-app                                # choose from menus
foundry template list                             # see what is available

# 2. Preview first: nothing is written
foundry new my-api --template go-http --dry-run

# 3. Make your own templates and defaults
foundry template add my-go ./path/to/directory    # save any directory as a template
foundry config Go my-go                           # make it the default for Go

# 4. Other sources
foundry new my-site --git https://github.com/you/template   # a repository as template
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
| `foundry template list` | List built-in and saved templates (`--saved`, `--builtin`, `--sort name\|language`, `--quiet`, `--json`) |
| `foundry template show <name>` | Show details and files (`--files-only`, `--summary`, `--json`) |
| `foundry template update [name...]` | Rescan templates so file lists stay current |
| `foundry template edit <name>` | Change description, language, path or name |
| `foundry template vars <name>` | List the placeholders a template uses |
| `foundry template export <name> <folder>` | Copy a built-in template to a folder so you can change it |
| `foundry template remove <name>` | Forget a template (`--force` if it is a default); files stay |
| `foundry config` | Show settings; set with `--user`, `--license`, `--default-language`, `--docker`, `--interactive`; `--json`, `--path` |
| `foundry config <language> <template>` | Set the default template of a language, built-in or your own (`""` clears it) |
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
