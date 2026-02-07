// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package project

import "testing"

func TestRandomHexLength(t *testing.T) {
	for _, n := range []int{1, 4, 8, 16} {
		got := randomHex(n)
		if len(got) != n*2 {
			t.Errorf("randomHex(%d) = %q (len %d), want len %d", n, got, len(got), n*2)
		}
	}
}

func TestRandomHexUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		h := randomHex(4)
		if seen[h] {
			t.Errorf("randomHex(4) produced duplicate: %s", h)
		}
		seen[h] = true
	}
}

func TestSlugifyPath(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/home/user/project", "home_user_project"},
		{"/tmp/foo bar", "tmp_foo_bar"},
		{"relative/path", "relative_path"},
	}
	for _, tc := range tests {
		got := SlugifyPath(tc.input)
		if got != tc.want {
			t.Errorf("SlugifyPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
