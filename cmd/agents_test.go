// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-exit/exitbox/internal/config"
)

func TestAgentsMdPath(t *testing.T) {
	p := agentsMdPath("personal")
	expected := filepath.Join(config.Home, "profiles", "global", "personal", "agents.md")
	if p != expected {
		t.Errorf("agentsMdPath(personal) = %q, want %q", p, expected)
	}
}

func TestAgentsMdPath_DefaultWorkspace(t *testing.T) {
	p := agentsMdPath("default")
	if !strings.HasSuffix(p, filepath.Join("global", "default", "agents.md")) {
		t.Errorf("agentsMdPath(default) = %q, expected to end with global/default/agents.md", p)
	}
}
