```
    _                    _   ____             
   / \   __ _  ___ _ __ | |_| __ )  _____  __
  / _ \ / _` |/ _ \ '_ \| __|  _ \ / _ \ \/ /
 / ___ \ (_| |  __/ | | | |_| |_) | (_) >  < 
/_/   \_\__, |\___|_| |_|\__|____/ \___/_/\_\
        |___/                                
                 By Cloud Exit / https://cloud-exit.com
```

# AgentBox

**Multi-Agent Container Sandbox** by [Cloud Exit](https://cloud-exit.com)

Run AI coding assistants (Claude, Codex, OpenCode) in isolated containers.


## Features

### Security
- **Rootless Containers**: Runs without host root privileges using Podman's user namespaces
- **Minimal Base Image**: Built on Debian Slim with only essential packages
- **Squid Proxy Firewall**: Reliable, TLD-based domain allowlisting (replaces flaky IP filtering)
- **Network Allowlist**: Only explicitly permitted root domains (and their subdomains) are reachable
- **No Privilege Escalation**: `--security-opt=no-new-privileges:true` enforced
- **Capability Dropping**: `--cap-drop=ALL` removes all Linux capabilities
- **Resource Limits**: Default 8GB RAM / 4 CPUs to prevent DoS
- **Secure Defaults**: No automatic mounting of sensitive SSH keys or AWS credentials

### Containers
- **Podman-First**: Optimized for Podman (rootless, daemonless) with Docker fallback
- **Multi-Agent Support**: Run Claude Code, OpenAI Codex, or OpenCode
- **Project Isolation**: Each project gets its own containerized environment
- **Development Profiles**: Pre-configured environments for Rust, Python, Go, Node.js, and more

### Usability
- **Cross-Platform**: Works on Linux, macOS, and Windows (via WSL2)
- **Zero Config**: Mounts your existing agent config directly - no import needed
- **Simple Commands**: Just run `agentbox claude` to get started

## Supported Agents

| Agent       | Description                  | Host Requirement |
|:------------|:-----------------------------|:-----------------|
| `claude`    | Anthropic's Claude Code CLI  | None (installed in container) |
| `codex`     | OpenAI's Codex CLI           | None (downloaded)|
| `opencode`  | OpenCode AI assistant        | None (official image + tools) |

**Note**: All agents are installed inside the container. If you have existing config (`~/.claude`, etc.), it will be mounted automatically.

## Installation

### Prerequisites

- **Podman** (recommended) or Docker
- Bash 3.2+
- Git

### Linux

```bash
# Install Podman (Ubuntu/Debian)
sudo apt update && sudo apt install -y podman

# Clone and install agentbox
git clone https://github.com/Cloud-Exit/agentbox.git ~/.agentbox
cd ~/.agentbox && chmod +x main.sh

# Add to PATH
mkdir -p ~/.local/bin
ln -sf ~/.agentbox/main.sh ~/.local/bin/agentbox

# Append shell aliases (e.g., 'claude' -> 'agentbox claude')
agentbox aliases >> ~/.bashrc
source ~/.bashrc

# Run an agent (builds image on first run)
agentbox claude
```

### macOS

```bash
# Install Podman
brew install podman
podman machine init && podman machine start

# Clone and install
git clone https://github.com/Cloud-Exit/agentbox.git ~/.agentbox
cd ~/.agentbox && chmod +x main.sh
mkdir -p ~/.local/bin
ln -sf ~/.agentbox/main.sh ~/.local/bin/agentbox
agentbox aliases >> ~/.zshrc
source ~/.zshrc
```

### Windows (WSL2)

```powershell
# In PowerShell as Administrator
wsl --install -d Ubuntu
```

Then in WSL2:
```bash
sudo apt update && sudo apt install -y podman
git clone https://github.com/Cloud-Exit/agentbox.git ~/.agentbox
cd ~/.agentbox && chmod +x main.sh
mkdir -p ~/.local/bin
ln -sf ~/.agentbox/main.sh ~/.local/bin/agentbox
agentbox aliases >> ~/.bashrc
source ~/.bashrc
```

## Quick Start

```bash
# Navigate to your project
cd /path/to/your/project

# Run an agent (builds image automatically on first run)
agentbox claude

