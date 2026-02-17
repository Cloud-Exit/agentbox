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
	"io/fs"
	"path/filepath"

	"github.com/cloud-exit/exitbox/internal/agent"
	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/platform"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/cloud-exit/exitbox/internal/vault"
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

		// Data Stores section.
		ui.Cecho("Data Stores", ui.Cyan)
		fmt.Println()

		cfg := config.LoadOrDefault()
		if len(cfg.Workspaces.Items) == 0 {
			fmt.Println("  No workspaces configured.")
		} else {
			for _, ws := range cfg.Workspaces.Items {
				fmt.Printf("  Workspace: %s\n", ws.Name)

				// Vault status.
				if vault.IsInitialized(ws.Name) {
					status := "initialized"
					if ws.Vault.Enabled {
						status += ", enabled"
					} else {
						status += ", disabled"
					}
					size, err := dirSize(config.VaultDir(ws.Name))
					if err == nil {
						fmt.Printf("    %-14s %s (%s)\n", "Vault:", status, formatBytes(size))
					} else {
						fmt.Printf("    %-14s %s\n", "Vault:", status)
					}
				} else {
					fmt.Printf("    %-14s not initialized\n", "Vault:")
				}

				// KV Store status.
				kvDir := config.KVDir(ws.Name)
				size, err := dirSize(kvDir)
				if err != nil || size == 0 {
					fmt.Printf("    %-14s empty\n", "KV Store:")
				} else {
					fmt.Printf("    %-14s %s\n", "KV Store:", formatBytes(size))
				}

				fmt.Println()
			}
		}
	},
}

// dirSize walks a directory tree and returns the total size of regular files.
func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
