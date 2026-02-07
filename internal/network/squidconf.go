// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"fmt"
	"strings"

	"github.com/cloud-exit/exitbox/internal/ui"
)

// GenerateSquidConfig generates the squid.conf content.
func GenerateSquidConfig(subnet string, domains []string, extraURLs []string) string {
	var b strings.Builder

	b.WriteString(`# Squid Configuration for Agentbox
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

# Only allow proxy clients from the internal agent network
`)
	fmt.Fprintf(&b, "acl agent_sources src %s\n\n# Allowlist\n", subnet)

	seen := make(map[string]bool)
	count := 0

	for _, domain := range domains {
		normalized, err := NormalizeAllowlistEntry(domain)
		if err != nil {
			ui.Warnf("Skipping invalid allowlist entry: %s", domain)
			continue
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		fmt.Fprintf(&b, "acl allowed_domains dstdomain %s\n", normalized)
		count++
	}

	// Extra URLs
	for _, url := range extraURLs {
		if url == "" {
			continue
		}
		normalized, err := NormalizeAllowlistEntry(url)
		if err != nil {
			ui.Warnf("Skipping invalid --allow-urls entry: %s", url)
			continue
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		fmt.Fprintf(&b, "acl allowed_domains dstdomain %s\n", normalized)
		count++
	}

	if count == 0 {
		ui.Warn("Allowlist is empty or invalid. Blocking all outbound destinations.")
		b.WriteString("acl allowed_domains dstdomain .__agentbox_block_all__.invalid\n")
	}

	b.WriteString(`
# Enforce Access Control
# Only allow access from localhost and our network
http_access allow localhost
http_access allow agent_sources allowed_domains

# Deny everything else
http_access deny all

# Hide proxy info
forwarded_for off
via off
`)

	return b.String()
}
