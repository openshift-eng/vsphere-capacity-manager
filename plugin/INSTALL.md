# Installation Guide for oc-vcm Plugin

Quick reference for installing the oc-vcm plugin and its dependencies.

## Prerequisites

- Python 3.6 or higher
- OpenShift CLI (`oc`)
- Access to a cluster with vSphere Capacity Manager

## Quick Install (Automated)

```bash
cd plugin
./install.sh
```

The script will:
1. Check Python version
2. Install Rich library (if pip is available)
3. Copy the plugin to `~/.local/bin`
4. Make it executable
5. Verify the installation

## Manual Installation

### Step 1: Install Rich Library

Choose the method that works for your system:

#### If you have pip or pip3:
```bash
pip3 install --user rich
# or
pip install --user rich
```

#### If pip is not available:

**Fedora/RHEL/CentOS:**
```bash
sudo dnf install python3-rich
```

**Debian/Ubuntu:**
```bash
sudo apt install python3-rich
```

**Arch Linux:**
```bash
sudo pacman -S python-rich
```

**macOS (using Homebrew):**
```bash
brew install python-rich
```

**Using Python's built-in installer:**
```bash
python3 -m ensurepip --user
python3 -m pip install --user rich
```

**Using uv (modern, fast package installer):**
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
uv pip install rich
```

**From source (always works):**
```bash
git clone https://github.com/Textualize/rich.git
cd rich
python3 setup.py install --user
```

### Step 2: Install the Plugin

```bash
# Copy to a directory in your PATH
cp oc-vcm ~/.local/bin/

# Make it executable
chmod +x ~/.local/bin/oc-vcm

# Verify installation
oc vcm --help
```

### Step 3: Add to PATH (if needed)

If you get "command not found", add `~/.local/bin` to your PATH:

```bash
# For bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# For zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Verify Installation

### Check Python version:
```bash
python3 --version
# Should be 3.6 or higher
```

### Check if Rich is installed:
```bash
python3 -c "import rich; print('Rich version:', rich.__version__)"
# Should print version number (e.g., "Rich version: 13.7.0")
```

### Check if plugin is accessible:
```bash
which oc-vcm
# Should print path like: /home/user/.local/bin/oc-vcm

oc vcm --help
# Should show help text
```

### Test the status command:
```bash
oc vcm status
# Should display pool status (rich format if library is installed)
```

## Installation Verification Checklist

- [ ] Python 3.6+ is installed (`python3 --version`)
- [ ] Rich library is installed (`python3 -c "import rich"`)
- [ ] Plugin is executable (`ls -l ~/.local/bin/oc-vcm`)
- [ ] Plugin is in PATH (`which oc-vcm`)
- [ ] Plugin runs without errors (`oc vcm --help`)
- [ ] OpenShift cluster is accessible (`oc whoami`)

## Common Issues

### "pip: command not found"

**Solution:** Install pip or use alternative methods:
```bash
# Install pip
sudo dnf install python3-pip  # Fedora/RHEL
sudo apt install python3-pip  # Debian/Ubuntu

# OR use system package manager
sudo dnf install python3-rich  # Fedora/RHEL
sudo apt install python3-rich  # Debian/Ubuntu

# OR use Python's ensurepip
python3 -m ensurepip --user
```

### "Permission denied" when running oc-vcm

**Solution:** Make the file executable:
```bash
chmod +x ~/.local/bin/oc-vcm
```

### "Warning: Rich library not installed"

**Solution:** The plugin still works! It just uses plain text output. To enable rich output, install Rich using any of the methods above.

### "oc vcm: command not found"

**Solutions:**
1. Check if plugin is installed: `ls -l ~/.local/bin/oc-vcm`
2. Add to PATH: `export PATH="$HOME/.local/bin:$PATH"`
3. Use full path: `~/.local/bin/oc-vcm status`

### Plugin installed but Rich features don't work

**Check Rich installation:**
```bash
python3 -c "import rich; print('OK')"
```

If this fails, Rich is not properly installed. Try reinstalling:
```bash
pip3 install --user --force-reinstall rich
```

## Alternative Installation Locations

You can install the plugin anywhere in your PATH:

```bash
# System-wide (requires root)
sudo cp oc-vcm /usr/local/bin/

# User bin directory
cp oc-vcm ~/bin/

# Custom location
mkdir -p ~/my-tools
cp oc-vcm ~/my-tools/
export PATH="$HOME/my-tools:$PATH"
```

## Uninstallation

To remove the plugin:

```bash
# Remove the plugin
rm ~/.local/bin/oc-vcm

# Optionally remove Rich library
pip3 uninstall rich
# or
sudo dnf remove python3-rich  # Fedora/RHEL
sudo apt remove python3-rich  # Debian/Ubuntu
```

## Container/Podman Installation

If you prefer to run in a container:

```bash
# Create a Dockerfile
cat > Dockerfile <<EOF
FROM python:3.11-slim
RUN pip install rich
COPY oc-vcm /usr/local/bin/
RUN chmod +x /usr/local/bin/oc-vcm
ENTRYPOINT ["oc-vcm"]
EOF

# Build
podman build -t oc-vcm .

# Run
podman run --rm -v ~/.kube:/root/.kube:ro oc-vcm status
```

## Next Steps

After installation, try these commands:

```bash
# View status with rich output
oc vcm status

# View in JSON format
oc vcm status --format json

# Sort by capacity
oc vcm status --sort capacity

# Include excluded pools
oc vcm status --include-excluded

# View all available commands
oc vcm --help
```

For more information, see [README.md](README.md).
