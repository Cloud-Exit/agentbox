#!/bin/sh
# ExitBox Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/cloud-exit/exitbox/main/scripts/install.sh | sh
set -e

REPO="cloud-exit/exitbox"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

BINARY="exitbox-${OS}-${ARCH}"

# Get latest release tag
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
    grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "ERROR: Could not determine latest release" >&2
    exit 1
fi

echo "Installing exitbox ${LATEST} (${OS}/${ARCH})..."

# Download binary and checksums
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${LATEST}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL -o "${TMPDIR}/exitbox" "$DOWNLOAD_URL"
curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL"

# Verify checksum
EXPECTED=$(grep "$BINARY" "${TMPDIR}/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
    echo "ERROR: No checksum found for $BINARY" >&2
    exit 1
fi

ACTUAL=$(sha256sum "${TMPDIR}/exitbox" 2>/dev/null | awk '{print $1}' || \
         shasum -a 256 "${TMPDIR}/exitbox" 2>/dev/null | awk '{print $1}')

if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "ERROR: Checksum verification failed!" >&2
    echo "  Expected: $EXPECTED" >&2
    echo "  Actual:   $ACTUAL" >&2
    exit 1
fi

echo "Checksum verified."

# Install
mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/exitbox" "${INSTALL_DIR}/exitbox"
chmod +x "${INSTALL_DIR}/exitbox"

echo "Installed exitbox to ${INSTALL_DIR}/exitbox"

# Check PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo ""
        echo "Add this to your shell config:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        ;;
esac

echo ""
echo "Get started:"
echo "  exitbox setup"
echo ""
