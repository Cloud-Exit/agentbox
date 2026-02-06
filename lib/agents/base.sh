#!/usr/bin/env bash
# Agent base module - Common interface and utilities for all agents
# ============================================================================
# This module defines the agent abstraction layer that enables agentbox to
# support multiple AI coding assistants (Claude, Codex, OpenCode) as drop-in
# replacements.

# ============================================================================
# SUPPORTED AGENTS
# ============================================================================
if [[ -z "${AGENTBOX_AGENTS[*]:-}" ]]; then
    AGENTBOX_AGENTS=(claude codex opencode)
fi

# ============================================================================
# AGENT DETECTION & CONFIGURATION PATHS
# ============================================================================

# Host config paths for detecting existing installations
get_agent_host_config_paths() {
    local agent="$1"
    case "$agent" in
        claude)
            printf '%s\n' "$HOME/.claude"
            ;;
        codex)
            printf '%s\n' "$HOME/.codex"
            printf '%s\n' "$HOME/.config/codex"
            ;;
        opencode)
            printf '%s\n' "$HOME/.opencode"
            printf '%s\n' "$HOME/.config/opencode"
            ;;
        *)
            return 1
            ;;
    esac
}

# Get the agentbox config directory for an agent
get_agent_config_dir() {
    local agent="$1"
    printf '%s/%s' "${AGENTBOX_HOME:-$HOME/.config/agentbox}" "$agent"
}

# Get the per-project config directory for an agent
get_agent_project_config_dir() {
    local agent="$1"
    local project_parent="${PROJECT_PARENT_DIR:-}"

    if [[ -z "$project_parent" ]]; then
        return 1
    fi

    printf '%s/%s' "$project_parent" "$agent"
}

# ============================================================================
# AGENT VALIDATION
# ============================================================================

# Check if an agent name is valid
agent_exists() {
    local agent="$1"
    local a
    for a in "${AGENTBOX_AGENTS[@]}"; do
        if [[ "$a" == "$agent" ]]; then
            return 0
        fi
    done
    return 1
}

# Check if an agent is enabled
agent_is_enabled() {
    local agent="$1"
    local config_file="${AGENTBOX_HOME:-$HOME/.config/agentbox}/config.ini"

    if [[ ! -f "$config_file" ]]; then
        return 1
    fi

    # Check if agent is in enabled list
    if grep -q "^${agent}=enabled$" "$config_file" 2>/dev/null; then
        return 0
    fi

    return 1
}

# Get list of enabled agents
get_enabled_agents() {
    local config_file="${AGENTBOX_HOME:-$HOME/.config/agentbox}/config.ini"
    local enabled=()

    if [[ -f "$config_file" ]]; then
        local agent
        for agent in "${AGENTBOX_AGENTS[@]}"; do
            if grep -q "^${agent}=enabled$" "$config_file" 2>/dev/null; then
                enabled+=("$agent")
            fi
        done
    fi

    printf '%s\n' "${enabled[@]}"
}

# ============================================================================
# AGENT ENABLE/DISABLE
# ============================================================================

# Enable an agent
enable_agent() {
    local agent="$1"
    local config_dir="${AGENTBOX_HOME:-$HOME/.config/agentbox}"
    local config_file="$config_dir/config.ini"

    # Validate agent name
    if ! agent_exists "$agent"; then
        printf 'Unknown agent: %s\n' "$agent" >&2
        return 1
    fi

    # Create config directory structure
    mkdir -p "$config_dir/$agent"

    # Create or update config file
    if [[ ! -f "$config_file" ]]; then
        printf '[agents]\n' > "$config_file"
    fi

    # Remove existing entry for this agent
    if [[ -f "$config_file" ]]; then
        local temp_file
        temp_file=$(mktemp)
        grep -v "^${agent}=" "$config_file" > "$temp_file" 2>/dev/null || true
        mv "$temp_file" "$config_file"
    fi

    # Add enabled entry
    printf '%s=enabled\n' "$agent" >> "$config_file"

    return 0
}

# Disable an agent
disable_agent() {
    local agent="$1"
    local config_file="${AGENTBOX_HOME:-$HOME/.config/agentbox}/config.ini"

    # Validate agent name
    if ! agent_exists "$agent"; then
        printf 'Unknown agent: %s\n' "$agent" >&2
        return 1
    fi

    if [[ ! -f "$config_file" ]]; then
        return 0
    fi

    # Remove existing entry for this agent
    local temp_file
    temp_file=$(mktemp)
    grep -v "^${agent}=" "$config_file" > "$temp_file" 2>/dev/null || true
    mv "$temp_file" "$config_file"

    # Add disabled entry
    printf '%s=disabled\n' "$agent" >> "$config_file"

    return 0
}

