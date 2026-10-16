# Security

## Reporting a vulnerability

Please report security problems privately, not in a public issue. Use GitHub's "Report a
vulnerability" button on the Security tab of this repository, or email info@sandsoftwaresolutions.nl.

Include what you found, how to reproduce it and the Foundry version (`foundry version`). You will get
an answer as soon as possible; please allow a reasonable time to fix the problem before you disclose it.

## What Foundry does and does not do

- It copies files from a template directory or a cloned repository into a new directory and replaces
  placeholders. **A template is trusted input**: only use templates and repositories you trust,
  especially with `--git`, because the language setup step runs `go mod tidy` or `npm install`, and
  those tools can run code from the project's dependencies. Use `--no-post` when you are unsure.
- It starts only `git`, `go`, `npm` and your editor, without a shell.
- It never sends data anywhere itself. `git clone` and the package managers use the network when you
  ask for them.
- It does not store secrets. The config file holds your name, license and template paths.
