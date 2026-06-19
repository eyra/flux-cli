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

set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
REPO="eyra/flux-cli"

install_skill() {
  SKILL_DIR="$HOME/.claude/skills/flux"
  if [ -f "skill/SKILL.md" ]; then
    echo "Installing Claude Code skill to $SKILL_DIR..."
    mkdir -p "$SKILL_DIR"
    cp skill/SKILL.md "$SKILL_DIR/SKILL.md"
  fi
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
  TMP=$(mktemp -d)
  curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
  tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

  mkdir -p "$INSTALL_DIR"
  mv "$TMP/flux" "$INSTALL_DIR/flux"
  chmod +x "$INSTALL_DIR/flux"

  # Install skill if bundled in the archive
  if [ -f "$TMP/SKILL.md" ]; then
    SKILL_DIR="$HOME/.claude/skills/flux"
    echo "Installing Claude Code skill to $SKILL_DIR..."
    mkdir -p "$SKILL_DIR"
    cp "$TMP/SKILL.md" "$SKILL_DIR/SKILL.md"
  fi

  rm -rf "$TMP"
  echo "Installed $LATEST to $INSTALL_DIR/flux"
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
if [ -f "go.mod" ] && command -v go &> /dev/null; then
  install_from_source
else
  install_from_release
fi

check_path

echo ""
echo "Done! Run 'flux --help' to get started."
