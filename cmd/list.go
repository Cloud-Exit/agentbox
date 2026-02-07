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

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available agents",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadOrDefault()
		rt := container.Detect()

		ui.LogoSmall()
		fmt.Println()
		ui.Cecho("Available Agents:", ui.Cyan)
		fmt.Println()
		fmt.Printf("  %-12s %-15s %-10s %-10s\n", "AGENT", "DISPLAY NAME", "ENABLED", "IMAGE")
		fmt.Printf("  %-12s %-15s %-10s %-10s\n", "-----", "------------", "-------", "-----")

		agents := []struct {
			Name    string
			Display string
		}{
			{"claude", "Claude Code"},
			{"codex", "OpenAI Codex"},
			{"opencode", "OpenCode"},
		}

		for _, a := range agents {
			enabledText := "no"
			enabledColor := ui.Dim
			if cfg.IsAgentEnabled(a.Name) {
				enabledText = "yes"
				enabledColor = ui.Green
			}

			imageText := "not built"
			imageColor := ui.Dim
			if rt != nil && rt.ImageExists("exitbox-"+a.Name+"-core") {
				imageText = "built"
				imageColor = ui.Green
			}

			fmt.Printf("  %-12s %-15s %s%-10s%s %s%-10s%s\n",
				a.Name, a.Display,
				enabledColor, enabledText, ui.NC,
				imageColor, imageText, ui.NC)
		}

		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  exitbox setup              Run the setup wizard")
		fmt.Println("  exitbox <agent>            Run an agent (builds if needed)")
		fmt.Println("  exitbox enable <agent>     Enable an agent")
		fmt.Println("  exitbox disable <agent>    Disable an agent")
		fmt.Println("  exitbox rebuild <agent>    Force rebuild of agent image")
		fmt.Println("  exitbox rebuild all        Rebuild all enabled agents")
		fmt.Println("  exitbox import <agent>     Import agent config from host")
		fmt.Println("  exitbox uninstall [agent]  Uninstall exitbox or specific agent")
		fmt.Println("  exitbox aliases            Print shell aliases")
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
