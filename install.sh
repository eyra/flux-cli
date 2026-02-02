#!/bin/bash
#
# Flux CLI Installer
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/eyra/flux-cli/master/install.sh | bash
#
# Or locally:
#   ./install.sh
#

set -e

REPO="github.com/eyra/flux-cli"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
CLONE_DIR="${CLONE_DIR:-$HOME/.flux-cli}"

echo "Installing Flux CLI..."

# Ensure Go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Install it first:"
    echo "  brew install go"
    exit 1
fi

# Create install directory if needed
mkdir -p "$INSTALL_DIR"

# Clone or update repo
if [ -d "$CLONE_DIR" ]; then
    echo "Updating existing installation..."
    cd "$CLONE_DIR"
    git pull --quiet
else
    echo "Cloning repository..."
    git clone --quiet "https://$REPO.git" "$CLONE_DIR"
    cd "$CLONE_DIR"
fi

# Build
echo "Building..."
go build -o flux .

# Install
echo "Installing to $INSTALL_DIR..."
mv flux "$INSTALL_DIR/flux"

# Check if install dir is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Add this to your shell profile (~/.zshrc or ~/.bashrc):"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

echo "Done! Run 'flux --help' to get started."
