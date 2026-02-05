#!/usr/bin/env bash
# Container runtime abstraction - Podman-first with Docker fallback
# ============================================================================
# Optimized for Podman (daemonless, rootless by default)

# ============================================================================
# RUNTIME DETECTION
# ============================================================================

# Detected container runtime (podman or docker)
CONTAINER_RUNTIME=""
CONTAINER_RUNTIME_VERSION=""

# Detect available container runtime (prefer Podman)
detect_container_runtime() {
    if [[ -n "$CONTAINER_RUNTIME" ]]; then
        return 0
    fi

    # Prefer Podman
    if command -v podman >/dev/null 2>&1; then
        CONTAINER_RUNTIME="podman"
        CONTAINER_RUNTIME_VERSION=$(podman --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || printf 'unknown')
        export CONTAINER_RUNTIME CONTAINER_RUNTIME_VERSION
        return 0
    fi

    # Fall back to Docker
    if command -v docker >/dev/null 2>&1; then
        CONTAINER_RUNTIME="docker"
        CONTAINER_RUNTIME_VERSION=$(docker --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || printf 'unknown')
        export CONTAINER_RUNTIME CONTAINER_RUNTIME_VERSION
        return 0
    fi

    return 1
}

# Get the container command (podman or docker)
container_cmd() {
    detect_container_runtime
    printf '%s' "$CONTAINER_RUNTIME"
}

# ============================================================================
# RUNTIME CHECKS
# ============================================================================

# Check if container runtime is available and working
check_container_runtime() {
    if ! detect_container_runtime; then
        return 1  # No runtime found
    fi

    local cmd
    cmd=$(container_cmd)

    # For Podman, just check if it can run (daemonless)
    if [[ "$cmd" == "podman" ]]; then
        if ! $cmd info >/dev/null 2>&1; then
            return 2  # Podman not working
        fi
        return 0
    fi

    # For Docker, check daemon is running
    if [[ "$cmd" == "docker" ]]; then
        if ! $cmd info >/dev/null 2>&1; then
            return 2  # Docker daemon not running
        fi
        if ! $cmd ps >/dev/null 2>&1; then
            return 3  # Docker permission issue
        fi
        return 0
    fi

    return 1
}

# Alias for backwards compatibility
check_docker() {
    check_container_runtime
}

# Install container runtime
install_container_runtime() {
    warn "No container runtime found."
    cecho "Podman is recommended (daemonless, rootless by default)." "$CYAN"
    printf '\n'
    printf 'Install options:\n'
    printf '  1) Podman (recommended)\n'
    printf '  2) Docker\n'
    printf '\n'
    printf 'Choice [1]: '

    local choice
    read -r choice
    choice="${choice:-1}"

    case "$choice" in
        1)
            install_podman
            ;;
        2)
            install_docker
            ;;
        *)
            error "Invalid choice. Please install Podman or Docker manually."
            ;;
    esac
}

# Install Podman
install_podman() {
    info "Installing Podman..."

    if [[ -f /etc/os-release ]]; then
        # shellcheck source=/dev/null
        . /etc/os-release
    else
        error "Cannot detect OS"
    fi

    case "${ID:-}" in
        ubuntu)
            warn "Installing Podman requires sudo privileges..."
            sudo apt-get update
            sudo apt-get install -y podman
            ;;
        debian)
            warn "Installing Podman requires sudo privileges..."
            sudo apt-get update
            sudo apt-get install -y podman
            ;;
        fedora)
            warn "Installing Podman requires sudo privileges..."
            sudo dnf install -y podman
            ;;
        rhel|centos|rocky|almalinux)
            warn "Installing Podman requires sudo privileges..."
            sudo dnf install -y podman
            ;;
        arch|manjaro)
            warn "Installing Podman requires sudo privileges..."
            sudo pacman -S --noconfirm podman
            ;;
        opensuse*|sles)
            warn "Installing Podman requires sudo privileges..."
            sudo zypper install -y podman
            ;;
        *)
            error "Unsupported OS: ${ID:-unknown}. Please install Podman manually."
            ;;
    esac

    success "Podman installed successfully!"

    # Configure for rootless (usually automatic, but ensure subuid/subgid)
    configure_podman_rootless
}

# Configure Podman for rootless operation
configure_podman_rootless() {
    # Check if subuid/subgid are configured
    if ! grep -q "^${USER}:" /etc/subuid 2>/dev/null; then
        info "Configuring rootless Podman..."
        warn "This requires sudo to set up subuid/subgid..."

        # Add subuid/subgid entries
        sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 "$USER"

        # Migrate to rootless
        podman system migrate 2>/dev/null || true
    fi

    success "Podman rootless configured!"
}

