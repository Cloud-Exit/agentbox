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
	"os/exec"

	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/network"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean [unused|all|containers]",
	Short: "Clean up Docker resources",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mode := "unused"
		if len(args) > 0 {
			mode = args[0]
		}

		rt := container.Detect()
		if rt == nil {
			ui.Error("No container runtime found.")
		}

		rtCmd := container.Cmd(rt)

		switch mode {
		case "unused":
			ui.Info("Removing unused exitbox images...")
			_ = exec.Command(rtCmd, "image", "prune", "-f", "--filter", "label=exitbox.version").Run()
			ui.Success("Cleanup complete")

		case "all":
			ui.Info("Removing all exitbox images...")
			// List and remove all exitbox images
			out, _ := exec.Command(rtCmd, "images", "--filter", "reference=exitbox-*", "--format", "{{.Repository}}:{{.Tag}}").Output()
			for _, img := range splitLines(string(out)) {
				_ = exec.Command(rtCmd, "rmi", "-f", img).Run()
			}
			ui.Success("Cleanup complete")

		case "containers":
			ui.Info("Stopping all exitbox containers...")
			out, _ := exec.Command(rtCmd, "ps", "--filter", "name=exitbox-", "--format", "{{.ID}}").Output()
			for _, id := range splitLines(string(out)) {
				_ = exec.Command(rtCmd, "stop", id).Run()
			}
			network.CleanupSquidIfUnused(rt)
			ui.Success("Cleanup complete")

		default:
			fmt.Println("Usage: exitbox clean [unused|all|containers]")
			fmt.Println()
			fmt.Println("Modes:")
			fmt.Println("  unused      Remove unused images (default)")
			fmt.Println("  all         Remove all exitbox images")
			fmt.Println("  containers  Stop all exitbox containers")
			fmt.Println()
		}
	},
}

func splitLines(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 {
				result = append(result, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
