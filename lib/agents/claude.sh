#!/usr/bin/env bash
# Claude agent module - Claude Code CLI specific configuration
# ============================================================================
# Installation: Official installer (https://claude.ai/install.sh)
# Source: Anthropic

# ============================================================================
# CLAUDE AGENT CONFIGURATION
# ============================================================================


# ============================================================================
# VERSION MANAGEMENT
# ============================================================================

# Get the installed Claude version from a Docker image
claude_get_installed_version() {
    local image_name="${1:-agentbox-claude-core}"
    local cmd
    cmd=$(container_cmd 2>/dev/null || printf 'podman')

    if ! $cmd image inspect "$image_name" >/dev/null 2>&1; then
        printf ''
        return 1
    fi

    # Run claude --version in the image
    local version
    version=$($cmd run --rm --entrypoint="" "$image_name" claude --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)
    printf '%s' "$version"
}

# Get the latest available Claude version
# Note: Claude uses native installer, version is determined at install time
claude_get_latest_version() {
    # Return the installed version
    claude_get_installed_version
}

# Check if Claude update is available
# Note: Always returns 1 (no update) since we use the installer which gets latest
claude_update_available() {
    return 1
}

# ============================================================================
# DOCKERFILE GENERATION
# ============================================================================

# Generate the Claude-specific Dockerfile installation commands
claude_get_dockerfile_install() {
    cat << 'EOF'
# Install Claude Code using official installer (as user so it goes to correct path)
USER user
RUN curl -fsSL https://claude.ai/install.sh -o /tmp/claude-install.sh && \
    bash /tmp/claude-install.sh && \
    rm -f /tmp/claude-install.sh && \
    command -v claude >/dev/null
USER root
EOF
}

# ============================================================================
# ENTRYPOINT LOGIC
# ============================================================================

# Get the Claude-specific entrypoint command
claude_get_entrypoint_command() {
    printf 'claude'
}

# Get Claude-specific environment variables for container
# Note: Claude uses config files, no env vars needed
claude_get_env_vars() {
    # No environment variables needed - Claude uses ~/.claude/ for auth
    :
}

# ============================================================================
# CONFIG PATHS
# ============================================================================

# Get the Claude credentials path inside container
claude_get_credentials_path() {
    printf '/home/user/.claude'
}

# Get the Claude config path inside container
claude_get_config_path() {
    printf '/home/user/.config'
}

# ============================================================================
# HOST CONFIG DETECTION
# ============================================================================

# Detect existing Claude installation on host
claude_detect_host_config() {
    local host_claude_dir="$HOME/.claude"

    if [[ -d "$host_claude_dir" ]]; then
        printf '%s' "$host_claude_dir"
        return 0
    fi

    return 1
}

# Get what Claude config files to import
claude_get_importable_files() {
    cat << 'EOF'
.credentials.json:Credentials:authentication
settings.json:Settings:user preferences
settings.local.json:Local Settings:MCP config
EOF
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f claude_get_installed_version claude_get_latest_version claude_update_available
export -f claude_get_dockerfile_install claude_get_entrypoint_command claude_get_env_vars
export -f claude_get_credentials_path claude_get_config_path
export -f claude_detect_host_config claude_get_importable_files
