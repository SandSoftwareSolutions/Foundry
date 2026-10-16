#!/usr/bin/env bash
set -euo pipefail

# Rooktest voor Foundry CLI met meerdere templates en variabelen

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TEST_WORKDIR="$ROOT_DIR/.smoke"
PROJECT_BASE="smoke-sample"

cleanup() {
  echo "[cleanup] Removing test workdir: $TEST_WORKDIR"
  rm -rf "$TEST_WORKDIR" || true
}
trap cleanup EXIT

mkdir -p "$TEST_WORKDIR"

if [ -n "${FOUNDRY_BIN:-}" ]; then
  FOUNDRY="$FOUNDRY_BIN"
else
  echo "[build] Building Foundry"
  pushd "$ROOT_DIR" >/dev/null
  go build -o "$TEST_WORKDIR/foundry" ./cmd/foundry
  popd >/dev/null
  FOUNDRY="$TEST_WORKDIR/foundry"
fi

# Never touch the real configuration: use a throw-away config file for the whole run.
export FOUNDRY_CONFIG_DIR="$TEST_WORKDIR/config"
mkdir -p "$FOUNDRY_CONFIG_DIR"
foundry() { "$FOUNDRY" --config "$FOUNDRY_CONFIG_DIR/config.yaml" --no-color "$@"; }

# --- Template definities ---
declare -A TEMPLATES
TEMPLATES["sample-go"]="package main

import \"fmt\"

func main() {
    fmt.Println(\"{{PROJECT_NAME}} by {{AUTHOR}}!\")
}
"
TEMPLATES["sample-python"]="# Sample Python file
print(\"{{PROJECT_NAME}} by {{AUTHOR}}!\")
"

# Add README to test placeholders
README_CONTENT="# {{PROJECT_NAME}}
Author: {{AUTHOR}}
lower: {{PROJECT_NAME_LOWER}}
upper: {{PROJECT_NAME_UPPER}}
"

# Loop over templates
for template_name in "${!TEMPLATES[@]}"; do
    TEMPLATE_DIR="$TEST_WORKDIR/$template_name"
    mkdir -p "$TEMPLATE_DIR"

    echo "[prepare] Creating sample template $template_name at $TEMPLATE_DIR"
    echo "${TEMPLATES[$template_name]}" > "$TEMPLATE_DIR/main.${template_name##*-}"
    echo "$README_CONTENT" > "$TEMPLATE_DIR/README.md"
    echo -e "bin/\n*.tmp" > "$TEMPLATE_DIR/.foundryignore"
    mkdir -p "$TEMPLATE_DIR/bin"
    echo "should-not-copy" > "$TEMPLATE_DIR/bin/ignored.txt"

    echo "[template add] Adding template $template_name"
    foundry template add "$template_name" "$TEMPLATE_DIR" --description "Smoke Sample $template_name" --language "${template_name##*-}"

    echo "[template list] Listing templates"
    foundry template list --quiet | grep -q "$template_name"
done

# --- Test scaffolding met variabelen ---
VAR_SETS=(
    "AUTHOR=SmokeTester"
    "AUTHOR=TestUser"
)

TARGET_DIR="$TEST_WORKDIR/target"
mkdir -p "$TARGET_DIR"

for template_name in "${!TEMPLATES[@]}"; do
    for var in "${VAR_SETS[@]}"; do
        PROJECT_NAME="${template_name}-$(echo $var | cut -d= -f2)"
        echo "[new] Creating new project $PROJECT_NAME from template $template_name"
        foundry new "$PROJECT_NAME" --template "$template_name" --path "$TARGET_DIR" --non-interactive --no-git --no-post --var "$var"

        PROJECT_DIR="$TARGET_DIR/$PROJECT_NAME"

        echo "[assert] Project directory exists"
        [ -d "$PROJECT_DIR" ]

        echo "[assert] Placeholders replaced"
        grep -q "${PROJECT_NAME} by $(echo $var | cut -d= -f2)" "$PROJECT_DIR/main.${template_name##*-}"
        grep -q "# ${PROJECT_NAME}" "$PROJECT_DIR/README.md"
        grep -q "lower: ${PROJECT_NAME,,}" "$PROJECT_DIR/README.md" || grep -q "lower: $(echo "$PROJECT_NAME" | tr '[:upper:]' '[:lower:]')" "$PROJECT_DIR/README.md"
        grep -q "upper: ${PROJECT_NAME^^}" "$PROJECT_DIR/README.md" || grep -q "upper: $(echo "$PROJECT_NAME" | tr '[:lower:]' '[:upper:]')" "$PROJECT_DIR/README.md"

        echo "[assert] Ignore patterns respected"
        [ ! -d "$PROJECT_DIR/bin" ]
    done
done

# Optioneel: Go build testen
if command -v go >/dev/null 2>&1; then
    for d in "$TARGET_DIR"/*; do
        if [ -f "$d/main.go" ]; then
            echo "[optional] go build $d"
            pushd "$d" >/dev/null
            go mod init "$(basename $d)" >/dev/null 2>&1 || true
            go mod tidy >/dev/null 2>&1 || true
            go build ./...
            popd >/dev/null
        fi
    done
fi

echo "[success] Multi-template smoke test completed"
