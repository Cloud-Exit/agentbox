#!/usr/bin/env bash
# Agent Commands - Multi-agent management
# ============================================================================
# Commands: list, enable, disable, aliases
# Manages multiple AI coding assistant agents (Claude, Codex, OpenCode)

# Agent modules are loaded by main.sh before this file is sourced.

# ============================================================================
# LIST COMMAND
# ============================================================================

_cmd_agent_list() {
    logo_small
    printf '\n'
    cecho "Available Agents:" "$CYAN"
    printf '\n'
    printf '  %-12s %-15s %-10s %-10s\n' "AGENT" "DISPLAY NAME" "ENABLED" "IMAGE"
    printf '  %-12s %-15s %-10s %-10s\n' "-----" "------------" "-------" "-----"

    local agent
    for agent in "${AGENTBOX_AGENTS[@]}"; do
        local display_name
        local enabled_text
        local enabled_color
        local image_text
        local image_color

        display_name=$(agent_get_display_name "$agent")

        if agent_is_enabled "$agent"; then
            enabled_text="yes"
            enabled_color="$GREEN"
        else
            enabled_text="no"
            enabled_color="$DIM"
        fi

        if container_image_exists "agentbox-${agent}-core"; then
            image_text="built"
            image_color="$GREEN"
        else
            image_text="not built"
            image_color="$DIM"
        fi

        printf '  %-12s %-15s %b%-10s%b %b%-10s%b\n' \
            "$agent" "$display_name" \
            "$enabled_color" "$enabled_text" "$NC" \
            "$image_color" "$image_text" "$NC"
    done

    printf '\n'
    printf 'Commands:\n'
    printf '  agentbox <agent>            Run an agent (builds if needed)\n'
    printf '  agentbox enable <agent>     Enable an agent\n'
    printf '  agentbox disable <agent>    Disable an agent\n'
    printf '  agentbox rebuild <agent>    Force rebuild of agent image\n'
    printf '  agentbox import <agent>     Import agent config from host\n'
    printf '  agentbox uninstall [agent]  Uninstall agentbox or specific agent\n'
    printf '  agentbox aliases            Print shell aliases\n'
    printf '\n'
}

# ============================================================================
# REBUILD COMMAND
# ============================================================================

_cmd_agent_rebuild() {
    local agent="${1:-}"

    if [[ -z "$agent" ]]; then
        error "Usage: agentbox rebuild <agent>"
    fi

    # Validate agent name
    if ! agent_exists "$agent"; then
        error "Unknown agent: $agent"
    fi

    local display_name
    display_name=$(agent_get_display_name "$agent")

    # Remove all project images for this agent (they'll be rebuilt on next run)
    local cmd
    cmd=$(container_cmd)
    $cmd images --filter "reference=agentbox-${agent}-*" --format "{{.Repository}}" 2>/dev/null | \
        grep -v "agentbox-${agent}-core" | \
        while read -r image; do
            $cmd rmi -f "$image" 2>/dev/null || true
        done || true

    info "Rebuilding $display_name container image..."
    if build_agent_core_image "$agent" "true"; then
        success "$display_name image rebuilt successfully"
    else
        error "Failed to rebuild $display_name image"
    fi
}

# ============================================================================
# IMPORT COMMAND
# ============================================================================

_cmd_agent_import() {
    local agent="${1:-}"

    if [[ -z "$agent" ]]; then
        error "Usage: agentbox import <agent>|all"
    fi

    local agents=()
    if [[ "$agent" == "all" ]]; then
        agents=("${AGENTBOX_AGENTS[@]}")
    else
        if ! agent_exists "$agent"; then
            error "Unknown agent: $agent"
        fi
        agents=("$agent")
    fi

    local imported_any=false
    local a
    for a in "${agents[@]}"; do
        local source_path=""
        source_path=$(detect_existing_agent_config "$a" 2>/dev/null || true)

        if [[ -z "$source_path" ]]; then
            warn "No host config found for $a"
            continue
        fi

        import_agent_config "$a" "$source_path"
        success "Imported $a config from $source_path"
        imported_any=true
    done

    if [[ "$imported_any" == "false" ]]; then
        return 1
    fi
}

# ============================================================================
# UNINSTALL COMMAND
# ============================================================================

