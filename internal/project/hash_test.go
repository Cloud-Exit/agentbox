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

package project

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPOSIXCksum(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		// These values can be verified with: printf '%s' "input" | cksum
		{"/home/user/projects/myapp", 0},   // Will be filled by reference
		{"/workspace", 0},                   // Will be filled by reference
		{"hello", 0},                        // Will be filled by reference
	}

	// Get reference values from system cksum
	for i, tc := range tests {
		out, err := exec.Command("sh", "-c", "printf '%s' '"+tc.input+"' | cksum | cut -d' ' -f1").Output()
		if err != nil {
			t.Fatalf("failed to get reference cksum for %q: %v", tc.input, err)
		}
		var ref uint32
		_, err = parseUint32(strings.TrimSpace(string(out)), &ref)
		if err != nil {
			t.Fatalf("failed to parse reference cksum for %q: %v", tc.input, err)
		}
		tests[i].expected = ref
	}

	for _, tc := range tests {
		got := POSIXCksumString(tc.input)
		if got != tc.expected {
			t.Errorf("POSIXCksumString(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestPOSIXCksumKnownValues(t *testing.T) {
	// Verify a few known values from POSIX cksum
	// printf '%s' "" | cksum -> 4294967295
	got := POSIXCksum([]byte{})
	if got != 4294967295 {
		t.Errorf("POSIXCksum(empty) = %d, want 4294967295", got)
	}
}

func TestGenerateFolderName(t *testing.T) {
	// The folder name should include a slug and hex-formatted CRC32
	name := GenerateFolderName("/home/user/project")
	if !strings.Contains(name, "_") {
		t.Errorf("GenerateFolderName should contain underscore separator, got %s", name)
	}
	// Should end with 8-char hex
	parts := strings.Split(name, "_")
	lastPart := parts[len(parts)-1]
	if len(lastPart) != 8 {
		t.Errorf("hash part should be 8 chars, got %d: %s", len(lastPart), lastPart)
	}
}

func parseUint32(s string, v *uint32) (int, error) {
	var n uint64
	for _, c := range s {
		n = n*10 + uint64(c-'0')
	}
	*v = uint32(n)
	return len(s), nil
}
