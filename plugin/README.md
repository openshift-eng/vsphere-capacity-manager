# oc-vcm Plugin

OpenShift CLI plugin for managing vSphere Capacity Manager resources.

## Installation

### Install Python Dependencies

The plugin requires Python 3.6+ and the Rich library for enhanced output formatting.

#### Option 1: Using pip (Recommended)

```bash
cd plugin
pip install -r requirements.txt
# or
pip install rich
```

#### Option 2: Using pip3 (if pip points to Python 2)

```bash
pip3 install rich
# or with --user flag if you don't have root access
pip3 install --user rich
```

#### Option 3: Using System Package Manager

If pip is not available, you can install Rich using your system's package manager:

**Fedora/RHEL/CentOS:**
```bash
sudo dnf install python3-rich
# or
sudo yum install python3-rich
```

**Debian/Ubuntu:**
```bash
sudo apt install python3-rich
```

**Arch Linux:**
```bash
sudo pacman -S python-rich
```

#### Option 4: Using pipx (Isolated Installation)

```bash
# Install pipx first if needed
python3 -m pip install --user pipx
python3 -m pipx ensurepath

# Install rich in an isolated environment
pipx install rich
```

#### Option 5: Using uv (Fast Python Package Manager)

```bash
# Install uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Install rich
uv pip install rich
```

#### Option 6: Using Virtual Environment

```bash
# Create virtual environment
python3 -m venv ~/.venv/oc-vcm
source ~/.venv/oc-vcm/bin/activate

# Install rich
pip install rich

# The plugin will use this when the venv is activated
```

#### Option 7: Install from Source (No Package Manager Required)

```bash
# Clone the Rich repository
git clone https://github.com/Textualize/rich.git
cd rich

# Install
python3 setup.py install --user
```

**Note:** The plugin will work without Rich, but will fall back to plain text output.

### Install the Plugin

Copy the `oc-vcm` executable to a directory in your PATH:

```bash
cp oc-vcm ~/.local/bin/
# or
cp oc-vcm /usr/local/bin/
```

Make sure it's executable:

```bash
chmod +x ~/.local/bin/oc-vcm
```

## Usage

### Status Command

Display the status of vSphere Capacity Manager pools and leases with rich, color-coded output.

```bash
oc vcm status [OPTIONS]
```

#### Options

- `--include-excluded` - Include pools marked as 'excluded' in the status output
- `--format {plain|rich|json}` - Output format (default: rich if available, otherwise plain)
  - `plain` - Simple ASCII table output (original format)
  - `rich` - Enhanced colored table with health indicators (requires Rich library)
  - `json` - Machine-readable JSON output
- `--sort {name|capacity|leases}` - Sort pools by name, capacity, or lease count (default: name)

#### Examples

**Default output with Rich formatting:**
```bash
oc vcm status
```

**Include excluded pools:**
```bash
oc vcm status --include-excluded
```

**Sort by capacity (lowest first):**
```bash
oc vcm status --sort capacity
```

**Sort by lease count (highest first):**
```bash
oc vcm status --sort leases
```

**JSON output for scripting:**
```bash
oc vcm status --format json
```

**Plain text output:**
```bash
oc vcm status --format plain
```

#### Rich Output Features

When using `--format rich`, the status command displays:

- **Summary Banner**: Quick overview of total pools, active pools, cordoned/excluded pools, and lease statistics
- **Health Indicators**: Color-coded dots showing pool health
  - 🟢 Green: ≥50% capacity available (healthy)
  - 🟡 Yellow: 25-49% capacity available (degraded)
  - 🔴 Red: <25% capacity available (critical)
- **Color-Coded Metrics**: CPU, Memory, and Network percentages with colors based on availability
- **Visual Symbols**: ✓ and ✗ for boolean values (cordoned/excluded)
- **Lease Statistics**: Detailed breakdown of single-tenant and multi-tenant leases

### Other Commands

#### Pool Management

**Cordon a pool** (prevent new leases):
```bash
oc vcm cordon --pool <pool-name>
```

**Uncordon a pool**:
```bash
oc vcm uncordon --pool <pool-name>
```

