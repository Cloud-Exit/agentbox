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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/profile"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage per-workspace agent instructions (agents.md)",
		Long: "Manage the agents.md file that provides custom instructions to AI agents.\n\n" +
			"Each workspace has its own agents.md file. On container start, ExitBox\n" +
			"combines your agents.md with its sandbox instructions into the agent's\n" +
			"instruction file (e.g. ~/.claude/CLAUDE.md).\n\n" +
			"Your agents.md is never modified by ExitBox — it is fully user-owned.",
	}

	cmd.AddCommand(newAgentsEditCmd())
	cmd.AddCommand(newAgentsShowCmd())
	cmd.AddCommand(newAgentsPathCmd())
	return cmd
}

func newAgentsEditCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the agents.md file for a workspace",
		Long:  "Opens the workspace's agents.md in your $EDITOR (or vi).",
		Run: func(cmd *cobra.Command, args []string) {
			ws := resolveAgentsWorkspace(workspace)
			p := agentsMdPath(ws)

			// Create the file with a helpful header if it doesn't exist.
			if _, err := os.Stat(p); os.IsNotExist(err) {
				if mkErr := os.MkdirAll(filepath.Dir(p), 0755); mkErr != nil {
					ui.Errorf("Failed to create directory: %v", mkErr)
				}
				header := fmt.Sprintf("# Agent Instructions — workspace: %s\n\n"+
					"# Add your custom instructions here. They will be included in\n"+
					"# the agent's instruction file (e.g. ~/.claude/CLAUDE.md) on\n"+
					"# every container start, before ExitBox's sandbox rules.\n\n", ws)
				if wErr := os.WriteFile(p, []byte(header), 0644); wErr != nil {
					ui.Errorf("Failed to create agents.md: %v", wErr)
				}
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			c := exec.Command(editor, p)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				ui.Errorf("Editor exited with error: %v", err)
			}
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

func newAgentsShowCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the agents.md file for a workspace",
		Run: func(cmd *cobra.Command, args []string) {
			ws := resolveAgentsWorkspace(workspace)
			p := agentsMdPath(ws)

			data, err := os.ReadFile(p)
			if os.IsNotExist(err) {
				ui.Infof("No agents.md for workspace '%s'.", ws)
				fmt.Printf("Create one with: exitbox agents edit -w %s\n", ws)
				return
			}
			if err != nil {
				ui.Errorf("Failed to read agents.md: %v", err)
			}
			fmt.Print(string(data))
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

func newAgentsPathCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print the path to the agents.md file",
		Run: func(cmd *cobra.Command, args []string) {
			ws := resolveAgentsWorkspace(workspace)
			fmt.Println(agentsMdPath(ws))
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

// agentsMdPath returns the host-side path to the workspace's agents.md.
// The file is stored at the workspace level (not per-agent) so the same
// instructions apply to all agents in the workspace.
func agentsMdPath(workspaceName string) string {
	return filepath.Join(config.Home, "profiles", "global", workspaceName, "agents.md")
}

func resolveAgentsWorkspace(flag string) string {
	if flag != "" {
		return flag
	}
	cfg := config.LoadOrDefault()
	projectDir, _ := os.Getwd()
	active, err := profile.ResolveActiveWorkspace(cfg, projectDir, "")
	if err == nil && active != nil {
		return active.Workspace.Name
	}
	if cfg.Settings.DefaultWorkspace != "" {
		return cfg.Settings.DefaultWorkspace
	}
	ui.Error("No workspace specified. Use -w <workspace> or set a default workspace.")
	return "" // unreachable (ui.Error exits)
}

func init() {
	rootCmd.AddCommand(newAgentsCmd())
}
