#!/bin/bash
#
# Flux CLI Installer
#
# Install latest release (no Go required):
#   curl -fsSL https://raw.githubusercontent.com/eyra/flux-cli/master/install.sh | bash
#
# Install from source (requires Go):
#   git clone https://github.com/eyra/flux-cli && cd flux-cli && ./install.sh
#
# The Claude Code skill is installed with `flux install claude`, into
# ~/.claude/skills/flux (or $CLAUDE_CONFIG_DIR/skills/flux).
#
# Overrides:
#   INSTALL_DIR   where the binary goes (default: ~/.local/bin)
#   FLUX_ARCHIVE  install this local release archive instead of downloading one
#                 (used by scripts/test-install.sh)
#

set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
REPO="eyra/flux-cli"

# install_skill: install the skill embedded in the binary. A failure (such as a
# symlinked skill folder) leaves the installed binary in place.
install_skill() {
  "$INSTALL_DIR/flux" install claude ||
    echo "Warning: the Claude Code skill was not installed; see the error above"
}

check_path() {
  SHELL_RC="$HOME/.zshrc"
  [ -f "$HOME/.bashrc" ] && [ ! -f "$HOME/.zshrc" ] && SHELL_RC="$HOME/.bashrc"

  if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null && ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add to your $SHELL_RC:"
    echo ""
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
    echo "Then run: source $SHELL_RC"
  fi
}

install_from_release() {
  TMP=$(mktemp -d)

  if [ -n "$FLUX_ARCHIVE" ]; then
    echo "Installing from local archive $FLUX_ARCHIVE..."
    LATEST="$(basename "$FLUX_ARCHIVE")"
    tar -xzf "$FLUX_ARCHIVE" -C "$TMP"
  else
    download_release
  fi

  mkdir -p "$INSTALL_DIR"
  mv "$TMP/flux" "$INSTALL_DIR/flux"
  chmod +x "$INSTALL_DIR/flux"

  install_skill

  rm -rf "$TMP"
  echo "Installed $LATEST to $INSTALL_DIR/flux"
}

# download_release: fetch the latest release archive and extract it into $TMP.
download_release() {
  echo "Downloading latest Flux CLI release..."

  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)
  [ "$ARCH" = "x86_64" ] && ARCH="amd64"
  [ "$ARCH" = "aarch64" ] && ARCH="arm64"

  LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
  if [ -z "$LATEST" ]; then
    echo "Error: could not fetch latest release from GitHub"
    exit 1
  fi

  ARCHIVE="flux_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/$REPO/releases/download/$LATEST/$ARCHIVE"

  echo "Downloading $LATEST ($OS/$ARCH)..."
  curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
  tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
}

install_from_source() {
  echo "Building Flux CLI from source..."

  if [ ! -f "go.mod" ]; then
    echo "Error: Run this script from the flux-cli repo directory, or omit Go to download a release binary"
    exit 1
  fi

  go build -o flux .
  mkdir -p "$INSTALL_DIR"
  mv flux "$INSTALL_DIR/flux"
  echo "Installed to $INSTALL_DIR/flux"

  install_skill
}

# Build from source if in the repo and Go is available; otherwise download a release
if [ -z "$FLUX_ARCHIVE" ] && [ -f "go.mod" ] && command -v go &> /dev/null; then
  install_from_source
else
  install_from_release
fi

check_path

echo ""
echo "Done! Run 'flux --help' to get started."
