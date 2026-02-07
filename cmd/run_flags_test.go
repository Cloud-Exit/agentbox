// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package cmd

import (
	"testing"

	"github.com/cloud-exit/exitbox/internal/config"
)

func TestParseRunFlags_BooleanFlags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(parsedFlags) bool
	}{
		{"short no-firewall", []string{"-f"}, func(f parsedFlags) bool { return f.NoFirewall }},
		{"long no-firewall", []string{"--no-firewall"}, func(f parsedFlags) bool { return f.NoFirewall }},
		{"short read-only", []string{"-r"}, func(f parsedFlags) bool { return f.ReadOnly }},
		{"long read-only", []string{"--read-only"}, func(f parsedFlags) bool { return f.ReadOnly }},
		{"short no-env", []string{"-n"}, func(f parsedFlags) bool { return f.NoEnv }},
		{"long no-env", []string{"--no-env"}, func(f parsedFlags) bool { return f.NoEnv }},
		{"short verbose", []string{"-v"}, func(f parsedFlags) bool { return f.Verbose }},
		{"long verbose", []string{"--verbose"}, func(f parsedFlags) bool { return f.Verbose }},
		{"short update", []string{"-u"}, func(f parsedFlags) bool { return f.ForceUpdate }},
		{"long update", []string{"--update"}, func(f parsedFlags) bool { return f.ForceUpdate }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := parseRunFlags(tc.args, config.DefaultFlags{})
			if !tc.check(f) {
				t.Errorf("flag not set for args %v", tc.args)
			}
		})
	}
}

func TestParseRunFlags_EnvLongForm(t *testing.T) {
	f := parseRunFlags([]string{"--env", "FOO=bar"}, config.DefaultFlags{})
	if len(f.EnvVars) != 1 || f.EnvVars[0] != "FOO=bar" {
		t.Errorf("--env not parsed: %v", f.EnvVars)
	}
}

func TestParseRunFlags_EnvShortForm(t *testing.T) {
	f := parseRunFlags([]string{"-e", "FOO=bar"}, config.DefaultFlags{})
	if len(f.EnvVars) != 1 || f.EnvVars[0] != "FOO=bar" {
		t.Errorf("-e not parsed: %v", f.EnvVars)
	}
}

func TestParseRunFlags_Tools(t *testing.T) {
	f := parseRunFlags([]string{"-t", "jq", "--tools", "ripgrep"}, config.DefaultFlags{})
	if len(f.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %v", len(f.Tools), f.Tools)
	}
	if f.Tools[0] != "jq" || f.Tools[1] != "ripgrep" {
		t.Errorf("unexpected tools: %v", f.Tools)
	}
}

func TestParseRunFlags_DoubleDash(t *testing.T) {
	f := parseRunFlags([]string{"-f", "--", "-r", "extra"}, config.DefaultFlags{})
	if !f.NoFirewall {
		t.Error("-f before -- should be parsed")
	}
	if f.ReadOnly {
		t.Error("-r after -- should not be parsed as flag")
	}
	if len(f.Remaining) != 2 || f.Remaining[0] != "-r" || f.Remaining[1] != "extra" {
		t.Errorf("unexpected remaining: %v", f.Remaining)
	}
}

func TestParseRunFlags_DefaultsFromConfig(t *testing.T) {
	defaults := config.DefaultFlags{
		NoFirewall: true,
		ReadOnly:   true,
		NoEnv:      true,
	}
	f := parseRunFlags(nil, defaults)
	if !f.NoFirewall {
		t.Error("NoFirewall default not applied")
	}
	if !f.ReadOnly {
		t.Error("ReadOnly default not applied")
	}
	if !f.NoEnv {
		t.Error("NoEnv default not applied")
	}
}

func TestParseRunFlags_UnknownArgsPassedThrough(t *testing.T) {
	f := parseRunFlags([]string{"--custom-flag", "value"}, config.DefaultFlags{})
	if len(f.Remaining) != 2 || f.Remaining[0] != "--custom-flag" || f.Remaining[1] != "value" {
		t.Errorf("unknown args not passed through: %v", f.Remaining)
	}
}