_cmd_agent_uninstall() {
    local agent="${1:-}"

    if [[ -z "$agent" ]]; then
        printf 'This will UNINSTALL AGENTBOX COMPLETELY.\n'
        printf 'Actions:\n'
        printf '  - Stop and remove all agentbox containers\n'
        printf '  - Remove all agentbox images\n'
        printf '  - Disable all agents\n'
        printf '  - Remove all agent configurations\n'
        printf '\n'
        printf 'Are you sure? [y/N] '
        local response
        read -r response

        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            info "Cancelled"
            return 0
        fi

        local cmd
        cmd=$(container_cmd)

        # Stop and remove all containers
        info "Stopping and removing all agentbox containers..."
        $cmd ps -a --filter "name=agentbox-*" --format "{{.ID}}" | \
            xargs -r $cmd rm -f >/dev/null 2>&1 || true

        # Remove all images
        info "Removing all agentbox images..."
        clean_container_resources "all"

        # Process each agent
        local a
        for a in "${AGENTBOX_AGENTS[@]}"; do
            # Disable agent
            disable_agent "$a" 2>/dev/null || true
            
            # Remove config directory
            local config_dir
            config_dir=$(get_agent_config_dir "$a")
            if [[ -d "$config_dir" ]]; then
                info "Removing config for $a..."
                rm -rf "$config_dir"
            fi
        done
        
        # Remove cache
        if [[ -d "${AGENTBOX_CACHE:-}" ]]; then
             rm -rf "$AGENTBOX_CACHE"
        fi

        success "AgentBox uninstalled successfully."
        return 0
    fi

    # Validate agent name
    if ! agent_exists "$agent"; then
        error "Unknown agent: $agent"
    fi

    local display_name
    display_name=$(agent_get_display_name "$agent")

    printf 'This will remove all %s images and configuration.\n' "$display_name"
    printf 'Are you sure? [y/N] '
    local response
    read -r response

    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        info "Cancelled"
        return 0
    fi

    # Remove agent images
    info "Removing $display_name container images..."
    _remove_agent_images "$agent"

    # Remove config directory
    local config_dir
    config_dir=$(get_agent_config_dir "$agent")
    if [[ -d "$config_dir" ]]; then
        info "Removing config at $config_dir..."
        rm -rf "$config_dir"
    fi

    # Disable the agent
    disable_agent "$agent" 2>/dev/null || true

    success "$display_name completely uninstalled"
}

# Remove all images for an agent
_remove_agent_images() {
    local agent="$1"
    local cmd
    cmd=$(container_cmd)

    # Remove core image
    local core_image="agentbox-${agent}-core"
    if container_image_exists "$core_image"; then
        $cmd rmi -f "$core_image" 2>/dev/null || true
    fi

    # Remove all project images for this agent
    $cmd images --filter "reference=agentbox-${agent}-*" --format "{{.Repository}}:{{.Tag}}" 2>/dev/null | \
        while read -r image; do
            $cmd rmi -f "$image" 2>/dev/null || true
        done
}

# ============================================================================
# ALIASES COMMAND
# ============================================================================

_cmd_agent_aliases() {
    printf '# Agentbox setup - add to ~/.zshrc or ~/.bashrc\n'
    printf '\n'

    # PATH setup
    printf '# Add agentbox to PATH\n'
    printf 'export PATH="$HOME/.local/bin:$PATH"\n'
    printf '\n'

    # Agent aliases for all available agents
    printf '# Agent aliases\n'

    local agent
    for agent in "${AGENTBOX_AGENTS[@]}"; do
        printf 'alias %s='\''agentbox %s'\''\n' "$agent" "$agent"
    done
}

# ============================================================================
# AGENT RUN COMMAND (dispatches to specific agent)
# ============================================================================

_cmd_agent_run() {
    local agent="${1:-}"
    shift || true

    if [[ -z "$agent" ]]; then
        error "Usage: agentbox <agent> [args...]"
    fi

    # Validate agent name
    if ! agent_exists "$agent"; then
        return 1  # Not an agent, let caller handle
    fi

    # Set the current agent for downstream functions
    export AGENTBOX_CURRENT_AGENT="$agent"

    # Check for updates only when explicitly requested via --update / -u
    if [[ "${CLI_UPDATE:-false}" == "true" ]]; then
        local new_version
        if new_version=$(check_agent_update "$agent"); then
             info "Update available for $agent: $new_version"
             info "Rebuilding image..."
             build_agent_core_image "$agent" "true"
        else
             info "$agent is up to date"
        fi
    fi

    # Build image if it doesn't exist
    local core_image="agentbox-${agent}-core"
    if ! container_image_exists "$core_image"; then
        info "Building $agent image (first run)..."
        if ! build_agent_core_image "$agent" "true"; then
            error "Failed to build $agent image"
        fi
    fi

    # Get project hash for container naming
    local project_hash
    project_hash=$(generate_project_folder_name "$PROJECT_DIR")

    # Get container name
    local container_name
    container_name=$(get_agent_container_name "$agent" "$project_hash")

    # Run the container
    local passthrough=("$@")
    if [[ "$agent" == "opencode" ]] && [[ "${VERBOSE:-false}" == "true" ]]; then
        local has_log_level=false
        local arg
        for arg in "${passthrough[@]+"${passthrough[@]}"}"; do
            if [[ "$arg" == "--log-level" ]] || [[ "$arg" == "--log-level="* ]]; then
                has_log_level=true
                break
            fi
        done
        if [[ "$has_log_level" == "false" ]]; then
            passthrough=(--log-level DEBUG "${passthrough[@]+"${passthrough[@]}"}")
        fi
    fi
    run_agent_container "$agent" "$container_name" "interactive" "${passthrough[@]+"${passthrough[@]}"}"
}

