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
	"github.com/cloud-exit/exitbox/internal/agent"
	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <agent|all>",
	Short: "Import agent config from host",
	Args:  cobra.ExactArgs(1),
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
			dst := config.AgentDir(name)
			if err := a.ImportConfig(src, dst); err != nil {
				ui.Warnf("Failed to import %s config: %v", name, err)
				continue
			}
			ui.Successf("Imported %s config from %s", name, src)
			importedAny = true
		}

		if !importedAny {
			ui.Warn("No configs were imported.")
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
