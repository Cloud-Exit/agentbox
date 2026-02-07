#!/usr/bin/env bash
# Project management - Per-project isolation without slots
# ============================================================================
# Simplified from claudebox: one container per agent per project (no slots)

# ============================================================================
# CRC32 FUNCTIONS
# ============================================================================

# Compute CRC-32 of a string
crc32_string() {
    printf '%s' "$1" | cksum | cut -d' ' -f1
}

# Compute CRC-32 of a file
crc32_file() {
    if [[ -f "$1" ]]; then
        cksum "$1" | cut -d' ' -f1
    else
        printf '0'
    fi
}

# ============================================================================
# PROJECT PATH UTILITIES
# ============================================================================

# Slugify a filesystem path for use in names
slugify_path() {
    local path="${1#/}"
    path="${path//\//_}"
    # Remove unsafe characters, convert to lowercase
    printf '%s' "${path//[^a-zA-Z0-9_]/_}" | tr '[:upper:]' '[:lower:]'
}

# Generate the project folder name (slug + hash)
generate_project_folder_name() {
    local path="$1"
    local slug
    slug=$(slugify_path "$path")
    local hash
    hash=$(crc32_string "$path")
    printf '%s_%08x' "$slug" "$hash"
}

# Get the parent directory for a project
get_project_parent_dir() {
    local path="${1:-$PROJECT_DIR}"
    printf '%s/projects/%s' "$AGENTBOX_HOME" "$(generate_project_folder_name "$path")"
}

# ============================================================================
# PROJECT INITIALIZATION
# ============================================================================

# Initialize the project directory structure
init_project_dir() {
    local path="${1:-$PROJECT_DIR}"
    local parent
    parent=$(get_project_parent_dir "$path")

    mkdir -p "$parent"

    # Store original project path
    printf '%s\n' "$path" > "$parent/.project_path"

    # Create per-agent directories
    local agent
    for agent in claude codex opencode; do
        mkdir -p "$parent/$agent"
    done

    # Create local .agentbox directory in project
    mkdir -p "$path/.agentbox"
}

# ============================================================================
# IMAGE NAMING
# ============================================================================

# Get Docker image name for an agent in a project
get_image_name() {
    local agent="${1:-$AGENTBOX_CURRENT_AGENT}"
    local path="${2:-$PROJECT_DIR}"
    local project_folder
    project_folder=$(generate_project_folder_name "$path")
    printf 'agentbox-%s-%s' "$agent" "$project_folder"
}

# Get container name for an agent in a project
get_container_name() {
    local agent="${1:-$AGENTBOX_CURRENT_AGENT}"
    local path="${2:-$PROJECT_DIR}"
    local project_folder
    project_folder=$(generate_project_folder_name "$path")
    printf 'agentbox-%s-%s' "$agent" "$project_folder"
}

# ============================================================================
# PROJECT LISTING
# ============================================================================

# List all known projects
list_all_projects() {
    local projects_dir="$AGENTBOX_HOME/projects"
    local projects_found=0
    local cmd
    cmd=$(container_cmd)

    if [[ ! -d "$projects_dir" ]]; then
        return 1
    fi

    printf '\n'
    cecho "Known Projects:" "$CYAN"
    printf '\n'
    printf '  %-40s %s\n' "PROJECT PATH" "AGENTS"
    printf '  %-40s %s\n' "────────────" "──────"

    for parent_dir in "$projects_dir"/*/; do
        if [[ ! -d "$parent_dir" ]]; then
            continue
        fi

        projects_found=1
        local project_path=""

        # Read stored project path
        if [[ -f "$parent_dir/.project_path" ]]; then
            project_path=$(cat "$parent_dir/.project_path")
        fi

        # Count agents with images
        local agents_with_images=""
        local parent_name
        parent_name=$(basename "$parent_dir")

        for agent in claude codex opencode; do
            local image_name="agentbox-${agent}-${parent_name}"
            if $cmd image inspect "$image_name" >/dev/null 2>&1; then
                if [[ -n "$agents_with_images" ]]; then
                    agents_with_images+=", "
                fi
                agents_with_images+="$agent"
            fi
        done

        if [[ -z "$agents_with_images" ]]; then
            agents_with_images="-"
        fi

        # Truncate path if too long
        if [[ ${#project_path} -gt 38 ]]; then
            project_path="...${project_path: -35}"
        fi

        printf '  %-40s %s\n' "$project_path" "$agents_with_images"
    done

    printf '\n'

    if [[ $projects_found -eq 0 ]]; then
        printf '  No projects found.\n\n'
        return 1
    fi

    return 0
}

# Get project by partial path or name
get_project_by_path() {
    local search="$1"
    local projects_dir="$AGENTBOX_HOME/projects"

    if [[ ! -d "$projects_dir" ]]; then
        return 1
    fi

    for parent_dir in "$projects_dir"/*/; do
        if [[ ! -d "$parent_dir" ]]; then
            continue
        fi

        if [[ -f "$parent_dir/.project_path" ]]; then
            local project_path
            project_path=$(cat "$parent_dir/.project_path")
            if [[ "$project_path" == *"$search"* ]]; then
                printf '%s' "$project_path"
                return 0
            fi
        fi
    done

    return 1
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f crc32_string crc32_file
export -f slugify_path generate_project_folder_name get_project_parent_dir
export -f init_project_dir
export -f get_image_name get_container_name
export -f list_all_projects get_project_by_path
