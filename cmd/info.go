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

	"github.com/cloud-exit/exitbox/internal/agent"
	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/platform"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show system and project information",
	Run: func(cmd *cobra.Command, args []string) {
		rt := container.Detect()

		ui.LogoSmall()
		fmt.Println()
		ui.Cecho("System Information", ui.Cyan)
		fmt.Println()

		fmt.Printf("  %-20s %s\n", "Version:", Version)
		fmt.Printf("  %-20s %s\n", "Platform:", platform.GetPlatform())
		fmt.Printf("  %-20s %s\n", "Config dir:", config.Home)
		fmt.Printf("  %-20s %s\n", "Cache dir:", config.Cache)
		fmt.Println()

		ui.Cecho("Container Runtime", ui.Cyan)
		fmt.Println()

		if rt != nil {
			fmt.Printf("  %-20s %s\n", "Runtime:", rt.Name())
			if container.IsAvailable(rt) {
				fmt.Printf("  %-20s %srunning%s\n", "Status:", ui.Green, ui.NC)
			} else {
				fmt.Printf("  %-20s %snot running%s\n", "Status:", ui.Red, ui.NC)
			}
		} else {
			fmt.Printf("  %-20s %snot found%s\n", "Runtime:", ui.Red, ui.NC)
			fmt.Println("  Install Podman (recommended) or Docker to use exitbox.")
		}

		fmt.Println()
		ui.Cecho("Built Agents", ui.Cyan)
		fmt.Println()

		found := false
		for _, name := range agent.AgentNames {
			if rt != nil && rt.ImageExists("exitbox-"+name+"-core") {
				fmt.Printf("  • %s (%s)\n", agent.DisplayName(name), name)
				found = true
			}
		}
		if !found {
			fmt.Println("  No agents built. Run 'exitbox run <agent>' to build and run one.")
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
