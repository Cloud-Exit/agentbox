#!/bin/bash
# Test script for Squid config generation

# Mock variables
export AGENTBOX_HOME="/workspace/config"
export AGENTBOX_CACHE="/workspace/cache"
export SCRIPT_DIR="/workspace"

# Create mock allowlist if not exists (using the one I just wrote)
if [[ ! -f "$AGENTBOX_HOME/allowlist.txt" ]]; then
    mkdir -p "$AGENTBOX_HOME"
    cp /workspace/config/allowlist.txt "$AGENTBOX_HOME/allowlist.txt"
fi

# Source network library
source /workspace/lib/network.sh

# Run generation
config_file=$(generate_squid_config)

echo "Generated config at: $config_file"
echo "--------------------------------"
cat "$config_file"
