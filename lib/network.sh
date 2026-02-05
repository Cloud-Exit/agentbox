#!/usr/bin/env bash
# Network management - Firewalling and Connectivity
# ============================================================================

# Network name for agentbox
readonly AGENTBOX_NETWORK="agentbox-net"
readonly AGENTBOX_SQUID_CONTAINER="agentbox-squid"

# Get the allowlist file path
get_allowlist_path() {
    local config_allowlist="${AGENTBOX_HOME}/allowlist.txt"
    local default_allowlist="${SCRIPT_DIR}/config/allowlist.txt"

    # User config takes precedence
    if [[ -f "$config_allowlist" ]]; then
        printf '%s' "$config_allowlist"
    elif [[ -f "$default_allowlist" ]]; then
        printf '%s' "$default_allowlist"
    else
        printf ''
    fi
}

# Extract root domain (TLD+1) from a domain
extract_root_domain() {
    local domain="$1"
    
    # Strip protocol and path
    domain="${domain#*://}"
    domain="${domain%%/*}"
    # Strip port
    domain="${domain%%:*}"

    # If IP, return as is
    if [[ "$domain" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        printf '%s' "$domain"
        return 0
    fi

    # Handle known multi-part TLDs (naive list)
    # This prevents "google.co.uk" becoming just "co.uk"
    if [[ "$domain" =~ \.(co\.uk|com\.au|co\.jp|gov\.uk|ac\.uk|org\.uk|net\.au|org\.au)$ ]]; then
         # Extract last 3 parts
         local IFS='.'
         read -r -a parts <<< "$domain"
         local count="${#parts[@]}"
         if (( count >= 3 )); then
             printf '%s.%s.%s' "${parts[$((count - 3))]}" "${parts[$((count - 2))]}" "${parts[$((count - 1))]}"
             return 0
         fi
    fi

    # Default: TLD+1
    local IFS='.'
    read -r -a parts <<< "$domain"
    local count="${#parts[@]}"
    
    if (( count <= 2 )); then
        printf '%s' "$domain"
    else
        printf '%s.%s' "${parts[$((count - 2))]}" "${parts[$((count - 1))]}"
    fi
}

# Get the subnet for the agentbox network
get_network_subnet() {
    local cmd
    cmd=$(container_cmd)
    # Ensure network exists first
    ensure_network
    # Get IPv4 subnet
    $cmd network inspect "$AGENTBOX_NETWORK" --format '{{range .IPAM.Config}}{{if .Subnet}}{{.Subnet}}{{end}}{{end}}' | head -n 1
}

# Generate Squid configuration
generate_squid_config() {
    local allowlist_path
    allowlist_path=$(get_allowlist_path)
    local config_file="$AGENTBOX_CACHE/squid.conf"
    
    # Get network subnet for locking down access
    # local subnet
    # subnet=$(get_network_subnet)
    
    # if [[ -z "$subnet" ]]; then
    #     # Fallback if subnet detection fails (shouldn't happen if network exists)
    #     subnet="0.0.0.0/0" 
    #     warn "Could not detect agentbox network subnet. Squid access will be open to all containers."
    # fi

    mkdir -p "$(dirname "$config_file")"

    cat > "$config_file" << EOF
# Squid Configuration for Agentbox
http_port 3128
shutdown_lifetime 1 seconds

# Access Control Lists
acl SSL_ports port 443
acl Safe_ports port 80		# http
acl Safe_ports port 21		# ftp
acl Safe_ports port 443		# https
acl Safe_ports port 70		# gopher
acl Safe_ports port 210		# wais
acl Safe_ports port 1025-65535	# unregistered ports
acl Safe_ports port 280		# http-mgmt
acl Safe_ports port 488		# gss-http
acl Safe_ports port 591		# filemaker
acl Safe_ports port 777		# multiling http
acl CONNECT method CONNECT

# Deny requests to certain unsafe ports
http_access deny !Safe_ports
http_access deny CONNECT !SSL_ports

# Localhost access
acl localhost src 127.0.0.1/32

# Allow all sources (security is handled by network isolation)
acl all_sources src all

# Allowlist
EOF

    if [[ -n "$allowlist_path" ]] && [[ -f "$allowlist_path" ]]; then
        local added_domains=""
        
        # Process allowlist
        while IFS= read -r line; do
            # Skip empty lines and comments
            if [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*# ]]; then
                continue
            fi

            local domain
            domain=$(printf '%s' "$line" | xargs) # Trim
            
            if [[ -n "$domain" ]]; then
                local root_domain
                root_domain=$(extract_root_domain "$domain")
                
                # Deduplicate
                if [[ " $added_domains " == *" $root_domain "* ]]; then
                    continue
                fi
                added_domains="$added_domains $root_domain"
                
                # Add dot for subdomain matching (Squid dstdomain)
                # If it's an IP, don't add dot.
                if [[ ! "$root_domain" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                     root_domain=".$root_domain"
                fi
                
                printf 'acl allowed_domains dstdomain %s\n' "$root_domain" >> "$config_file"
            fi
        done < "$allowlist_path"
    else
        # If no allowlist, allow everything
        printf 'acl allowed_domains src 0.0.0.0/0\n' >> "$config_file"
    fi

    cat >> "$config_file" << 'EOF'

# Enforce Access Control
# Only allow access from localhost and our network
http_access allow localhost
http_access allow all_sources allowed_domains

# Deny everything else
http_access deny all

# Hide proxy info
forwarded_for off
via off
EOF

    echo "$config_file"
}

# Ensure the shared network exists
ensure_network() {
    local cmd
    cmd=$(container_cmd)
    
    if ! $cmd network ls --format '{{.Name}}' | grep -q "^${AGENTBOX_NETWORK}$"; then
        info "Creating network $AGENTBOX_NETWORK..."
        $cmd network create "$AGENTBOX_NETWORK" >/dev/null
    fi
}

# Start the Squid proxy container
start_squid_proxy() {
    local cmd
    cmd=$(container_cmd)
    
    # Check if running
    if $cmd ps --filter "name=^/${AGENTBOX_SQUID_CONTAINER}$" --format '{{.Names}}' | grep -q "^${AGENTBOX_SQUID_CONTAINER}$"; then
        return 0
    fi
    
    # Remove if stopped
    $cmd rm -f "$AGENTBOX_SQUID_CONTAINER" >/dev/null 2>&1 || true

    # Ensure network exists first (needed for subnet detection)
    ensure_network

    # Build image if needed
    build_squid_image

    # Generate config
    local config_file
    config_file=$(generate_squid_config)
    
    info "Starting Squid proxy..."
    $cmd run -d \
        --name "$AGENTBOX_SQUID_CONTAINER" \
        --network "$AGENTBOX_NETWORK" \
        -v "$config_file":/etc/squid/squid.conf:ro \
        --restart=unless-stopped \
        agentbox-squid >/dev/null

    # Wait briefly for IP allocation
    local retries=10
    while [[ $retries -gt 0 ]]; do
        if $cmd inspect "$AGENTBOX_SQUID_CONTAINER" --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null | grep -q .; then
             break
        fi
        sleep 0.1
        ((retries--))
    done
}

# Get proxy environment variables
get_proxy_env_vars() {
    if [[ "${AGENTBOX_NO_FIREWALL:-false}" == "true" ]]; then
        return 0
    fi
    
    # We assume the proxy is reachable via hostname 'agentbox-squid' on the network
    # However, to be robust against DNS issues in rootless containers, we try to resolve IP first
    local proxy_host="$AGENTBOX_SQUID_CONTAINER"
    local cmd
    cmd=$(container_cmd)
    
    # Try to get IP address of squid container on agentbox network
    local squid_ip
    squid_ip=$($cmd inspect "$AGENTBOX_SQUID_CONTAINER" --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null | head -1)

    if [[ -n "$squid_ip" ]]; then
        proxy_host="$squid_ip"
    fi
    
    local proxy_url="http://${proxy_host}:3128"
    
    printf -- '-e http_proxy=%s ' "$proxy_url"
    printf -- '-e https_proxy=%s ' "$proxy_url"
    printf -- '-e HTTP_PROXY=%s ' "$proxy_url"
    printf -- '-e HTTPS_PROXY=%s ' "$proxy_url"
    # Important: No_proxy for local network and internal docker stuff
    printf -- '-e no_proxy=localhost,127.0.0.1,.local,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16 '
    printf -- '-e NO_PROXY=localhost,127.0.0.1,.local,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16 '
}

export -f ensure_network start_squid_proxy get_proxy_env_vars generate_squid_config extract_root_domain get_network_subnet
