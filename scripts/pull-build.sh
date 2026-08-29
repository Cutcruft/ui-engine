#!/bin/sh
# pull-build.sh — тянуть готовый билд с GitHub Releases
# Использование: ./scripts/pull-build.sh [version] [--to bin/]
# По умолчанию — latest
set -e

REPO="${REPO:-ui-engine/ui-engine}"
VERSION="${1:-latest}"
if [ "$1" = "--to" ]; then
  VERSION="latest"
  DEST="$2"
else
  if [ "$2" = "--to" ]; then
    DEST="$3"
  else
    DEST="bin"
  fi
  # если первый arg — версия с v
  case "$VERSION" in
    v*|latest) ;;
    *) VERSION="latest"; DEST="$1" ;;
  esac
fi

# если VERSION — latest, резолвим тег
if [ "$VERSION" = "latest" ]; then
  if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  fi
  if [ -z "$VERSION" ]; then
    echo "не удалось определить latest, используйте v0.1.0" >&2
    VERSION="v0.1.0"
  fi
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in x86_64|amd64) ARCH="amd64" ;; arm64|aarch64) ARCH="arm64" ;; *) echo "unsupported arch $ARCH" >&2; exit 1 ;; esac

FILE="ui-engine-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE}"

echo "→ pull-build ${VERSION} для ${OS}/${ARCH}"
echo "  ${URL} → ${DEST}/"

mkdir -p "$DEST"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$DEST/ui-engine" || {
    echo "не удалось скачать $URL" >&2
    echo "проверьте Releases: https://github.com/${REPO}/releases" >&2
    exit 1
  }
elif command -v wget >/dev/null 2>&1; then
  wget -q "$URL" -O "$DEST/ui-engine"
else
  echo "нужен curl или wget" >&2; exit 1
fi

chmod +x "$DEST/ui-engine"
echo "✓ $DEST/ui-engine (${VERSION})"

# также пробуем скачать wasm и модули
for extra in "core.wasm" "button.uimod" "layout.uimod" "richtext.uimod"; do
  EURL="https://github.com/${REPO}/releases/download/${VERSION}/${extra}"
  if curl -fsSL -o "$DEST/$extra" "$EURL" 2>/dev/null; then
    echo "✓ $extra"
  fi
done

echo "ok — $DEST/ui-engine --help"
