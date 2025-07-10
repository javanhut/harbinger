#!/usr/bin/env bash

# Harbinger installation script

set -e

echo "Installing Harbinger..."

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case $OS in
    Darwin)
        PLATFORM="darwin"
        ;;
    Linux)
        PLATFORM="linux"
        ;;
    MINGW* | MSYS* | CYGWIN*)
        PLATFORM="windows"
        ;;
    *)
        echo "Unsupported operating system: $OS"
        exit 1
        ;;
esac

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64 | aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Check if go is installed
if command -v go &> /dev/null; then
    echo "Go is installed, using go install..."
    go install github.com/javanhut/harbinger@latest
    echo "Harbinger installed successfully!"
    echo "Run 'harbinger --help' to get started"
else
    echo "Go is not installed. Please install Go from https://golang.org/dl/"
    echo "Alternatively, download pre-built binaries from:"
    echo "https://github.com/javanhut/harbinger/releases"
    exit 1
fi