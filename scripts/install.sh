#!/usr/bin/env sh
set -eu

REPO="${REPO:-neko233-com/dbbackup233}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux) GOOS="linux" ;;
  darwin) GOOS="darwin" ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  arm64|aarch64) GOARCH="arm64" ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac

ASSET="dbbackup233-${GOOS}-${GOARCH}"
API="https://api.github.com/repos/${REPO}/releases/latest"
TMP="$(mktemp)"

echo "Resolving latest release from $API"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$API" -o "$TMP.json"
  URL="$(grep -o "\"browser_download_url\": *\"[^\"]*${ASSET}\"" "$TMP.json" | head -n 1 | sed 's/.*"browser_download_url": *"//;s/"$//')"
  curl -fsSL "$URL" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMP.json" "$API"
  URL="$(grep -o "\"browser_download_url\": *\"[^\"]*${ASSET}\"" "$TMP.json" | head -n 1 | sed 's/.*"browser_download_url": *"//;s/"$//')"
  wget -qO "$TMP" "$URL"
else
  echo "curl or wget is required" >&2
  exit 1
fi

if [ -z "${URL:-}" ]; then
  echo "release asset not found: $ASSET" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP" "$INSTALL_DIR/dbbackup233"
rm -f "$TMP" "$TMP.json"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Add this to PATH if needed: export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac

"$INSTALL_DIR/dbbackup233" version
