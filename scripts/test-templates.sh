#!/usr/bin/env bash
# Creates a project from every built-in template with the real foundry binary and runs that language's
# own build or tests on it. A language whose tool is not installed is skipped (and said so).
#
#   FOUNDRY_BIN=./foundry scripts/test-templates.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FOUNDRY="${FOUNDRY_BIN:-$ROOT/foundry}"
[ -x "$FOUNDRY" ] || { echo "build foundry first (make build) or set FOUNDRY_BIN" >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
export HOME="$WORK/home"; mkdir -p "$HOME"
CFG="$WORK/config.yaml"
failed=0; skipped=0; passed=0

have() {
  command -v "$1" >/dev/null 2>&1 || return 1
  # rustup installs a "cargo" that does nothing until a toolchain is chosen
  [ "$1" != cargo ] || cargo --version >/dev/null 2>&1
}

check() { # template-name  tool  command...
  local name="$1" tool="$2"; shift 2
  local dir="$WORK/$name-project"
  "$FOUNDRY" --config "$CFG" --no-color new "demo-$name" --template "$name" --path "$WORK" \
    --no-git --no-post --non-interactive >/dev/null
  mv "$WORK/demo-$name" "$dir"
  if ! have "$tool"; then
    echo "[skip] $name ($tool not installed)"; skipped=$((skipped+1)); return
  fi
  if (cd "$dir" && "$@" >"$WORK/$name.log" 2>&1); then
    echo "[ ok ] $name"; passed=$((passed+1))
  else
    echo "[FAIL] $name"; sed 's/^/        /' "$WORK/$name.log" | tail -25; failed=$((failed+1))
  fi
}

check go-basic         go     bash -c 'go vet ./... && go test ./... && go run . -name Ada | grep -q "Hello, Ada"'
check go-http          go     bash -c 'go vet ./... && go test ./...'
check node-basic       node   bash -c 'node --test && node src/index.js Ada | grep -q "Hello, Ada"'
check node-http        node   node --test
check typescript-basic npm    bash -c 'npm install --no-audit --no-fund --silent && npm test'
check python-basic     python3 bash -c 'python3 -m unittest discover -s tests && PYTHONPATH=src python3 -m demo_python_basic --name Ada | grep -q "Hello, Ada"'
check python-http      python3 python3 -m unittest discover -s tests
check rust-basic       cargo  bash -c 'cargo test --quiet && cargo run --quiet -- Ada | grep -q "Hello, Ada"'
check static-site      python3 python3 -c "import html.parser,sys; p=html.parser.HTMLParser(); p.feed(open('index.html').read()); print('ok')"
check blank            true   test -f README.md

echo
echo "passed: $passed, skipped: $skipped, failed: $failed"
[ "$failed" -eq 0 ]
