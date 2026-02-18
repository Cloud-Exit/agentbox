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

package update

import "testing"

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"3.2.0", "3.3.0", true},
		{"3.2.0", "3.2.1", true},
		{"3.2.0", "4.0.0", true},
		{"3.2.0", "3.2.0", false},
		{"3.3.0", "3.2.0", false},
		{"4.0.0", "3.9.9", false},
		{"dev", "3.3.0", false},
		{"3.2.0", "dev", false},
		{"dev", "dev", false},
		{"1.0.0", "1.0.1", true},
		{"0.9.0", "1.0.0", true},
		{"v3.2.0", "v3.3.0", true},
		{"3.2.0", "v3.3.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			got := IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestBinaryURLFor(t *testing.T) {
	tests := []struct {
		version string
		goos    string
		goarch  string
		want    string
	}{
		{
			"3.3.0", "linux", "amd64",
			"https://github.com/Cloud-Exit/ExitBox/releases/download/v3.3.0/exitbox-linux-amd64",
		},
		{
			"3.3.0", "darwin", "arm64",
			"https://github.com/Cloud-Exit/ExitBox/releases/download/v3.3.0/exitbox-darwin-arm64",
		},
		{
			"3.3.0", "windows", "amd64",
			"https://github.com/Cloud-Exit/ExitBox/releases/download/v3.3.0/exitbox-windows-amd64.exe",
		},
		{
			"1.0.0", "linux", "arm64",
			"https://github.com/Cloud-Exit/ExitBox/releases/download/v1.0.0/exitbox-linux-arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got := BinaryURLFor(tt.version, tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("BinaryURLFor(%q, %q, %q) =\n  %q\nwant\n  %q", tt.version, tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
		ok    bool
	}{
		{"3.2.0", [3]int{3, 2, 0}, true},
		{"v3.2.0", [3]int{3, 2, 0}, true},
		{"0.0.1", [3]int{0, 0, 1}, true},
		{"dev", [3]int{}, false},
		{"3.2", [3]int{}, false},
		{"3.2.x", [3]int{}, false},
		{"", [3]int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseSemver(tt.input)
			if ok != tt.ok {
				t.Errorf("parseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