# ============================================================================
# AGENT LOGS COMMAND
# ============================================================================

_cmd_agent_logs() {
    local agent="${1:-}"
    shift || true

    if [[ -z "$agent" ]]; then
        error "Usage: agentbox logs <agent>"
    fi

    # Validate agent name
    if ! agent_exists "$agent"; then
        error "Unknown agent: $agent"
    fi

    local agent_config_dir
    agent_config_dir=$(get_agent_config_dir "$agent")

    local search_dirs=()
    case "$agent" in
        opencode)
            search_dirs+=(
                "$HOME/.local/share/opencode/log"
                "$HOME/.local/share/opencode/logs"
                "$HOME/.opencode"
                "$agent_config_dir/.opencode"
                "$agent_config_dir/.config/opencode"
            )
            ;;
        codex)
            search_dirs+=(
                "$HOME/.codex"
                "$HOME/.config/codex"
                "$agent_config_dir/.codex"
                "$agent_config_dir/.config/codex"
            )
            ;;
        claude)
            search_dirs+=(
                "$HOME/.claude"
                "$agent_config_dir/.claude"
            )
            ;;
    esac

    # Always include agent config dir as a fallback search root
    search_dirs+=("$agent_config_dir")

    # Deduplicate directories
    local unique_dirs=()
    local seen=" "
    local dir
    for dir in "${search_dirs[@]}"; do
        if [[ -z "$dir" ]]; then
            continue
        fi
        if [[ "$seen" != *" $dir "* ]]; then
            unique_dirs+=("$dir")
            seen="$seen$dir "
        fi
    done

    local log_files=()
    for dir in "${unique_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            while IFS= read -r file; do
                log_files+=("$file")
            done < <(find "$dir" -maxdepth 5 -type f -name '*.log' 2>/dev/null || true)
        fi
    done

    if [[ ${#log_files[@]} -eq 0 ]]; then
        error "No log files found for $agent (searched: ${unique_dirs[*]})"
    fi

    local latest=""
    local latest_ts=0
    local file
    for file in "${log_files[@]}"; do
        local ts
        ts=$(stat -c %Y "$file" 2>/dev/null || stat -f %m "$file" 2>/dev/null || echo 0)
        if [[ "$ts" -ge "$latest_ts" ]]; then
            latest_ts="$ts"
            latest="$file"
        fi
    done

    if [[ -z "$latest" ]]; then
        error "No readable log files found for $agent"
    fi

    printf '==> %s <==\n' "$latest"
    tail -n "${AGENTBOX_LOG_TAIL:-200}" "$latest"
}

# ============================================================================
# AGENT PROFILE SUBCOMMANDS
# ============================================================================

_cmd_agent_profile() {
    local agent="${1:-}"
    local subcommand="${2:-}"
    shift 2 || true

    if [[ -z "$agent" ]] || [[ -z "$subcommand" ]]; then
        printf 'Usage: agentbox <agent> profile <command>\n'
        printf '\n'
        printf 'Commands:\n'
        printf '  list         List available profiles\n'
        printf '  add <name>   Add a profile\n'
        printf '  remove <name>   Remove a profile\n'
        printf '  status       Show current profiles\n'
        printf '\n'
        return 1
    fi

    # Set current agent for profile commands
    export AGENTBOX_CURRENT_AGENT="$agent"

    case "$subcommand" in
        list)
            _cmd_profiles "$@"
            ;;
        add)
            _cmd_add "$@"
            ;;
        remove)
            _cmd_remove "$@"
            ;;
        status)
            _cmd_profile_status "$@"
            ;;
        *)
            error "Unknown profile command: $subcommand"
            ;;
    esac
}

# Show current profile status for an agent
_cmd_profile_status() {
    local agent="${AGENTBOX_CURRENT_AGENT:-claude}"
    local display_name
    display_name=$(agent_get_display_name "$agent")

    printf '\n'
    cecho "$display_name Profile Status" "$CYAN"
    printf '\n'

    local current_profiles=()
    local line
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            current_profiles+=("$line")
        fi
    done < <(get_project_profiles "$agent")

    if [[ ${#current_profiles[@]} -gt 0 ]]; then
        printf 'Active profiles:\n'
        local profile
        for profile in "${current_profiles[@]}"; do
            local desc
            desc=$(get_profile_description "$profile")
            printf '  %s - %s\n' "$profile" "$desc"
        done
    else
        printf 'No profiles configured.\n'
    fi

    printf '\n'
    printf 'Run '\''agentbox %s profile add <name>'\'' to add profiles.\n' "$agent"
    printf '\n'
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f _cmd_agent_list _cmd_agent_rebuild _cmd_agent_uninstall _cmd_agent_aliases
export -f _cmd_agent_run _cmd_agent_logs _cmd_agent_profile _cmd_profile_status
export -f _remove_agent_images
