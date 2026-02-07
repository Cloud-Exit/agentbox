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

package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/cloud-exit/exitbox/internal/agent"
	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/image"
	"github.com/cloud-exit/exitbox/internal/project"
	"github.com/cloud-exit/exitbox/internal/run"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <agent> [args...]",
	Short: "Run an agent in a container",
	Long: `Run an AI coding assistant in an isolated container.

Available agents:
  claude      Claude Code (Anthropic)
  codex       OpenAI Codex CLI
  opencode    OpenCode (open-source)

Profiles:
  Profiles install languages and tools into the container environment.
  Default profiles are configured via 'exitbox setup' and apply to all projects.
  Per-project profiles override or extend defaults for a single directory.

  exitbox run <agent> profile list         List all profiles (* = default)
  exitbox run <agent> profile status       Show active global + project profiles
  exitbox run <agent> profile add <name>   Add a per-project profile
  exitbox run <agent> profile remove <name>

Flags (passed after the agent name):
  -f, --no-firewall       Disable network firewall
  -r, --read-only         Mount workspace as read-only
  -n, --no-env            Don't pass host environment variables
  -u, --update            Check for and apply agent updates
  -v, --verbose           Enable verbose output
  -e, --env KEY=VALUE     Pass environment variables
  -t, --tools PKG         Add Alpine packages to the image
  -i, --include-dir DIR   Mount host dir inside /workspace
  -a, --allow-urls DOM    Allow extra domains for this session

Examples:
  exitbox run claude
  exitbox run claude -f -e GITHUB_TOKEN=$GITHUB_TOKEN
  exitbox run codex --update
  exitbox run claude profile add go`,
}

func newAgentRunCmd(agentName string) *cobra.Command {
	display := agent.DisplayName(agentName)
	return &cobra.Command{
		Use:   agentName + " [args...]",
		Short: "Run " + display,
		Long: "Run " + display + " in an isolated container.\n\n" +
			"Subcommands:\n" +
			"  profile   Manage development profiles (languages and tools)\n\n" +
			"Run 'exitbox run " + agentName + " profile --help' for profile commands.",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			// DisableFlagParsing swallows --help; handle it manually
			for _, a := range args {
				if a == "--" {
					break
				}
				if a == "--help" || a == "-h" {
					_ = cmd.Help()
					return
				}
				if !strings.HasPrefix(a, "-") {
					break
				}
			}
			runAgent(agentName, args)
		},
	}
}

func runAgent(agentName string, passthrough []string) {
	cfg := config.LoadOrDefault()
	if !cfg.IsAgentEnabled(agentName) {
		ui.Errorf("Agent '%s' is not enabled. Run 'exitbox enable %s' first.", agentName, agentName)
	}

	rt := container.Detect()
	if rt == nil {
		ui.Error("No container runtime found. Install Podman or Docker.")
	}

	projectDir, _ := os.Getwd()
	ctx := context.Background()

	// Initialize project
	if err := project.Init(projectDir); err != nil {
		ui.Warnf("Failed to initialize project directory: %v", err)
	}

	// Build images (flags parsed below, but tools/update need to be set before build)
	image.Version = Version

	flags := parseRunFlags(passthrough, cfg.Settings.DefaultFlags)

	if flags.Verbose {
		ui.Verbose = true
	}

	// Wire tools and update flags before building images
	image.SessionTools = flags.Tools
	image.ForceRebuild = flags.ForceUpdate
	image.AutoUpdate = cfg.Settings.AutoUpdate || flags.ForceUpdate

	if err := image.BuildProject(ctx, rt, agentName, projectDir); err != nil {
		ui.Errorf("Failed to build images: %v", err)
	}

	opts := run.Options{
		Agent:       agentName,
		ProjectDir:  projectDir,
		NoFirewall:  flags.NoFirewall,
		ReadOnly:    flags.ReadOnly,
		NoEnv:       flags.NoEnv,
		EnvVars:     flags.EnvVars,
		IncludeDirs: flags.IncludeDirs,
		AllowURLs:   flags.AllowURLs,
		Passthrough: flags.Remaining,
		Verbose:     flags.Verbose,
		StatusBar:   cfg.Settings.StatusBar,
		Version:     Version,
	}

	exitCode, err := run.AgentContainer(rt, opts)
	if err != nil {
		ui.Errorf("%v", err)
	}
	os.Exit(exitCode)
}

type parsedFlags struct {
	NoFirewall  bool
	ReadOnly    bool
	NoEnv       bool
	Verbose     bool
	ForceUpdate bool
	EnvVars     []string
	IncludeDirs []string
	AllowURLs   []string
	Tools       []string
	Remaining   []string
}

func parseRunFlags(passthrough []string, defaults config.DefaultFlags) parsedFlags {
	f := parsedFlags{
		NoFirewall: defaults.NoFirewall,
		ReadOnly:   defaults.ReadOnly,
		NoEnv:      defaults.NoEnv,
	}

	for i := 0; i < len(passthrough); i++ {
		arg := passthrough[i]
		switch arg {
		case "-f", "--no-firewall":
			f.NoFirewall = true
		case "-r", "--read-only":
			f.ReadOnly = true
		case "-n", "--no-env":
			f.NoEnv = true
		case "-v", "--verbose":
			f.Verbose = true
		case "-u", "--update":
			f.ForceUpdate = true
		case "-e", "--env":
			if i+1 < len(passthrough) {
				i++
				f.EnvVars = append(f.EnvVars, passthrough[i])
			}
		case "-t", "--tools":
			if i+1 < len(passthrough) {
				i++
				f.Tools = append(f.Tools, passthrough[i])
			}
		case "-i", "--include-dir":
			if i+1 < len(passthrough) {
				i++
				f.IncludeDirs = append(f.IncludeDirs, passthrough[i])
			}
		case "-a", "--allow-urls":
			if i+1 < len(passthrough) {
				i++
				f.AllowURLs = append(f.AllowURLs, passthrough[i])
			}
		case "--":
			f.Remaining = append(f.Remaining, passthrough[i+1:]...)
			i = len(passthrough)
		default:
			f.Remaining = append(f.Remaining, arg)
		}
	}

	return f
}

func init() {
	for _, name := range agent.AgentNames {
		agentCmd := newAgentRunCmd(name)

		// Add profile subcommand
		agentProfileCmd := &cobra.Command{
			Use:   "profile",
			Short: "Manage profiles for " + agent.DisplayName(name),
		}
		agentName := name // capture for closure
		agentProfileCmd.AddCommand(newProfileListCmd())
		agentProfileCmd.AddCommand(newProfileAddCmd(agentName))
		agentProfileCmd.AddCommand(newProfileRemoveCmd(agentName))
		agentProfileCmd.AddCommand(newProfileStatusCmd(agentName))

		agentCmd.AddCommand(agentProfileCmd)
		runCmd.AddCommand(agentCmd)
	}

	rootCmd.AddCommand(runCmd)
}
