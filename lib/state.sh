#!/usr/bin/env bash
# State management - Symlink and installation state
# ============================================================================

# ============================================================================
# SYMLINK MANAGEMENT
# ============================================================================

# Update the agentbox symlink in ~/.local/bin
update_symlink() {
    local bin_dir="$HOME/.local/bin"
    local link_path="$bin_dir/agentbox"
    local script_path="${SCRIPT_DIR}/main.sh"

    # Ensure bin directory exists
    mkdir -p "$bin_dir"

    # Check if symlink needs updating
    if [[ -L "$link_path" ]]; then
        local current_target
        current_target=$(readlink "$link_path")
        if [[ "$current_target" == "$script_path" ]]; then
            return 0  # Already correct
        fi
        rm -f "$link_path"
    elif [[ -e "$link_path" ]]; then
        warn "Removing non-symlink at $link_path"
        rm -f "$link_path"
    fi

    # Create symlink
    ln -s "$script_path" "$link_path"

    # Check if bin_dir is in PATH
    if [[ ":$PATH:" != *":$bin_dir:"* ]]; then
        warn "$bin_dir is not in your PATH"
        info "Add this to your shell config:"
        printf '  export PATH="$HOME/.local/bin:$PATH"\n'
    fi
}

# ============================================================================
# INSTALLATION STATE
# ============================================================================

# Check if this is a first-time installation
is_first_install() {
    if [[ ! -f "$AGENTBOX_HOME/.installed" ]]; then
        return 0
    fi
    return 1
}

# Mark installation as complete
mark_installed() {
    mkdir -p "$AGENTBOX_HOME"
    touch "$AGENTBOX_HOME/.installed"
}

# Get the installed version
get_installed_version() {
    local version_file="$AGENTBOX_HOME/.version"
    if [[ -f "$version_file" ]]; then
        cat "$version_file"
    else
        printf ''
    fi
}

# Save the current version
save_version() {
    local version="${1:-$AGENTBOX_VERSION}"
    mkdir -p "$AGENTBOX_HOME"
    printf '%s\n' "$version" > "$AGENTBOX_HOME/.version"
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f update_symlink
export -f is_first_install mark_installed
export -f get_installed_version save_version
