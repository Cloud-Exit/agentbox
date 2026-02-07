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

var enableCmd = &cobra.Command{
	Use:   "enable <agent>",
	Short: "Enable an agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if !agent.IsValidAgent(name) {
			ui.Errorf("Unknown agent: %s", name)
		}

		cfg := config.LoadOrDefault()
		if cfg.IsAgentEnabled(name) {
			ui.Infof("Agent '%s' is already enabled", name)
			return
		}

		cfg.SetAgentEnabled(name, true)
		if err := config.SaveConfig(cfg); err != nil {
			ui.Errorf("Failed to save config: %v", err)
		}

		ui.Successf("%s enabled", agent.DisplayName(name))
		ui.Infof("Run 'exitbox run %s' to start using it", name)
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
