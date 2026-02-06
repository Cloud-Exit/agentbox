#!/usr/bin/env bash
# Command dispatcher - Routes commands to appropriate handlers
# ============================================================================

# ============================================================================
# COMMAND REQUIREMENTS
# ============================================================================

# Get what a command requires to run
# Returns: "none", "image", or "docker"
get_command_requirements() {
    local cmd="$1"
    local subcmd="${2:-}"

    case "$cmd" in
        # Commands that need nothing
        help|version|aliases|logs|import)
            printf 'none'
            ;;
        # Commands that need Docker but not a specific image
        list|rebuild|uninstall|info|clean|projects|enable|disable)
            printf 'docker'
            ;;
        # Profile commands depend on subcommand
        profile)
            case "$subcmd" in
                list|available)
                    printf 'none'
                    ;;
                *)
                    printf 'docker'
                    ;;
            esac
            ;;
        # Agent execution needs full Docker setup
        *)
            printf 'docker'
            ;;
    esac
}

# ============================================================================
# COMMAND DISPATCH
# ============================================================================

# Main command dispatcher
dispatch_command() {
    local cmd="$1"
    shift
    local args=("$@")

    case "$cmd" in
        # Agent management commands
        list)
            _cmd_agent_list
            ;;
        rebuild)
            _cmd_agent_rebuild "${args[@]}"
            ;;
        uninstall)
            _cmd_agent_uninstall "${args[@]}"
            ;;
        enable)
            _cmd_enable "${args[@]}"
            ;;
        disable)
            _cmd_disable "${args[@]}"
            ;;
        aliases)
            _cmd_agent_aliases
            ;;

        logs)
            _cmd_agent_logs "${args[@]}"
            ;;
        import)
            _cmd_agent_import "${args[@]}"
            ;;

        # Profile commands (agent-specific)
        profile)
            if [[ -n "${AGENTBOX_CURRENT_AGENT:-}" ]]; then
                _cmd_agent_profile "${AGENTBOX_CURRENT_AGENT}" "${args[@]}"
            else
                _cmd_profiles "${args[@]}"
            fi
            ;;

        # Utility commands
        help)
            print_usage
            ;;
        version)
            print_version
            ;;
        info)
            _cmd_info "${args[@]}"
            ;;
        clean)
            _cmd_clean "${args[@]}"
            ;;
        projects)
            _cmd_projects "${args[@]}"
            ;;

        # Profile shortcut commands
        profiles)
            _cmd_profiles "${args[@]}"
            ;;
        add)
            _cmd_add "${args[@]}"
            ;;
        remove)
            _cmd_remove "${args[@]}"
            ;;

        # Unknown command
        *)
            error "Unknown command: $cmd. Run 'agentbox help' for usage."
            ;;
    esac
}

# ============================================================================
# UTILITY COMMAND IMPLEMENTATIONS
# ============================================================================

# Show system info
_cmd_info() {
    logo_small
    printf '\n'
    cecho "System Information" "$CYAN"
    printf '\n'

    printf '  %-20s %s\n' "Version:" "${AGENTBOX_VERSION:-unknown}"
    printf '  %-20s %s\n' "Platform:" "$(get_platform)"
    printf '  %-20s %s\n' "Config dir:" "$AGENTBOX_HOME"
    printf '  %-20s %s\n' "Cache dir:" "$AGENTBOX_CACHE"

    printf '\n'
    cecho "Container Runtime" "$CYAN"
    printf '\n'

    if detect_container_runtime; then
        local cmd
        cmd=$(container_cmd)
        printf '  %-20s %s\n' "Runtime:" "$cmd"
        printf '  %-20s %s\n' "Version:" "$CONTAINER_RUNTIME_VERSION"

        if check_container_runtime; then
            printf '  %-20s %s\n' "Status:" "${GREEN}running${NC}"

            # Count images
            local base_count
            base_count=$($cmd images --filter "reference=agentbox-base" --format "{{.ID}}" 2>/dev/null | wc -l)
            local core_count
            core_count=$($cmd images --filter "reference=agentbox-*-core" --format "{{.ID}}" 2>/dev/null | wc -l)
            local project_count
            project_count=$($cmd images --filter "reference=agentbox-*" --format "{{.ID}}" 2>/dev/null | wc -l)

            printf '  %-20s %s\n' "Base images:" "$base_count"
            printf '  %-20s %s\n' "Core images:" "$core_count"
            printf '  %-20s %s\n' "Project images:" "$((project_count - base_count - core_count))"
        else
            printf '  %-20s %s\n' "Status:" "${RED}not running${NC}"
        fi
    else
        printf '  %-20s %s\n' "Runtime:" "${RED}not found${NC}"
        printf '  Install Podman (recommended) or Docker to use agentbox.\n'
    fi

    printf '\n'
    cecho "Built Agents" "$CYAN"
    printf '\n'

    local found_any=false
    local agent
    for agent in "${AGENTBOX_AGENTS[@]}"; do
        if container_image_exists "agentbox-${agent}-core"; then
            local display_name
            display_name=$(agent_get_display_name "$agent")
            printf '  • %s (%s)\n' "$display_name" "$agent"
            found_any=true
        fi
    done
    if [[ "$found_any" != "true" ]]; then
        printf '  No agents built. Run '\''agentbox <agent>'\'' to build and run one.\n'
    fi

    printf '\n'
}

