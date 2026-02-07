// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package network

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/cloud-exit/exitbox/internal/config"
)

func TestSessionURLLifecycle(t *testing.T) {
	// Use a temp dir for cache
	tmpDir := t.TempDir()
	origCache := config.Cache
	config.Cache = tmpDir
	defer func() { config.Cache = origCache }()

	// Register URLs for two containers
	if err := RegisterSessionURLs("container-a", []string{"example.com", "foo.com"}); err != nil {
		t.Fatalf("RegisterSessionURLs(a): %v", err)
	}
	if err := RegisterSessionURLs("container-b", []string{"bar.com", "example.com"}); err != nil {
		t.Fatalf("RegisterSessionURLs(b): %v", err)
	}

	// Collect all - should be deduplicated
	all := collectAllSessionURLs()
	sort.Strings(all)
	if len(all) != 3 {
		t.Errorf("expected 3 unique URLs, got %d: %v", len(all), all)
	}

	// Remove container-a's session
	sessionFile := filepath.Join(sessionDir(), "container-a.urls")
	os.Remove(sessionFile)

	// Remaining should only have bar.com and example.com
	remaining := collectAllSessionURLs()
	sort.Strings(remaining)
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining URLs, got %d: %v", len(remaining), remaining)
	}

	// Clean all
	os.RemoveAll(sessionDir())
	empty := collectAllSessionURLs()
	if len(empty) != 0 {
		t.Errorf("expected 0 URLs after cleanup, got %d", len(empty))
	}
}
