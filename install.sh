#!/bin/bash
#
# Flux CLI Installer
#
# Run from the repo directory:
#   ./install.sh
#
# To update:
#   git pull && ./install.sh
#

set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Run: brew install go"
    exit 1
fi

# Ensure we're in the repo directory
if [ ! -f "go.mod" ]; then
    echo "Error: Run this script from the flux-cli repo directory"
    exit 1
fi

echo "Building Flux CLI..."
go build -o flux .

echo "Installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
mv flux "$INSTALL_DIR/flux"

# Check PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Add to your ~/.zshrc or ~/.bashrc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "Done! Run 'flux --help' to get started."
