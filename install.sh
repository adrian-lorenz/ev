#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
#  ev installer
#  Usage: curl -fsSL https://git-wall.de/noa-x/ev/install.sh | bash
# ──────────────────────────────────────────────

REPO_BASE="https://git-wall.de/noa-x/ev"
BINARY="ev"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="v1.0.5"

# ── colours ──────────────────────────────────
if [ -t 1 ]; then
  GREEN="\033[0;32m"; YELLOW="\033[0;33m"; RED="\033[0;31m"; BOLD="\033[1m"; RESET="\033[0m"
else
  GREEN=""; YELLOW=""; RED=""; BOLD=""; RESET=""
fi

info()    { echo -e "${BOLD}${GREEN}✓${RESET} $*"; }
warn()    { echo -e "${YELLOW}!${RESET} $*"; }
fatal()   { echo -e "${RED}✗${RESET} $*" >&2; exit 1; }
heading() { echo -e "\n${BOLD}$*${RESET}"; }

# ── dependency check ─────────────────────────
for cmd in curl tar uname; do
  command -v "$cmd" >/dev/null 2>&1 || fatal "Required command not found: $cmd"
done

# ── detect OS ────────────────────────────────
heading "Detecting system…"

RAW_OS=$(uname -s)
case "$RAW_OS" in
  Linux)   GOOS="linux"   ;;
  Darwin)  GOOS="darwin"  ;;
  MINGW*|MSYS*|CYGWIN*) GOOS="windows" ;;
  *)       fatal "Unsupported OS: $RAW_OS" ;;
esac

# ── detect architecture ───────────────────────
RAW_ARCH=$(uname -m)
case "$RAW_ARCH" in
  x86_64)          GOARCH="amd64" ;;
  aarch64|arm64)   GOARCH="arm64" ;;
  *)               fatal "Unsupported architecture: $RAW_ARCH" ;;
esac

info "OS: $GOOS / Arch: $GOARCH"
info "Version: ${VERSION}"

# ── build download URL ────────────────────────
if [ "$GOOS" = "windows" ]; then
  ARCHIVE="${BINARY}_${GOOS}_${GOARCH}.zip"
else
  ARCHIVE="${BINARY}_${GOOS}_${GOARCH}.tar.gz"
fi

URL="${REPO_BASE}/releases/${VERSION}/assets/${ARCHIVE}"
CHECKSUM_URL="${REPO_BASE}/releases/${VERSION}/assets/checksums.txt"

# ── download ──────────────────────────────────
heading "Downloading ${ARCHIVE}…"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fSL --progress-bar "$URL" -o "${TMP_DIR}/${ARCHIVE}"
curl -fsSL "$CHECKSUM_URL" -o "${TMP_DIR}/checksums.txt"

info "Downloaded successfully"

# ── verify checksum ───────────────────────────
heading "Verifying checksum…"

cd "$TMP_DIR"

EXPECTED=$(grep "  ${ARCHIVE}$" checksums.txt | awk '{print $1}')
[ -z "$EXPECTED" ] && fatal "Could not find checksum for ${ARCHIVE} in checksums.txt"

if command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')
elif command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}')
else
  warn "No sha256sum or shasum found — skipping checksum verification"
  ACTUAL="$EXPECTED"
fi

[ "$EXPECTED" = "$ACTUAL" ] \
  || fatal "Checksum verification failed — the download may be corrupted or tampered with"

info "Checksum OK"

# ── extract ───────────────────────────────────
heading "Extracting…"

if [ "$GOOS" = "windows" ]; then
  unzip -q "${ARCHIVE}" "${BINARY}.exe"
  BINARY="${BINARY}.exe"
else
  tar -xzf "${ARCHIVE}" "$BINARY"
fi

chmod +x "$BINARY"

# ── install ───────────────────────────────────
heading "Installing to ${INSTALL_DIR}…"

if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
else
  warn "Need sudo to write to ${INSTALL_DIR}"
  sudo mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
fi

# ── done ──────────────────────────────────────
echo ""
echo -e "${BOLD}${GREEN}ev ${VERSION} installed!${RESET}"
echo ""
echo "  Location : ${INSTALL_DIR}/${BINARY}"
echo ""
echo "  Get started:"
echo "    ev init"
echo "    ev set MY_SECRET"
echo "    ev run <your-command>"
echo ""
echo "  Web UI:"
echo "    ev manage"
echo ""