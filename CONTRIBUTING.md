# Contributing

Thanks for helping. Foundry is small on purpose, so changes that keep it small and predictable are
the easiest to accept.

## Set up

```console
git clone https://github.com/SandSoftwareSolutions/Foundry
cd Foundry
make check
```

You need Go 1.25 or newer and git. `make lint` (staticcheck) and `make vuln` (govulncheck) run what CI runs.

## Before you open a pull request

- `make check` passes (gofmt, `go vet`, tests); `make lint` is clean.
- New behaviour has a test. Package tests live next to the code; command line behaviour is tested in
  `cmd/cli_test.go` with a throw-away config file, so tests never touch your real `~/.foundry`.
- User visible changes get a line in `CHANGELOG.md` under *Unreleased*.
- Keep commits focused and write the message in the imperative ("Add template vars command").

## Design rules

1. **Plan first, write second.** Anything that creates files is built as a plan (`internal/project`)
   so `--dry-run` and the real run cannot differ.
2. **No shell.** External commands are started directly with explicit arguments and a working
   directory. Never build a command line from user input.
3. **No surprises.** Nothing outside the target directory and the config file is written. Commands
   that read must not write, and `--help`/`version` must have no side effects.
4. **Fail clean.** An error leaves no half-made project behind and says what to do next.
5. **Testable.** Commands return errors; code takes its inputs as arguments.

## Reporting bugs

Open an issue with the command you ran, what you expected, what happened, `foundry version` and your
operating system. Never paste secrets from your config or templates.

## License

By contributing you agree that your contribution is licensed under the project's license
(AGPL-3.0-or-later).
