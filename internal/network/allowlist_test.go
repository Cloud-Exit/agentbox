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

import "testing"

func TestNormalizeAllowlistEntry(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		// Regular hostnames get leading dot
		{"example.com", ".example.com", false},
		{"api.example.com", ".api.example.com", false},

		// Case normalization
		{"Example.COM", ".example.com", false},

		// Wildcard stripping
		{"*.example.com", ".example.com", false},
		{".example.com", ".example.com", false},

		// Protocol stripping
		{"https://example.com", ".example.com", false},
		{"http://example.com/path", ".example.com", false},

		// Port stripping
		{"example.com:443", ".example.com", false},

		// Trailing dot
		{"example.com.", ".example.com", false},

		// IPv4 - no leading dot
		{"1.2.3.4", "1.2.3.4", false},
		{"127.0.0.1", "127.0.0.1", false},

		// localhost - no leading dot
		{"localhost", "localhost", false},

		// IPv6
		{"2606:4700:4700::1111", "2606:4700:4700::1111", false},
		{"[2001:db8::1]", "2001:db8::1", false},
		{"[2001:db8::1]:443", "2001:db8::1", false},

		// Invalid entries
		{"", "", true},
		{"not valid!", "", true},
	}

	for _, tc := range tests {
		got, err := NormalizeAllowlistEntry(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("NormalizeAllowlistEntry(%q) = %q, want error", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeAllowlistEntry(%q) error = %v", tc.input, err)
			continue
		}
		if got != tc.expected {
			t.Errorf("NormalizeAllowlistEntry(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
