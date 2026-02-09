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
	"os"
	"strings"

	"github.com/cloud-exit/exitbox/internal/agent"
	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/profile"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "import <agent|all>",
		Short: "Import agent config from host",
		Long: `Import agent configuration and credentials from the host into a workspace.

By default, imports into the active workspace. Use --workspace to target
a specific workspace.

Examples:
  exitbox import claude                    Import Claude config into active workspace
  exitbox import all                       Import all agent configs into active workspace
  exitbox import claude -w work            Import Claude config into 'work' workspace
  exitbox import all --workspace personal  Import all configs into 'personal' workspace`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]

			var agents []string
			if target == "all" {
				agents = agent.AgentNames
			} else {
				if !agent.IsValidAgent(target) {
					ui.Errorf("Unknown agent: %s", target)
				}
				agents = []string{target}
			}

			// Resolve target workspace.
			cfg := config.LoadOrDefault()
			workspaceName := resolveImportWorkspace(cfg, workspace)

			importedAny := false
			for _, name := range agents {
				a := agent.Get(name)
				if a == nil {
					continue
				}
				src, err := a.DetectHostConfig()
				if err != nil {
					ui.Warnf("No host config found for %s", name)
					continue
				}
				dst := profile.WorkspaceAgentDir(workspaceName, name)
				if err := os.MkdirAll(dst, 0755); err != nil {
					ui.Warnf("Failed to create workspace dir for %s: %v", name, err)
					continue
				}
				if err := a.ImportConfig(src, dst); err != nil {
					ui.Warnf("Failed to import %s config: %v", name, err)
					continue
				}
				ui.Successf("Imported %s config from %s → workspace '%s'", name, src, workspaceName)
				importedAny = true
			}

			if !importedAny {
				ui.Warn("No configs were imported.")
			}
		},
	}

	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Target workspace (default: active workspace)")
	return cmd
}

// resolveImportWorkspace determines which workspace to import into.
func resolveImportWorkspace(cfg *config.Config, override string) string {
	if override != "" {
		w := profile.FindWorkspace(cfg, override)
		if w == nil {
			available := profile.WorkspaceNames(cfg)
			if len(available) > 0 {
				ui.Errorf("Unknown workspace '%s'. Available: %s", override, strings.Join(available, ", "))
			} else {
				ui.Errorf("Unknown workspace '%s'. No workspaces configured. Run 'exitbox setup' first.", override)
			}
		}
		return w.Name
	}

	// Use the active/default workspace.
	projectDir, _ := os.Getwd()
	active, _ := profile.ResolveActiveWorkspace(cfg, projectDir, "")
	if active != nil {
		return active.Workspace.Name
	}

	// Fallback if no workspace exists at all.
	if len(cfg.Workspaces.Items) > 0 {
		return cfg.Workspaces.Items[0].Name
	}
	return "default"
}

func init() {
	rootCmd.AddCommand(newImportCmd())
}
