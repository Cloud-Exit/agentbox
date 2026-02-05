#!/usr/bin/env bash
# Common utilities - Colors, logging, and shared functions
# ============================================================================

# ============================================================================
# COLORS
# ============================================================================

# Define colors (check if terminal supports colors)
if [[ -t 1 ]] && [[ "${TERM:-}" != "dumb" ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[0;33m'
    readonly BLUE='\033[0;34m'
    readonly MAGENTA='\033[0;35m'
    readonly CYAN='\033[0;36m'
    readonly WHITE='\033[0;37m'
    readonly BOLD='\033[1m'
    readonly DIM='\033[2m'
    readonly NC='\033[0m'  # No Color
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly MAGENTA=''
    readonly CYAN=''
    readonly WHITE=''
    readonly BOLD=''
    readonly DIM=''
    readonly NC=''
fi

# ============================================================================
# LOGGING FUNCTIONS
# ============================================================================

# Print colored text
cecho() {
    local msg="$1"
    local color="${2:-$NC}"
    printf '%b%s%b\n' "$color" "$msg" "$NC"
}

# Info message (cyan)
info() {
    printf '%b[INFO]%b %s\n' "$CYAN" "$NC" "$1"
}

# Success message (green)
success() {
    printf '%b[OK]%b %s\n' "$GREEN" "$NC" "$1"
}

# Warning message (yellow)
warn() {
    printf '%b[WARN]%b %s\n' "$YELLOW" "$NC" "$1" >&2
}

# Error message and exit (red)
error() {
    printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$1" >&2
    exit 1
}

# Debug message (only if VERBOSE is set)
debug() {
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        printf '%b[DEBUG]%b %s\n' "$DIM" "$NC" "$1" >&2
    fi
}

# ============================================================================
# LOGO
# ============================================================================

# Print the AgentBox logo (small version)
logo_small() {
    printf '%b' "$CYAN"
    cat << 'EOF'
     _                    _   ____
    / \   __ _  ___ _ __ | |_| __ )  _____  __
   / _ \ / _` |/ _ \ '_ \| __|  _ \ / _ \ \/ /
  / ___ \ (_| |  __/ | | | |_| |_) | (_) >  <
 /_/   \_\__, |\___|_| |_|\__|____/ \___/_/\_\
         |___/
EOF
    printf '%b' "$NC"
    printf '%b%s%b\n' "$DIM" "         by Cloud Exit (https://cloud-exit.com)" "$NC"
}

# Print the AgentBox logo (full version with tagline)
logo() {
    logo_small
    printf '\n'
    printf '%b%s%b\n' "$DIM" "Multi-Agent Container Sandbox" "$NC"
}

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Generate a hash from a string (CRC32-based, portable)
generate_hash() {
    local input="$1"
    local length="${2:-8}"

    # Use cksum for portable CRC32
    local hash
    hash=$(printf '%s' "$input" | cksum | cut -d' ' -f1)

    # Convert to hex and truncate
    printf '%x' "$hash" | head -c "$length"
}

# Generate a project folder name from a path
generate_parent_folder_name() {
    local project_dir="$1"
    local parent_name

    parent_name=$(basename "$project_dir")
    printf '%s' "$parent_name"
}

# Get real path (portable, works without readlink -f)
get_real_path() {
    local path="$1"

    if [[ -d "$path" ]]; then
        (cd "$path" && pwd)
    elif [[ -f "$path" ]]; then
        local dir
        dir=$(dirname "$path")
        local file
        file=$(basename "$path")
        (cd "$dir" && printf '%s/%s' "$(pwd)" "$file")
    else
        printf '%s' "$path"
    fi
}

# ============================================================================
# PROFILE UTILITIES
# ============================================================================

# Read a section from an INI file
read_profile_section() {
    local file="$1"
    local section="$2"
    local in_section=false

    if [[ ! -f "$file" ]]; then
        return 1
    fi

    while IFS= read -r line; do
        # Skip empty lines and comments
        if [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*# ]]; then
            continue
        fi

        # Check for section header
        if [[ "$line" =~ ^\[([^]]+)\]$ ]]; then
            if [[ "${BASH_REMATCH[1]}" == "$section" ]]; then
                in_section=true
            else
                in_section=false
            fi
            continue
        fi

        # Output lines in the target section
        if [[ "$in_section" == true ]]; then
            printf '%s\n' "$line"
        fi
    done < "$file"
}

# Get profile description
get_profile_description() {
    local profile="$1"

    case "$profile" in
        base)        printf 'Base development tools' ;;
        node)        printf 'Node.js runtime' ;;
        python)      printf 'Python runtime' ;;
        rust)        printf 'Rust toolchain' ;;
        go)          printf 'Go runtime' ;;
        java)        printf 'Java/JDK' ;;
        ruby)        printf 'Ruby runtime' ;;
        php)         printf 'PHP runtime' ;;
        dotnet)      printf '.NET SDK' ;;
        c)           printf 'C/C++ toolchain' ;;
        flutter)     printf 'Flutter SDK' ;;
        android)     printf 'Android SDK' ;;
        docker)      printf 'Docker-in-Docker' ;;
        kubernetes)  printf 'Kubernetes tools' ;;
        terraform)   printf 'Terraform' ;;
        aws)         printf 'AWS CLI' ;;
        gcloud)      printf 'Google Cloud SDK' ;;
        azure)       printf 'Azure CLI' ;;
        *)           printf 'Unknown profile' ;;
    esac
}

# ============================================================================
# EXPORTS
# ============================================================================

export RED GREEN YELLOW BLUE MAGENTA CYAN WHITE BOLD DIM NC
export -f cecho info success warn error debug
export -f logo logo_small
export -f command_exists generate_hash generate_parent_folder_name get_real_path
export -f read_profile_section get_profile_description