# ============================================================================
# AGENT INTERFACE FUNCTIONS (must be implemented by each agent module)
# ============================================================================

# Get agent display name
# Usage: agent_get_display_name <agent>
agent_get_display_name() {
    local agent="$1"
    case "$agent" in
        claude)   printf 'Claude Code' ;;
        codex)    printf 'OpenAI Codex' ;;
        opencode) printf 'OpenCode' ;;
        *)        printf '%s' "$agent" ;;
    esac
}

# Get agent installation method
# Returns: npm, binary, or pip
# Usage: agent_get_install_method <agent>
agent_get_install_method() {
    local agent="$1"
    case "$agent" in
        claude)   printf 'npm' ;;
        codex)    printf 'binary' ;;
        opencode) printf 'binary' ;;
        *)        printf 'unknown' ;;
    esac
}

# Get agent CLI command name
# Usage: agent_get_cli_command <agent>
agent_get_cli_command() {
    local agent="$1"
    case "$agent" in
        claude)   printf 'claude' ;;
        codex)    printf 'codex' ;;
        opencode) printf 'opencode' ;;
        *)        printf '%s' "$agent" ;;
    esac
}

# ============================================================================
# DOCKER IMAGE NAMING
# ============================================================================

# Get the base image name for an agent
get_agent_base_image() {
    local agent="$1"
    printf 'agentbox-%s-core' "$agent"
}

# Get the project image name for an agent
get_agent_project_image() {
    local agent="$1"
    local project_hash="$2"
    printf 'agentbox-%s-%s' "$agent" "$project_hash"
}

# Get the container name for an agent in a project
# Appends a short random suffix to allow multiple parallel instances
get_agent_container_name() {
    local agent="$1"
    local project_hash="$2"
    local suffix
    suffix=$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n')
    printf 'agentbox-%s-%s-%s' "$agent" "$project_hash" "$suffix"
}

# ============================================================================
# EXISTING CONFIG DETECTION & IMPORT
# ============================================================================

# Check if an agent has existing config on host
detect_existing_agent_config() {
    local agent="$1"
    local config_paths

    config_paths=$(get_agent_host_config_paths "$agent") || return 1

    while IFS= read -r path; do
        if [[ -d "$path" ]]; then
            printf '%s' "$path"
            return 0
        fi
    done <<< "$config_paths"

    return 1
}

# Import existing config for an agent
import_agent_config() {
    local agent="$1"
    local source_path="$2"
    local dest_dir

    dest_dir=$(get_agent_config_dir "$agent")
    mkdir -p "$dest_dir"

    case "$agent" in
        claude)
            # Import Claude credentials and settings
            if [[ -d "$source_path" ]]; then
                # Copy credentials
                if [[ -f "$source_path/.credentials.json" ]]; then
                    mkdir -p "$dest_dir/.claude"
                    cp "$source_path/.credentials.json" "$dest_dir/.claude/"
                fi
                # Copy settings
                if [[ -f "$source_path/settings.json" ]]; then
                    mkdir -p "$dest_dir/.claude"
                    cp "$source_path/settings.json" "$dest_dir/.claude/"
                fi
                # Copy MCP config if present
                if [[ -f "$source_path/settings.local.json" ]]; then
                    mkdir -p "$dest_dir/.claude"
                    cp "$source_path/settings.local.json" "$dest_dir/.claude/"
                fi
            fi
            ;;
        codex)
            # Import Codex tokens and settings
            if [[ -d "$source_path" ]]; then
                if [[ "$source_path" == "$HOME/.config/codex" ]]; then
                    mkdir -p "$dest_dir/.config/codex"
                    cp -r "$source_path"/* "$dest_dir/.config/codex/" 2>/dev/null || true
                else
                    mkdir -p "$dest_dir/.codex"
                    cp -r "$source_path"/* "$dest_dir/.codex/" 2>/dev/null || true
                fi
            fi
            ;;
        opencode)
            # Import OpenCode config
            if [[ -d "$source_path" ]]; then
                if [[ "$source_path" == "$HOME/.config/opencode" ]]; then
                    mkdir -p "$dest_dir/.config/opencode"
                    cp -r "$source_path"/* "$dest_dir/.config/opencode/" 2>/dev/null || true
                else
                    mkdir -p "$dest_dir/.opencode"
                    cp -r "$source_path"/* "$dest_dir/.opencode/" 2>/dev/null || true
                fi
            fi
            ;;
    esac

    return 0
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f get_agent_host_config_paths get_agent_config_dir get_agent_project_config_dir
export -f agent_exists agent_is_enabled get_enabled_agents
export -f enable_agent disable_agent
export -f agent_get_display_name agent_get_install_method agent_get_cli_command
export -f get_agent_base_image get_agent_project_image get_agent_container_name
export -f detect_existing_agent_config import_agent_config