# Or run other agents
agentbox codex
agentbox opencode
```

That's it! AgentBox automatically:
- Builds the container image if needed
- Mounts your existing config (`~/.claude`, `~/.codex`, etc.)
- Mounts your project directory
- Sets up the network firewall (Squid proxy)

## Commands

### Running Agents

```bash
agentbox claude [args]     # Run Claude Code
agentbox codex [args]      # Run Codex
agentbox opencode [args]   # Run OpenCode
```

### Updating Agents

Agents automatically check for updates each time you run them. If a new version is available, the container image will be rebuilt automatically.

You can also force a rebuild manually:

```bash
agentbox rebuild <agent>
```

### Management

```bash
agentbox list              # List available agents and build status
agentbox enable <agent>    # Enable an agent
agentbox disable <agent>   # Disable an agent
agentbox rebuild <agent>   # Force rebuild of agent image
agentbox uninstall <agent> # Remove agent images and config
agentbox aliases           # Print shell aliases for ~/.bashrc
```

### Profile Management

```bash
agentbox <agent> profile list       # List available profiles
agentbox <agent> profile add <name> # Add a development profile
agentbox <agent> profile remove <n> # Remove a profile
agentbox <agent> profile status     # Show current profiles
```

### Utilities

```bash
agentbox info              # Show system information
agentbox logs <agent>      # Show latest agent log file
agentbox clean             # Clean unused container resources
agentbox clean all         # Remove all agentbox images
agentbox projects          # List known projects
```

### Options

```bash
agentbox --no-firewall <agent>  # Disable network firewall for this run
agentbox --read-only <agent>    # Mount workspace as read-only (safety)
agentbox --verbose <agent>      # Enable verbose output
agentbox --include-dir /tmp/foo <agent>  # Mount /tmp/foo into /workspace/foo
```

## Available Profiles

| Profile       | Description                              |
|:--------------|:-----------------------------------------|
| `node`        | Node.js runtime with npm                 |
| `python`      | Python 3 with pip                        |
| `rust`        | Rust toolchain with cargo                |
| `go`          | Go runtime                               |
| `java`        | Java JDK with Maven and Gradle           |
| `ruby`        | Ruby with bundler                        |
| `php`         | PHP with composer                        |
| `c`           | C/C++ toolchain (gcc, make, cmake)       |
| `flutter`     | Flutter SDK                              |
| `dotnet`      | .NET SDK                                 |

## Configuration

### Resource Limits

AgentBox enforces default resource limits to prevent runaway agents:
- **Memory**: 8GB
- **CPU**: 4 vCPUs

### What Gets Mounted

AgentBox mounts your existing config directly from the host:

| Agent   | Host Path              | Container Path         |
|:--------|:-----------------------|:-----------------------|
| Claude  | `~/.claude`            | `/home/user/.claude`   |
| Claude  | `~/.claude.json`       | `/home/user/.claude.json` |
| Claude  | `~/.local/share/claude`| `/home/user/.local/share/claude` |
| Codex   | `~/.codex`             | `/home/user/.codex`    |
| OpenCode| `~/.opencode`          | `/home/user/.opencode` |

Your project directory is mounted at `/workspace`.

**Note:** SSH keys (`~/.ssh`) and AWS credentials (`~/.aws`) are **NOT** mounted by default for security.

### Environment Variables

| Variable              | Description                          |
|:----------------------|:-------------------------------------|
| `VERBOSE`             | Enable verbose output                |
| `CONTAINER_RUNTIME`   | Force runtime (`podman` or `docker`) |
| `AGENTBOX_NO_FIREWALL`| Disable firewall (`true`)            |

## Network Firewall

AgentBox uses a **Squid Proxy** container to enforce strict domain allowlisting:

1. **Proxy**: All HTTP/HTTPS traffic is routed through a local Squid instance.
2. **Allowlist**: Only domains listed in `allowlist.txt` (and their subdomains) are permitted.
3. **Reliability**: Works regardless of cloud IP changes (unlike iptables).

### Configuring the Allowlist

Edit `~/.config/agentbox/allowlist.txt` or the default at `config/allowlist.txt`.
Add root domains only (e.g., `google.com` allows `api.google.com`).

```bash
# Add a custom root domain
echo "mycompany.com" >> ~/.config/agentbox/allowlist.txt
```

### Disabling the Firewall

```bash
agentbox --no-firewall claude
```

## Why Podman?

| Feature              | Podman           | Docker               |
|:---------------------|:-----------------|:---------------------|
| Rootless by default  | Yes              | No (requires group)  |
| Daemonless           | Yes              | No (requires daemon) |
| Security             | Better isolation | Requires daemon root |

## Troubleshooting

### Podman: "cannot find UID/GID for user"

```bash
sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 $USER
podman system migrate
```

### Podman on macOS: "Cannot connect to Podman"

```bash
podman machine start
```

### Docker: Permission Denied

```bash
sudo usermod -aG docker $USER
newgrp docker
```

## Uninstallation

```bash
rm -rf ~/.agentbox ~/.local/bin/agentbox
rm -rf ~/.config/agentbox ~/.cache/agentbox
podman images | grep agentbox | awk '{print $3}' | xargs podman rmi -f
```

## License

MIT License - see LICENSE file.

## Contributing

Contributions welcome via pull requests and issues.
