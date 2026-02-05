#!/usr/bin/env bash
# OS detection and platform-specific utilities
# ============================================================================

# ============================================================================
# PLATFORM DETECTION
# ============================================================================

# Detect the current operating system
detect_os() {
    local os
    os=$(uname -s)

    case "$os" in
        Darwin)
            printf 'macos'
            ;;
        Linux)
            printf 'linux'
            ;;
        MINGW*|MSYS*|CYGWIN*)
            printf 'windows'
            ;;
        *)
            printf 'unknown'
            ;;
    esac
}

# Detect the current architecture
detect_arch() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64)
            printf 'x86_64'
            ;;
        aarch64|arm64)
            printf 'arm64'
            ;;
        armv7l)
            printf 'armv7'
            ;;
        *)
            printf '%s' "$arch"
            ;;
    esac
}

# Get the current platform string (os-arch)
get_platform() {
    local os
    local arch

    os=$(detect_os)
    arch=$(detect_arch)

    printf '%s-%s' "$os" "$arch"
}

# ============================================================================
# PLATFORM-SPECIFIC UTILITIES
# ============================================================================

# Get the stat command for file modification time
# macOS uses -f, Linux uses -c
get_file_mtime() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        printf '0'
        return 1
    fi

    local os
    os=$(detect_os)

    case "$os" in
        macos)
            stat -f %m "$file" 2>/dev/null || printf '0'
            ;;
        linux)
            stat -c %Y "$file" 2>/dev/null || printf '0'
            ;;
        *)
            printf '0'
            ;;
    esac
}

# Get the sed in-place flag (macOS requires empty string backup)
get_sed_inplace() {
    local os
    os=$(detect_os)

    case "$os" in
        macos)
            printf '%s' "-i ''"
            ;;
        *)
            printf '%s' "-i"
            ;;
    esac
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f detect_os detect_arch get_platform
export -f get_file_mtime get_sed_inplace
