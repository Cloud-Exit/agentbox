#!/usr/bin/env bash
# Codex agent module - OpenAI Codex CLI specific configuration
# ============================================================================
# Installation: Binary download from GitHub releases
# Source: https://github.com/openai/codex/releases

# ============================================================================
# CODEX AGENT CONFIGURATION
# ============================================================================

CODEX_GITHUB_REPO="openai/codex"

# ============================================================================
# ARCHITECTURE DETECTION
# ============================================================================

# Get the Codex binary name for the current architecture
codex_get_binary_name() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64)
            printf 'codex-x86_64-unknown-linux-musl.tar.gz'
            ;;
        aarch64|arm64)
            printf 'codex-aarch64-unknown-linux-musl.tar.gz'
            ;;
        *)
            printf '' >&2
            return 1
            ;;
    esac
}

# ============================================================================
# VERSION MANAGEMENT
# ============================================================================

# Get the installed Codex version from a Docker image
codex_get_installed_version() {
    local image_name="${1:-agentbox-codex-core}"
    local cmd
    cmd=$(container_cmd 2>/dev/null || printf 'podman')

    if ! $cmd image inspect "$image_name" >/dev/null 2>&1; then
        printf ''
        return 1
    fi

    # Run codex --version in the image
    local version
    version=$($cmd run --rm "$image_name" codex --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)
    printf '%s' "$version"
}

# Get the latest available Codex version from GitHub
codex_get_latest_version() {
    # Fetch from GitHub API (keep full tag name like "rust-v0.97.0")
    local version
    version=$(curl -s "https://api.github.com/repos/${CODEX_GITHUB_REPO}/releases/latest" 2>/dev/null | \
        grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' || true)

    if [[ -n "$version" ]]; then
        printf '%s' "$version"
        return 0
    fi

    return 1
}

# Check if Codex update is available
codex_update_available() {
    local installed
    local latest

    installed=$(codex_get_installed_version) || return 1
    latest=$(codex_get_latest_version) || return 1

    if [[ -z "$installed" ]] || [[ -z "$latest" ]]; then
        return 1
    fi

    if [[ "$installed" != "$latest" ]]; then
        return 0
    fi

    return 1
}

# Get the download URL for the latest Codex release
codex_get_download_url() {
    local version="${1:-}"
    local binary_name

    binary_name=$(codex_get_binary_name) || return 1

    if [[ -z "$version" ]]; then
        version=$(codex_get_latest_version) || return 1
    fi

    printf 'https://github.com/%s/releases/download/%s/%s' "$CODEX_GITHUB_REPO" "$version" "$binary_name"
}

# ============================================================================
# DOCKERFILE GENERATION
# ============================================================================

# Generate the Codex-specific Dockerfile installation commands.
# The tarball is pre-downloaded by the host build script (image.sh) and
# passed into the build context so we can verify its SHA-256 checksum
# before installing.
codex_get_dockerfile_install() {
    local binary_name
    binary_name=$(codex_get_binary_name) || error "Unsupported architecture for Codex"

    # Binary name inside tarball (without .tar.gz extension)
    local binary_inside="${binary_name%.tar.gz}"

    cat << 'DOCKERFILE'
# Install Codex binary with SHA-256 verification
ARG CODEX_VERSION
ARG CODEX_CHECKSUM
DOCKERFILE
    cat << EOF
COPY ${binary_name} /tmp/codex.tar.gz
RUN echo "\${CODEX_CHECKSUM}  /tmp/codex.tar.gz" | sha256sum -c - && \\
    mkdir -p \$HOME/.local/bin && \\
    tar -xzf /tmp/codex.tar.gz -C /tmp && \\
    mv /tmp/${binary_inside} \$HOME/.local/bin/codex && \\
    chmod +x \$HOME/.local/bin/codex && \\
    rm -f /tmp/codex.tar.gz && \\
    \$HOME/.local/bin/codex --version
EOF
}

# ============================================================================
# ENTRYPOINT LOGIC
# ============================================================================

# Get the Codex-specific entrypoint command
codex_get_entrypoint_command() {
    printf 'codex'
}

# Get Codex-specific environment variables for container
codex_get_env_vars() {
    # No environment variables needed - Codex uses ~/.codex/ for auth
    :
}

# ============================================================================
# CONFIG PATHS
# ============================================================================

# Get the Codex credentials path inside container
codex_get_credentials_path() {
    printf '/home/user/.codex'
}

# Get the Codex config path inside container
codex_get_config_path() {
    printf '/home/user/.config/codex'
}

# ============================================================================
# HOST CONFIG DETECTION
# ============================================================================

# Detect existing Codex installation on host
codex_detect_host_config() {
    local host_codex_dir="$HOME/.codex"
    local host_config_dir="$HOME/.config/codex"

    if [[ -d "$host_codex_dir" ]]; then
        printf '%s' "$host_codex_dir"
        return 0
    fi

    if [[ -d "$host_config_dir" ]]; then
        printf '%s' "$host_config_dir"
        return 0
    fi

    return 1
}

# Get what Codex config files to import
codex_get_importable_files() {
    cat << 'EOF'
auth.json:Auth tokens:API authentication
config.json:Settings:user preferences
EOF
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f codex_get_binary_name
export -f codex_get_installed_version codex_get_latest_version codex_update_available codex_get_download_url
export -f codex_get_dockerfile_install codex_get_entrypoint_command codex_get_env_vars
export -f codex_get_credentials_path codex_get_config_path
export -f codex_detect_host_config codex_get_importable_files
