#!/usr/bin/env bash
set -euo pipefail

REPO="groundsgg/grounds-cli"
LATEST=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
VERSION="${LATEST#v}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac

NAME="grounds_${VERSION}_${OS}_${ARCH}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${NAME}.tar.gz"

echo "→ Downloading $URL …"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
curl -sSL "$URL" | tar -xz -C "$TMP"

INSTALL_DIR=${INSTALL_DIR:-$HOME/.local/bin}
mkdir -p "$INSTALL_DIR"
mv "$TMP/grounds" "$INSTALL_DIR/grounds"
chmod +x "$INSTALL_DIR/grounds"

echo "✔ Installed to $INSTALL_DIR/grounds"
"$INSTALL_DIR/grounds" version
