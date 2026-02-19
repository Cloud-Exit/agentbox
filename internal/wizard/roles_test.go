// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package wizard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeExternalToolPackages_Valid(t *testing.T) {
	pkgs := ComputeExternalToolPackages([]string{"GitHub CLI"})
	if len(pkgs) != 1 || pkgs[0] != "github-cli" {
		t.Errorf("ComputeExternalToolPackages(GitHub CLI) = %v, want [github-cli]", pkgs)
	}
}

func TestComputeExternalToolPackages_Empty(t *testing.T) {
	pkgs := ComputeExternalToolPackages(nil)
	if pkgs != nil {
		t.Errorf("ComputeExternalToolPackages(nil) = %v, want nil", pkgs)
	}
}

func TestComputeExternalToolPackages_Unknown(t *testing.T) {
	pkgs := ComputeExternalToolPackages([]string{"Unknown Tool"})
	if pkgs != nil {
		t.Errorf("ComputeExternalToolPackages(Unknown Tool) = %v, want nil", pkgs)
	}
}

func TestDetectExternalToolConfigs(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create the .config/gh directory to simulate GitHub CLI installed.
	ghDir := filepath.Join(tmpHome, ".config", "gh")
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatal(err)
	}

	detected := DetectExternalToolConfigs()
	if detected == nil {
		t.Fatal("DetectExternalToolConfigs() returned nil, expected map with GitHub CLI")
	}
	paths, ok := detected["GitHub CLI"]
	if !ok || len(paths) == 0 {
		t.Errorf("expected GitHub CLI to be detected, got %v", detected)
	}
	if paths[0] != ".config/gh" {
		t.Errorf("expected .config/gh, got %q", paths[0])
	}
}

func TestDetectExternalToolConfigs_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	detected := DetectExternalToolConfigs()
	if detected != nil {
		t.Errorf("DetectExternalToolConfigs() with empty HOME = %v, want nil", detected)
	}
}

func TestDetectExternalToolConfigs_NothingDetected(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	detected := DetectExternalToolConfigs()
	if detected != nil {
		t.Errorf("DetectExternalToolConfigs() with empty home = %v, want nil", detected)
	}
}
