#!/usr/bin/env bash
# Test script for Squid config generation

set -euo pipefail

# Source required libraries
source /workspace/lib/common.sh
source /workspace/lib/env.sh
source /workspace/lib/os.sh
source /workspace/lib/container.sh
source /workspace/lib/network.sh

tmp_cache="$(mktemp -d)"
trap 'rm -rf "$tmp_cache"' EXIT

# Mock variables (set after env.sh initialization).
export AGENTBOX_HOME="/workspace/config"
export AGENTBOX_CACHE="$tmp_cache"
export SCRIPT_DIR="/workspace"

# Create mock allowlist if not exists.
if [[ ! -f "$AGENTBOX_HOME/allowlist.txt" ]]; then
    mkdir -p "$AGENTBOX_HOME"
    cp /workspace/config/allowlist.txt "$AGENTBOX_HOME/allowlist.txt"
fi

# Avoid container-runtime dependency in this standalone test.
get_network_subnet() {
    printf '10.123.0.0/24'
}

# Run generation
config_file=$(generate_squid_config)

echo "Generated config at: $config_file"
echo "--------------------------------"
cat "$config_file"

grep -q '^acl agent_sources src 10\.123\.0\.0/24$' "$config_file"
grep -q '^http_access deny all$' "$config_file"

echo "Squid config generation test passed."
