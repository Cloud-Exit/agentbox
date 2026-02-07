#!/usr/bin/env bash
# Basic profile integrity tests.

set -euo pipefail

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

export HOME="$tmp_root/home"
export XDG_CONFIG_HOME="$tmp_root/config"
export XDG_CACHE_HOME="$tmp_root/cache"
export XDG_DATA_HOME="$tmp_root/data"
export PROJECT_DIR="$tmp_root/project"
mkdir -p "$HOME" "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME" "$XDG_DATA_HOME" "$PROJECT_DIR"

source /workspace/lib/common.sh
source /workspace/lib/env.sh
source /workspace/lib/config.sh

init_agentbox_config
init_project_config "$PROJECT_DIR"

# Every listed profile must have an installer function.
while IFS= read -r entry; do
    [[ -z "$entry" ]] && continue
    profile_name="${entry%%:*}"
    profile_fn="get_profile_${profile_name//-/_}"
    if ! declare -F "$profile_fn" >/dev/null 2>&1; then
        echo "Missing profile installer function: $profile_fn" >&2
        exit 1
    fi
done < <(list_available_profiles)

# Unknown profiles must be rejected.
if (add_project_profile "claude" "does-not-exist" >/dev/null 2>&1); then
    echo "add_project_profile accepted an unknown profile" >&2
    exit 1
fi

# Known profiles should be written once (no duplicates).
add_project_profile "claude" "python"
add_project_profile "claude" "python"
profiles_file="${PROJECT_DIR}/.agentbox/claude/profiles.ini"

if [[ "$(grep -Fxc -- 'python' "$profiles_file" || true)" != "1" ]]; then
    echo "Expected exactly one 'python' entry in $profiles_file" >&2
    exit 1
fi

echo "Profile integrity tests passed."