# Install Docker (fallback)
install_docker() {
    warn "Installing Docker..."
    info "Note: Podman is recommended for rootless container support."

    if [[ -f /etc/os-release ]]; then
        # shellcheck source=/dev/null
        . /etc/os-release
    else
        error "Cannot detect OS"
    fi

    case "${ID:-}" in
        ubuntu|debian)
            warn "Installing Docker requires sudo privileges..."
            sudo apt-get update
            sudo apt-get install -y ca-certificates curl gnupg lsb-release
            sudo mkdir -p /etc/apt/keyrings
            curl -fsSL "https://download.docker.com/linux/${ID}/gpg" | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/%s %s stable\n' \
                "$(dpkg --print-architecture)" "$ID" "$(lsb_release -cs)" | \
                sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
            sudo apt-get update
            sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin
            ;;
        fedora)
            warn "Installing Docker requires sudo privileges..."
            sudo dnf -y install dnf-plugins-core
            sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
            sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin
            sudo systemctl start docker
            sudo systemctl enable docker
            ;;
        arch|manjaro)
            warn "Installing Docker requires sudo privileges..."
            sudo pacman -S --noconfirm docker
            sudo systemctl start docker
            sudo systemctl enable docker
            ;;
        *)
            error "Unsupported OS: ${ID:-unknown}. Please install Docker manually."
            ;;
    esac

    success "Docker installed!"
    configure_docker_nonroot
}

# Configure Docker for non-root usage
configure_docker_nonroot() {
    warn "Configuring Docker for non-root usage..."
    warn "This requires sudo to add you to the docker group..."

    getent group docker >/dev/null || sudo groupadd docker
    sudo usermod -aG docker "$USER"

    success "Docker configured for non-root usage!"
    warn "You need to log out and back in for group changes to take effect."
    warn "Or run: ${CYAN}newgrp docker"
    info "Trying to activate docker group in current shell..."
    exec newgrp docker
}

# ============================================================================
# PODMAN-SPECIFIC OPTIMIZATIONS
# ============================================================================

# Get Podman-specific build arguments
get_podman_build_args() {
    local args=()

    # Podman supports --layers for better caching
    args+=("--layers")

    # Use local storage for faster builds
    args+=("--pull=newer")

    # Squash layers for smaller images (optional)
    # args+=("--squash")

    printf '%s\n' "${args[@]}"
}

# Get runtime-specific run arguments
get_runtime_run_args() {
    local cmd
    cmd=$(container_cmd)
    local args=()

    if [[ "$cmd" == "podman" ]]; then
        # Podman-specific optimizations

        # Use host's user namespace for better rootless support
        args+=("--userns=keep-id")

        # Better security defaults
        args+=("--security-opt=no-new-privileges")

        # Use overlayfs for better performance
        # args+=("--storage-driver=overlay")
    fi

    printf '%s\n' "${args[@]}"
}

# Check if running in rootless mode (Podman)
is_rootless() {
    local cmd
    cmd=$(container_cmd)

    if [[ "$cmd" == "podman" ]]; then
        # Podman is rootless if not running as root
        if [[ "$(id -u)" != "0" ]]; then
            return 0
        fi
    fi

    return 1
}

# ============================================================================
# CONTAINER OPERATIONS
# ============================================================================

# Run a container
container_run() {
    local cmd
    cmd=$(container_cmd)

    # Get runtime-specific args
    local runtime_args=()
    while IFS= read -r arg; do
        if [[ -n "$arg" ]]; then
            runtime_args+=("$arg")
        fi
    done < <(get_runtime_run_args)

    $cmd run "${runtime_args[@]}" "$@"
}

# Build an image
container_build() {
    local cmd
    cmd=$(container_cmd)

    local build_args=()

    if [[ "$cmd" == "podman" ]]; then
        while IFS= read -r arg; do
            if [[ -n "$arg" ]]; then
                build_args+=("$arg")
            fi
        done < <(get_podman_build_args)
    fi

    $cmd build "${build_args[@]}" "$@"
}

# Check if image exists
container_image_exists() {
    local image="$1"
    local cmd
    cmd=$(container_cmd)

    $cmd image inspect "$image" >/dev/null 2>&1
}

# Remove image
container_image_rm() {
    local cmd
    cmd=$(container_cmd)
    $cmd rmi "$@"
}

# List images
container_images() {
    local cmd
    cmd=$(container_cmd)
    $cmd images "$@"
}

# Check container status
container_ps() {
    local cmd
    cmd=$(container_cmd)
    $cmd ps "$@"
}

# Stop container
container_stop() {
    local cmd
    cmd=$(container_cmd)
    $cmd stop "$@"
}

# Remove container
container_rm() {
    local cmd
    cmd=$(container_cmd)
    $cmd rm "$@"
}

# Execute in container
container_exec() {
    local cmd
    cmd=$(container_cmd)
    $cmd exec "$@"
}

# ============================================================================
# EXPORTS
# ============================================================================

export CONTAINER_RUNTIME CONTAINER_RUNTIME_VERSION
export -f detect_container_runtime container_cmd
export -f check_container_runtime install_container_runtime check_docker
export -f install_podman configure_podman_rootless
export -f install_docker configure_docker_nonroot
export -f get_podman_build_args get_runtime_run_args is_rootless
export -f container_run container_build container_image_exists
export -f container_image_rm container_images container_ps
export -f container_stop container_rm container_exec
