#!/bin/sh
# ui-engine install.sh — curl -fsSL https://ui-engine.dev/install.sh | sh
set -e

REPO="ui-engine/ui-engine"
VERSION="${UI_ENGINE_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BIN="ui-engine"

echo "→ ui-engine installer (version: $VERSION)"

# detect OS/arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac
case "$OS" in
  darwin|linux) ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

# resolve version
if [ "$VERSION" = "latest" ]; then
  if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  fi
  if [ -z "$VERSION" ]; then
    VERSION="v0.1.0"
  fi
fi

FILE="${BIN}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE}"

echo "→ downloading ${URL}"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMP/$BIN"
elif command -v wget >/dev/null 2>&1; then
  wget -q "$URL" -O "$TMP/$BIN"
else
  echo "curl or wget required" >&2; exit 1
fi

chmod +x "$TMP/$BIN"

# install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BIN" "$INSTALL_DIR/$BIN"
else
  echo "→ need sudo for $INSTALL_DIR"
  sudo mv "$TMP/$BIN" "$INSTALL_DIR/$BIN"
fi

echo "✓ ui-engine ${VERSION} installed to ${INSTALL_DIR}/${BIN}"
echo "  ui-engine --help"
"${INSTALL_DIR}/${BIN}" --help | head -20