**Exclude a pool** (remove from scheduling):
```bash
oc vcm exclude --pool <pool-name>
```

**Include a pool**:
```bash
oc vcm include --pool <pool-name>
```

**Set pool capacity**:
```bash
oc vcm set-capacity --pool <pool-name> --cpu <vcpus> --memory <memory-mb>
```

#### VLAN Management

**Add VLAN to pool(s)**:
```bash
# Add to all uncordoned, unexcluded pools
oc vcm add-vlan --vlan <vlan-id>

# Add to specific pool
oc vcm add-vlan --vlan <vlan-id> --pool <pool-name>
```

**Drop VLAN from pool(s)**:
```bash
# Drop from all pools
oc vcm drop-vlan --vlan <vlan-id>

# Drop from specific pool
oc vcm drop-vlan --vlan <vlan-id> --pool <pool-name>
```

#### Network Management

**List networks**:
```bash
# List all networks
oc vcm networks

# Filter by network type
oc vcm networks --networkType single-tenant
oc vcm networks --networkType multi-tenant
```

**Split a network**:
```bash
oc vcm split-network --network <network-name> --subnets <count>
```

#### Lease Information

**List jobs with leases**:
```bash
oc vcm jobs
```

**List all leases**:
```bash
oc vcm leases
```

## Color Scheme

The Rich output uses the following color scheme:

- **Green**: Healthy status, ≥50% capacity
- **Yellow**: Warning status, 25-49% capacity, cordoned pools
- **Red**: Critical status, <25% capacity, excluded pools
- **Cyan**: Informational text, pool names
- **Dim**: Disabled or false values

## Requirements

- Python 3.6+
- OpenShift CLI (`oc`)
- Rich library (optional, for enhanced output)
- Access to a cluster with vSphere Capacity Manager installed

## Troubleshooting

### Rich library not found

If you see this warning:
```
Warning: Rich library not installed. Using plain output.
Install with: pip install rich
```

The plugin will still work but will use plain text output. To enable the enhanced Rich output, install the Rich library using one of the methods above.

**Quick fixes:**
```bash
# Try pip3
pip3 install --user rich

# Try system package manager (Fedora/RHEL)
sudo dnf install python3-rich

# Try system package manager (Debian/Ubuntu)
sudo apt install python3-rich
```

### pip/pip3 command not found

If neither `pip` nor `pip3` is available on your system, you have several options:

**Option A: Install pip itself**
```bash
# Fedora/RHEL/CentOS
sudo dnf install python3-pip

# Debian/Ubuntu
sudo apt install python3-pip

# Arch Linux
sudo pacman -S python-pip
```

**Option B: Use system package manager** (see installation options above)

**Option C: Use Python's ensurepip module**
```bash
python3 -m ensurepip --user
python3 -m pip install --user rich
```

**Option D: Download wheel file manually**
1. Visit https://pypi.org/project/rich/#files
2. Download the `.whl` file for your platform
3. Install with: `python3 -m pip install --user /path/to/rich-*.whl`

### Permission denied

Make sure the `oc-vcm` file is executable:
```bash
chmod +x oc-vcm
# or if installed
chmod +x ~/.local/bin/oc-vcm
```

### Command not found

Ensure the plugin is in your PATH and the filename starts with `oc-vcm`.

**Check if it's installed:**
```bash
which oc-vcm
```

**Add to PATH if needed:**
```bash
# Add this to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"

# Reload your shell
source ~/.bashrc  # or source ~/.zshrc
```

### Python version too old

The plugin requires Python 3.6 or higher. Check your version:
```bash
python3 --version
```

If your Python is too old, you may need to upgrade or use a newer Python installation:
```bash
# Fedora/RHEL
sudo dnf install python3.11

# Debian/Ubuntu
sudo apt install python3.11
```

### Plugin works but output is not colored

This can happen if:
1. Rich is not installed (install it using the methods above)
2. Your terminal doesn't support colors
3. Output is being piped to another command

**Force Rich output:**
```bash
oc vcm status --format rich
```

**Check if Rich is installed:**
```bash
python3 -c "import rich; print(rich.__version__)"
```

If this command prints a version number, Rich is installed correctly.
