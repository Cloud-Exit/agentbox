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
	"fmt"
	"time"

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/cloud-exit/exitbox/internal/update"
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
		fmt.Println("  exitbox run <agent>        Run an agent (builds if needed)")
		fmt.Println("  exitbox generate <agent>   Generate config for a third-party LLM server")
		fmt.Println("  exitbox enable <agent>     Enable an agent")
		fmt.Println("  exitbox disable <agent>    Disable an agent")
		fmt.Println("  exitbox rebuild <agent>    Force rebuild of agent image")
		fmt.Println("  exitbox rebuild all        Rebuild all enabled agents")
		fmt.Println("  exitbox import <agent>     Import agent config from host")
		fmt.Println("  exitbox workspaces         Manage workspaces")
		fmt.Println("  exitbox sessions           Manage sessions")
		fmt.Println("  exitbox vault              Manage vault secrets")
		fmt.Println("  exitbox logs <agent>       View agent build logs")
		fmt.Println("  exitbox info               Show system and project information")
		fmt.Println("  exitbox uninstall [agent]  Uninstall exitbox or specific agent")
		fmt.Println("  exitbox update             Update ExitBox to the latest version")
		fmt.Println("  exitbox aliases            Print shell aliases")
		fmt.Println()

		// Non-blocking update check with a short timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		ch := make(chan string, 1)
		go func() {
			latest, err := update.GetLatestVersion()
			if err == nil && update.IsNewer(Version, latest) {
				ch <- latest
			}
			close(ch)
		}()

		select {
		case latest, ok := <-ch:
			if ok && latest != "" {
				fmt.Printf("  %sUpdate available: v%s → v%s — run `exitbox update` to update.%s\n\n",
					ui.Yellow, Version, latest, ui.NC)
			}
		case <-ctx.Done():
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
