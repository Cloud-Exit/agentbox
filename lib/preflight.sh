#!/usr/bin/env bash
# Preflight checks - Validation before running commands
# ============================================================================

# ============================================================================
# PREFLIGHT VALIDATION
# ============================================================================

# Run pre-flight checks for a command
# Returns 0 if checks pass, 1 if they fail
preflight_check() {
    local cmd="$1"
    shift
    local args=("$@")

    # Check if Docker is available
    if ! check_docker; then
        error "Docker is not available. Please install and start Docker."
    fi

    # Check agent-specific requirements
    if [[ -n "${AGENTBOX_CURRENT_AGENT:-}" ]]; then
        local agent="$AGENTBOX_CURRENT_AGENT"

        # Check if agent is enabled
        if ! agent_is_enabled "$agent"; then
            error "Agent '$agent' is not enabled. Run 'agentbox enable $agent' first."
        fi

        # All agents use their own config files for authentication
        # No environment variables needed
    fi

    return 0
}

# ============================================================================
# DEPENDENCY CHECKS
# ============================================================================

# Check if jq is available (needed for MCP config)
check_jq() {
    if ! command -v jq >/dev/null 2>&1; then
        warn "jq is not installed. MCP server configuration will not work."
        info "Install jq:"
        printf '  macOS: brew install jq\n'
        printf '  Ubuntu/Debian: apt-get install jq\n'
        printf '  RHEL/CentOS: yum install jq\n'
        return 1
    fi
    return 0
}

# Check if git is available
check_git() {
    if ! command -v git >/dev/null 2>&1; then
        warn "git is not installed. Some features may not work."
        return 1
    fi
    return 0
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f preflight_check
export -f check_jq check_git