# Clean command
_cmd_clean() {
    local mode="${1:-unused}"

    case "$mode" in
        unused|all|containers)
            clean_docker_resources "$mode"
            success "Cleanup complete"
            ;;
        *)
            printf 'Usage: agentbox clean [unused|all|containers]\n'
            printf '\n'
            printf 'Modes:\n'
            printf '  unused      Remove unused images (default)\n'
            printf '  all         Remove all agentbox images\n'
            printf '  containers  Stop all agentbox containers\n'
            printf '\n'
            ;;
    esac
}

# Projects command
_cmd_projects() {
    list_all_projects
}

# Enable an agent
_cmd_enable() {
    local agent="${1:-}"

    if [[ -z "$agent" ]]; then
        error "Usage: agentbox enable <agent>"
    fi

    if ! agent_exists "$agent"; then
        error "Unknown agent: $agent"
    fi

    if agent_is_enabled "$agent"; then
        info "Agent '$agent' is already enabled"
        return 0
    fi

    enable_agent "$agent"
    local display_name
    display_name=$(agent_get_display_name "$agent")
    success "$display_name enabled"
    info "Run 'agentbox $agent' to start using it"
}

# Disable an agent
_cmd_disable() {
    local agent="${1:-}"

    if [[ -z "$agent" ]]; then
        error "Usage: agentbox disable <agent>"
    fi

    if ! agent_exists "$agent"; then
        error "Unknown agent: $agent"
    fi

    if ! agent_is_enabled "$agent"; then
        info "Agent '$agent' is already disabled"
        return 0
    fi

    disable_agent "$agent"
    local display_name
    display_name=$(agent_get_display_name "$agent")
    success "$display_name disabled"
}

# ============================================================================
# PROFILE COMMAND IMPLEMENTATIONS
# ============================================================================

# List available profiles
_cmd_profiles() {
    logo_small
    printf '\n'
    cecho "Available Profiles:" "$CYAN"
    printf '\n'

    local profiles=(
        "base:Base development tools (git, vim, curl)"
        "node:Node.js runtime with npm"
        "python:Python 3 with pip and venv"
        "rust:Rust toolchain with cargo"
        "go:Go runtime"
        "java:Java JDK"
        "ruby:Ruby runtime with bundler"
        "php:PHP runtime with composer"
        "dotnet:.NET SDK"
        "c:C/C++ toolchain (gcc, make, cmake)"
        "flutter:Flutter SDK for mobile development"
        "android:Android SDK and tools"
        "docker:Docker-in-Docker support"
        "kubernetes:kubectl and helm"
        "terraform:Terraform CLI"
        "aws:AWS CLI"
        "gcloud:Google Cloud SDK"
        "azure:Azure CLI"
    )

    printf '  %-15s %s\n' "PROFILE" "DESCRIPTION"
    printf '  %-15s %s\n' "───────" "───────────"

    for entry in "${profiles[@]}"; do
        local name="${entry%%:*}"
        local desc="${entry#*:}"
        printf '  %-15s %s\n' "$name" "$desc"
    done

    printf '\n'
    printf 'Add profiles with: agentbox <agent> profile add <name>\n'
    printf '\n'
}

# Add a profile
_cmd_add() {
    local profile="${1:-}"

    if [[ -z "$profile" ]]; then
        error "Usage: agentbox add <profile>"
    fi

    local agent="${AGENTBOX_CURRENT_AGENT:-claude}"
    add_project_profile "$agent" "$profile"
    success "Added profile '$profile' for $agent"
    info "Run 'agentbox $agent' to rebuild with the new profile"
}

# Remove a profile
_cmd_remove() {
    local profile="${1:-}"

    if [[ -z "$profile" ]]; then
        error "Usage: agentbox remove <profile>"
    fi

    local agent="${AGENTBOX_CURRENT_AGENT:-claude}"
    remove_project_profile "$agent" "$profile"
    success "Removed profile '$profile' for $agent"
    info "Run 'agentbox $agent' to rebuild without the profile"
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f get_command_requirements dispatch_command
export -f _cmd_info _cmd_clean _cmd_projects _cmd_enable _cmd_disable
export -f _cmd_profiles _cmd_add _cmd_remove
