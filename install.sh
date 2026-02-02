#!/bin/bash
#
# Flux CLI Installer
#
# Usage:
#   go install github.com/eyra/flux-cli@latest
#
# Or with this script:
#   ./install.sh
#

set -e

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Run: brew install go"
    exit 1
fi

echo "Installing Flux CLI..."
go install github.com/eyra/flux-cli@latest

echo "Done! Run 'flux --help' to get started."
echo ""
echo "Make sure $(go env GOPATH)/bin is in your PATH:"
echo "  export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
