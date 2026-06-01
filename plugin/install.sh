#!/bin/bash
# Installation script for oc-vcm plugin

set -e

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_NAME="oc-vcm"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

echo "Installing oc-vcm plugin..."

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed."
    exit 1
fi

# Check Python version (need 3.7+ for Rich library)
PYTHON_VERSION=$(python3 -c 'import sys; print(".".join(map(str, sys.version_info[:2])))')
REQUIRED_VERSION="3.7"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$PYTHON_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "Error: Python $REQUIRED_VERSION or higher is required (found $PYTHON_VERSION)"
    exit 1
fi

echo "✓ Python $PYTHON_VERSION found"

# Install Rich library
echo "Installing Python dependencies..."
RICH_INSTALLED=false

# Try pip3 first
if command -v pip3 &> /dev/null; then
    if pip3 install --user rich &> /dev/null; then
        echo "✓ Rich library installed via pip3"
        RICH_INSTALLED=true
    fi
fi

# Try pip if pip3 failed
if [ "$RICH_INSTALLED" = false ] && command -v pip &> /dev/null; then
    if pip install --user rich &> /dev/null; then
        echo "✓ Rich library installed via pip"
        RICH_INSTALLED=true
    fi
fi

# If pip methods failed, provide alternative installation instructions
if [ "$RICH_INSTALLED" = false ]; then
    echo "⚠ Warning: Could not install Rich library automatically."
    echo ""
    echo "Please install Rich manually using one of these methods:"
    echo ""
    echo "Method 1 - Using pip3:"
    echo "  pip3 install --user rich"
    echo ""
    echo "Method 2 - Using system package manager:"

    # Detect OS and provide appropriate command
    if command -v dnf &> /dev/null; then
        echo "  sudo dnf install python3-rich"
    elif command -v yum &> /dev/null; then
        echo "  sudo yum install python3-rich"
    elif command -v apt &> /dev/null; then
        echo "  sudo apt install python3-rich"
    elif command -v pacman &> /dev/null; then
        echo "  sudo pacman -S python-rich"
    else
        echo "  (Use your system's package manager to install python3-rich)"
    fi

    echo ""
    echo "Method 3 - Using uv (fast package manager):"
    echo "  curl -LsSf https://astral.sh/uv/install.sh | sh"
    echo "  uv pip install rich"
    echo ""
    echo "The plugin will work without Rich but will use plain text output."
    echo ""
fi

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Copy and make executable
cp "$PLUGIN_DIR/$PLUGIN_NAME" "$INSTALL_DIR/$PLUGIN_NAME"
chmod +x "$INSTALL_DIR/$PLUGIN_NAME"

echo "✓ Plugin installed to $INSTALL_DIR/$PLUGIN_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "⚠ Warning: $INSTALL_DIR is not in your PATH"
    echo "Add it to your PATH by adding this line to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

# Test the installation
echo ""
echo "Testing installation..."
if "$INSTALL_DIR/$PLUGIN_NAME" --help &> /dev/null; then
    echo "✓ Installation successful!"
    echo ""
    echo "Usage:"
    echo "  oc vcm status                    # Rich formatted output (default)"
    echo "  oc vcm status --format plain     # Plain text output"
    echo "  oc vcm status --format json      # JSON output"
    echo "  oc vcm status --sort capacity    # Sort by capacity"
    echo "  oc vcm status --help             # Show all options"
else
    echo "✗ Installation test failed"
    exit 1
fi
