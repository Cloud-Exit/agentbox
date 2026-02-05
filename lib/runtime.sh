#!/usr/bin/env bash
# Runtime management - Running containers
# ============================================================================

# Count running agent containers (excluding squid proxy).
count_running_agent_containers() {
    local cmd
    cmd=$(container_cmd)

    local count=0
    local name
    while IFS= read -r name; do
        [[ -z "$name" ]] && continue
        [[ "$name" == "$AGENTBOX_SQUID_CONTAINER" ]] && continue
        if [[ "$name" == agentbox-* ]]; then
            ((count++))
        fi
    done < <($cmd ps --format '{{.Names}}' 2>/dev/null || true)

    printf '%s' "$count"
}

# Stop/remove squid proxy if no agent containers are running.
cleanup_squid_if_unused() {
    local cmd
    cmd=$(container_cmd)

    local running_agents_count
    running_agents_count=$(count_running_agent_containers)

    if [[ "$running_agents_count" == "0" ]]; then
        if $cmd ps --filter "name=^/${AGENTBOX_SQUID_CONTAINER}$" --format '{{.Names}}' | grep -q "^${AGENTBOX_SQUID_CONTAINER}$"; then
            info "Stopping Squid proxy (no running agents)..."
            $cmd rm -f "$AGENTBOX_SQUID_CONTAINER" >/dev/null 2>&1 || true
        fi
    fi
}

