// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProjectProfiles_NotExist(t *testing.T) {
	profiles, err := GetProjectProfiles("claude", "/nonexistent/path")
	if err != nil {
		t.Errorf("expected nil error for nonexistent profiles, got %v", err)
	}
	if profiles != nil {
		t.Errorf("expected nil profiles for nonexistent path, got %v", profiles)
	}
}

func TestGetProjectProfiles_ValidFile(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".exitbox", "claude")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "profiles:\n  - python\n  - node\n"
	if err := os.WriteFile(filepath.Join(agentDir, "profiles.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	profiles, err := GetProjectProfiles("claude", dir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(profiles) != 2 || profiles[0] != "python" || profiles[1] != "node" {
		t.Errorf("unexpected profiles: %v", profiles)
	}
}

func TestGetProjectProfiles_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".exitbox", "claude")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "profiles.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := GetProjectProfiles("claude", dir)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
