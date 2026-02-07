#!/usr/bin/env bash
# Environment configuration - Paths and environment variables
# ============================================================================

# ============================================================================
# SCRIPT PATHS
# ============================================================================

# Get the real path of the script (follows symlinks)
_get_script_dir() {
    local source="${BASH_SOURCE[0]}"
    local dir

    # Resolve symlinks
    while [[ -L "$source" ]]; do
        dir=$(cd -P "$(dirname "$source")" && pwd)
        source=$(readlink "$source")
        # Handle relative symlinks
        if [[ "$source" != /* ]]; then
            source="$dir/$source"
        fi
    done

    dir=$(cd -P "$(dirname "$source")" && pwd)
    printf '%s' "$dir"
}

# Script locations (only set if not already defined)
if [[ -z "${LIB_DIR:-}" ]]; then
    LIB_DIR="$(_get_script_dir)"
fi
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(dirname "$LIB_DIR")"
fi
BUILD_DIR="${SCRIPT_DIR}/build"
TEMPLATES_DIR="${SCRIPT_DIR}/templates"

# ============================================================================
# AGENTBOX PATHS (XDG-compliant)
# ============================================================================

# Config home (~/.config/agentbox)
AGENTBOX_HOME="${XDG_CONFIG_HOME:-$HOME/.config}/agentbox"

# Cache home (~/.cache/agentbox)
AGENTBOX_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/agentbox"

# Data home (~/.local/share/agentbox)
AGENTBOX_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/agentbox"

# Symlink location for CLI
LINK_TARGET="${HOME}/.local/bin/agentbox"

# ============================================================================
# PROJECT PATHS
# ============================================================================

# Current working directory (project root)
if [[ -z "${PROJECT_DIR:-}" ]]; then
    PROJECT_DIR="$(pwd)"
fi

# Per-project agentbox config directory
PROJECT_AGENTBOX_DIR="${PROJECT_DIR}/.agentbox"

# ============================================================================
# DOCKER CONFIGURATION
# ============================================================================

# Docker image prefix
DOCKER_IMAGE_PREFIX="agentbox"

# Base image name
DOCKER_BASE_IMAGE="${DOCKER_IMAGE_PREFIX}-base"

# Container username inside Docker (must match USERNAME build arg in Dockerfile.base)
CONTAINER_USER="user"
CONTAINER_UID=1000
CONTAINER_GID=1000

# ============================================================================
# VERSION
# ============================================================================

if [[ -z "${AGENTBOX_VERSION:-}" ]]; then
    AGENTBOX_VERSION="3.2.0"
fi

# ============================================================================
# EXPORTS
# ============================================================================

export LIB_DIR SCRIPT_DIR BUILD_DIR TEMPLATES_DIR
export AGENTBOX_HOME AGENTBOX_CACHE AGENTBOX_DATA LINK_TARGET
export PROJECT_DIR PROJECT_AGENTBOX_DIR
export DOCKER_IMAGE_PREFIX DOCKER_BASE_IMAGE
export CONTAINER_USER CONTAINER_UID CONTAINER_GID
export AGENTBOX_VERSION