# Run an agent container
# Usage: run_agent_container <agent> <container_name> <mode> [args...]
run_agent_container() {
    local agent="$1"
    local container_name="$2"
    local run_mode="$3"  # "interactive", "detached", "pipe"
    shift 3
    local container_args=("$@")

    local cmd
    cmd=$(container_cmd)

    local run_args=()

    # Set run mode
    case "$run_mode" in
        "interactive")
            if [[ -t 0 ]] && [[ -t 1 ]]; then
                run_args+=("-it")
            fi
            run_args+=("--rm")
            if [[ -n "$container_name" ]]; then
                run_args+=("--name" "$container_name")
            fi
            run_args+=("--init")
            ;;
        "detached")
            run_args+=("-d")
            if [[ -n "$container_name" ]]; then
                run_args+=("--name" "$container_name")
            fi
            ;;
        "pipe")
            run_args+=("--rm" "--init")
            ;;
    esac

    # Podman-specific: use keep-id for rootless uid mapping
    if [[ "$cmd" == "podman" ]]; then
        run_args+=("--userns=keep-id")
        run_args+=("--security-opt=no-new-privileges")
    fi

    # Network Setup: agents are attached only to the internal network.
    ensure_networks
    run_args+=("--network" "$AGENTBOX_INTERNAL_NETWORK")
    
    # Firewall / Proxy Setup
    if [[ "${AGENTBOX_NO_FIREWALL:-false}" != "true" ]]; then
        if ! start_squid_proxy; then
            error "Failed to start firewall (Squid proxy). Aborting for security."
        fi
        # Inject proxy vars
        # We read the output of get_proxy_env_vars which returns a string of flags
        local proxy_flags
        proxy_flags=$(get_proxy_env_vars)
        # We can just split it by space if we trust it doesn't contain spaces in values (urls don't).
        read -r -a proxy_flags_array <<< "$proxy_flags"
        run_args+=("${proxy_flags_array[@]}")
    fi

    # Get the image name for this agent and project
    local image_name
    image_name=$(get_agent_image_name "$agent")

    # Resource Limits (Sane defaults)
    run_args+=(
        "--memory=8g"
        "--cpus=4"
    )

    # Mount workspace
    local workspace_mount_mode=""
    if [[ "${CLI_READ_ONLY:-false}" == "true" ]]; then
        workspace_mount_mode=":ro"
    fi

    run_args+=(
        -w /workspace
        -v "${PROJECT_DIR}:/workspace${workspace_mount_mode}"
    )

    # Enforce non-root execution for all agent containers.
    run_args+=("--user" "$(id -u):$(id -g)")


    # Mount additional include directories inside /workspace
    if [[ ${#CLI_INCLUDE_DIRS[@]} -gt 0 ]]; then
        local include_src
        for include_src in "${CLI_INCLUDE_DIRS[@]}"; do
            if [[ -z "$include_src" ]]; then
                continue
            fi

            # Expand ~/
            if [[ "$include_src" == "~/"* ]]; then
                include_src="$HOME/${include_src#~/}"
            fi

            # Resolve relative paths against project dir
            if [[ "$include_src" != /* ]]; then
                include_src="$PROJECT_DIR/$include_src"
            fi

            # Trim trailing slash
            include_src="${include_src%/}"

            if [[ ! -d "$include_src" ]]; then
                warn "Include dir not found: $include_src"
                continue
            fi

            local rel=""
            if [[ "$include_src" == /tmp/* ]]; then
                rel="${include_src#/tmp/}"
            elif [[ "$include_src" == /tmp ]]; then
                warn "Skipping include dir /tmp (would map to /workspace)"
                continue
            else
                rel="${include_src#/}"
            fi

            if [[ -z "$rel" ]]; then
                warn "Skipping include dir $include_src (invalid target)"
                continue
            fi

            local dest="/workspace/${rel}"
            run_args+=(-v "$include_src:$dest")
        done
    fi


    # Mount agent-specific config directories
    local agent_config_dir
    agent_config_dir=$(get_agent_config_dir "$agent")

    # Ensure config directories exist
    mkdir -p "$agent_config_dir"

    # Mount credentials based on agent type
    case "$agent" in
        claude)
            # Mount original ~/.claude directly if it exists, otherwise use agentbox config
            if [[ -d "$HOME/.claude" ]]; then
                run_args+=(-v "$HOME/.claude":/home/user/.claude)
            else
                mkdir -p "$agent_config_dir/.claude"
                run_args+=(-v "$agent_config_dir/.claude":/home/user/.claude)
            fi
            # Mount ~/.claude.json (global onboarding state) if it exists
            if [[ -f "$HOME/.claude.json" ]]; then
                run_args+=(-v "$HOME/.claude.json":/home/user/.claude.json)
            fi
            # Claude binary is installed in the container via the build process
            mkdir -p "$agent_config_dir/.config"
            run_args+=(-v "$agent_config_dir/.config":/home/user/.config)
            ;;
        codex)
            # Mount original ~/.codex directly if it exists
            if [[ -d "$HOME/.codex" ]]; then
                run_args+=(-v "$HOME/.codex":/home/user/.codex)
            else
                mkdir -p "$agent_config_dir/.codex"
                run_args+=(-v "$agent_config_dir/.codex":/home/user/.codex)
            fi
            mkdir -p "$agent_config_dir/.config/codex"
            run_args+=(-v "$agent_config_dir/.config/codex":/home/user/.config/codex)
            ;;
        opencode)
            # Mount original ~/.opencode directly if it exists
            if [[ -d "$HOME/.opencode" ]]; then
                if [[ ! -w "$HOME/.opencode" ]]; then
                    error "OpenCode config dir not writable: $HOME/.opencode (fix ownership, e.g. sudo chown -R $USER:$USER \"$HOME/.opencode\")"
                fi
                run_args+=(-v "$HOME/.opencode":/home/user/.opencode)
            else
                mkdir -p "$agent_config_dir/.opencode"
                if [[ ! -w "$agent_config_dir/.opencode" ]]; then
                    error "OpenCode config dir not writable: $agent_config_dir/.opencode (fix ownership, e.g. sudo chown -R $USER:$USER \"$agent_config_dir/.opencode\")"
                fi
                run_args+=(-v "$agent_config_dir/.opencode":/home/user/.opencode)
            fi
            # Persist config on host, but only seed opencode.json if present.
            mkdir -p "$agent_config_dir/.config/opencode"
            if [[ ! -w "$agent_config_dir/.config/opencode" ]]; then
                error "OpenCode config dir not writable: $agent_config_dir/.config/opencode (fix ownership, e.g. sudo chown -R $USER:$USER \"$agent_config_dir/.config/opencode\")"
            fi
            if [[ -f "$HOME/.config/opencode/opencode.json" ]] && [[ ! -f "$agent_config_dir/.config/opencode/opencode.json" ]]; then
                cp "$HOME/.config/opencode/opencode.json" "$agent_config_dir/.config/opencode/opencode.json"
            fi
            run_args+=(-v "$agent_config_dir/.config/opencode":/home/user/.config/opencode)
            # Mount local share for OpenCode auth/state
            local opencode_share_dir="$HOME/.local/share/opencode"
            mkdir -p "$opencode_share_dir"
            if [[ ! -w "$opencode_share_dir" ]]; then
                error "OpenCode share dir not writable: $opencode_share_dir (fix ownership, e.g. sudo chown -R $USER:$USER \"$opencode_share_dir\")"
            fi
            run_args+=(-v "$opencode_share_dir":/home/user/.local/share/opencode)
            # Mount local state for OpenCode (prevents root-owned /home/user/.local)
            local opencode_state_dir="$agent_config_dir/.local/state"
            mkdir -p "$opencode_state_dir"
            if [[ ! -w "$opencode_state_dir" ]]; then
                error "OpenCode state dir not writable: $opencode_state_dir (fix ownership, e.g. sudo chown -R $USER:$USER \"$opencode_state_dir\")"
            fi
            run_args+=(-v "$opencode_state_dir":/home/user/.local/state)
            ;;
    esac

    # SECURITY FIX: REMOVED DEFAULT SSH MOUNT
    # Previous code mounted ~/.ssh read-only. This is unsafe.
    # Users must provide secrets explicitly via other means if needed.

    # Mount tmux socket if available
    if [[ -n "${TMUX:-}" ]]; then
        local tmux_socket="${TMUX%%,*}"
        local tmux_socket_dir
        tmux_socket_dir=$(dirname "$tmux_socket")
        if [[ -d "$tmux_socket_dir" ]]; then
            run_args+=(-v "$tmux_socket_dir:$tmux_socket_dir")
            run_args+=(-e "TMUX=$TMUX")
        fi
    fi

    # Add environment variables
    local project_name
    project_name=$(basename "$PROJECT_DIR")

    run_args+=(
        -e "NODE_ENV=${NODE_ENV:-production}"
        -e "TERM=${TERM:-xterm-256color}"
        -e "VERBOSE=${VERBOSE:-false}"
        -e "AGENTBOX_AGENT=$agent"
        -e "AGENTBOX_PROJECT_NAME=$project_name"
    )

    # Security options
    run_args+=(
        --security-opt=no-new-privileges:true
        --cap-drop=ALL
    )

    # Add image name
    run_args+=("$image_name")

    # Add container arguments
    if [[ ${#container_args[@]} -gt 0 ]]; then
        run_args+=("${container_args[@]}")
    fi

    # Run the container
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        printf '[DEBUG] Container run: %s run %s\n' "$cmd" "${run_args[*]}" >&2
    fi

    # Always capture container exit code and continue to proxy cleanup even
    # when the agent exits non-zero (e.g., SIGINT/SIGTERM/user abort).
    local exit_code=0
    set +e
    $cmd run "${run_args[@]}"
    exit_code=$?
    set -e

    cleanup_squid_if_unused

    return $exit_code
}

# Check if a container exists
check_container_exists() {
    local container_name="$1"
    local cmd
    cmd=$(container_cmd)

    if $cmd ps -a --filter "name=^${container_name}$" --format "{{.Names}}" | grep -q "^${container_name}$"; then
        if $cmd ps --filter "name=^${container_name}$" --format "{{.Names}}" | grep -q "^${container_name}$"; then
            printf 'running'
        else
            printf 'stopped'
        fi
    else
        printf 'none'
    fi
}

# Clean up container resources
clean_container_resources() {
    local mode="${1:-unused}"
    local cmd
    cmd=$(container_cmd)

    case "$mode" in
        unused)
            info "Removing unused agentbox images..."
            $cmd images --filter "reference=agentbox-*" --format "{{.Repository}}:{{.Tag}}" | \
                while read -r image; do
                    if ! $cmd ps -a --filter "ancestor=$image" --format "{{.ID}}" | grep -q .; then
                        $cmd rmi "$image" 2>/dev/null || true
                    fi
                done

            # Podman-specific: prune build cache
            if [[ "$cmd" == "podman" ]]; then
                $cmd system prune -f 2>/dev/null || true
            fi
            ;;
        all)
            info "Removing all agentbox images..."
            $cmd images --filter "reference=agentbox-*" --format "{{.Repository}}:{{.Tag}}" | \
                xargs -r $cmd rmi -f 2>/dev/null || true
            ;;
        containers)
            info "Stopping all agentbox containers..."
            $cmd ps --filter "name=agentbox-*" --format "{{.ID}}" | \
                xargs -r $cmd stop 2>/dev/null || true
            cleanup_squid_if_unused
            ;;
    esac
}

# Alias for backwards compatibility
clean_docker_resources() {
    clean_container_resources "$@"
}


export -f count_running_agent_containers cleanup_squid_if_unused
export -f run_agent_container check_container_exists clean_container_resources clean_docker_resources
