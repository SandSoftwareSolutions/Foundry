#!/usr/bin/env bash
# Installs the foundry binary that sits next to this script (it is part of the release archive).
#   ./install.sh                       asks where to install
#   ./install.sh --prefix ~/.local/bin non-interactive
set -euo pipefail

BIN_NAME="foundry"
DEFAULT_DIR="$HOME/.local/bin"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR=""

while [ $# -gt 0 ]; do
  case "$1" in
    --prefix) INSTALL_DIR="${2:?--prefix needs a directory}"; shift 2 ;;
    -h|--help) sed -n '2,5p' "$0"; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [ ! -f "$HERE/$BIN_NAME" ]; then
  echo "Cannot find '$BIN_NAME' next to the installer ($HERE)." >&2
  exit 1
fi

if [ -z "$INSTALL_DIR" ]; then
  read -rp "Install directory [$DEFAULT_DIR]: " INSTALL_DIR
  INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_DIR}"
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "$HERE/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
echo "Installed $BIN_NAME to $INSTALL_DIR"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) echo "$INSTALL_DIR is already on your PATH." ;;
  *) echo "Add it to your PATH, for example:  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
echo "Try: $BIN_NAME --help"
