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

	"github.com/cloud-exit/exitbox/internal/agent"
	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/profile"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.LoadOrDefault()
			defaultSet := make(map[string]bool, len(cfg.Profiles))
			for _, p := range cfg.Profiles {
				defaultSet[p] = true
			}

			ui.LogoSmall()
			fmt.Println()
			ui.Cecho("Available Profiles:", ui.Cyan)
			fmt.Println()
			fmt.Printf("  %-15s %-10s %s\n", "PROFILE", "DEFAULT", "DESCRIPTION")
			fmt.Printf("  %-15s %-10s %s\n", "───────", "───────", "───────────")
			for _, p := range profile.All() {
				marker := ""
				if defaultSet[p.Name] {
					marker = "*"
				}
				fmt.Printf("  %-15s %-10s %s\n", p.Name, marker, p.Description)
			}
			fmt.Println()
			if len(cfg.Profiles) > 0 {
				fmt.Println("  * = default profile (configured via 'exitbox setup')")
				fmt.Println()
			}
			fmt.Println("Add per-project profiles with: exitbox run <agent> profile add <name>")
			fmt.Println("Change default profiles with:  exitbox setup")
			fmt.Println()
		},
	}
}

func newProfileAddCmd(agentName string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Add a profile",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectDir, _ := os.Getwd()
			if err := profile.AddProjectProfile(agentName, projectDir, args[0]); err != nil {
				ui.Errorf("%v", err)
			}
			ui.Successf("Added profile '%s' for %s", args[0], agentName)
			ui.Infof("Run 'exitbox run %s' to rebuild with the new profile", agentName)
		},
	}
}

func newProfileRemoveCmd(agentName string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a profile",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectDir, _ := os.Getwd()
			if err := profile.RemoveProjectProfile(agentName, projectDir, args[0]); err != nil {
				ui.Errorf("%v", err)
			}
			ui.Successf("Removed profile '%s' for %s", args[0], agentName)
			ui.Infof("Run 'exitbox run %s' to rebuild without the profile", agentName)
		},
	}
}

func newProfileStatusCmd(agentName string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current profiles",
		Run: func(cmd *cobra.Command, args []string) {
			projectDir, _ := os.Getwd()
			cfg := config.LoadOrDefault()
			projectProfiles, _ := profile.GetProjectProfiles(agentName, projectDir)

			fmt.Println()
			ui.Cecho(agent.DisplayName(agentName)+" Profile Status", ui.Cyan)
			fmt.Println()

			if len(cfg.Profiles) > 0 {
				fmt.Println("Global profiles (from setup wizard):")
				for _, p := range cfg.Profiles {
					desc := ""
					if pr := profile.Get(p); pr != nil {
						desc = pr.Description
					}
					fmt.Printf("  %s - %s\n", p, desc)
				}
			} else {
				fmt.Println("Global profiles: none")
			}

			fmt.Println()

			if len(projectProfiles) > 0 {
				fmt.Println("Project profiles (this directory):")
				for _, p := range projectProfiles {
					desc := ""
					if pr := profile.Get(p); pr != nil {
						desc = pr.Description
					}
					fmt.Printf("  %s - %s\n", p, desc)
				}
			} else {
				fmt.Println("Project profiles: none")
			}

			fmt.Println()
			fmt.Printf("Run 'exitbox run %s profile add <name>' to add per-project profiles.\n", agentName)
			fmt.Println("Run 'exitbox setup' to change global profiles.")
		},
	}
}
