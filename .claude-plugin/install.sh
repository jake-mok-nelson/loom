#!/bin/bash
# Installation script for Loom Claude Code plugin
# This script installs the loom binary using go install with a pinned version

set -e

VERSION="v1.0.0"
MODULE="github.com/jake-mok-nelson/loom"

echo "Installing Loom ${VERSION}..."

# Install using go install with version pinning
go install "${MODULE}@${VERSION}"

echo "Loom ${VERSION} installed successfully!"
echo "Binary location: $(go env GOPATH)/bin/loom"
