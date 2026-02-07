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
	"net"
	"regexp"
	"strings"
)

var hostnameRE = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)+$`)
var ipv4RE = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`)
var ipv6RE = regexp.MustCompile(`^[0-9A-Fa-f:]+$`)

// NormalizeAllowlistEntry normalizes a domain for Squid dstdomain ACL.
// Ported from lib/network.sh:normalize_allowlist_entry.
func NormalizeAllowlistEntry(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty entry")
	}

	// Strip protocol and path
	if idx := strings.Index(value, "://"); idx >= 0 {
		value = value[idx+3:]
	}
	if idx := strings.Index(value, "/"); idx >= 0 {
		value = value[:idx]
	}

	// Handle wildcard subdomains
	if strings.HasPrefix(value, "*.") {
		value = value[2:]
	} else if strings.HasPrefix(value, ".") {
		value = value[1:]
	}

	// Handle bracketed IPv6 literals like [2001:db8::1] or [2001:db8::1]:443
	if strings.HasPrefix(value, "[") {
		end := strings.Index(value, "]")
		if end >= 0 {
			inner := value[1:end]
			if net.ParseIP(inner) != nil {
				return strings.ToLower(inner), nil
			}
		}
	}

	// Strip trailing DNS dot
	value = strings.TrimSuffix(value, ".")

	// Detect IPv6 literals
	isIPv6 := strings.Contains(value, ":") && ipv6RE.MatchString(value)

	// Strip port from hostnames only (not IPv4 or IPv6)
	if strings.Contains(value, ":") && !ipv4RE.MatchString(value) && !isIPv6 {
		value = value[:strings.Index(value, ":")]
	}

	// IPv4
	if ipv4RE.MatchString(value) {
		return value, nil
	}

	// localhost
	if value == "localhost" {
		return value, nil
	}

	// IPv6
	if isIPv6 {
		return strings.ToLower(value), nil
	}

	// Hostname validation
	if !hostnameRE.MatchString(value) {
		return "", fmt.Errorf("invalid entry: %s", value)
	}

	// Squid dstdomain with leading dot
	return "." + strings.ToLower(value), nil
}
