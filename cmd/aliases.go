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
	"github.com/spf13/cobra"
)

var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Print shell aliases",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("# ExitBox setup - add to ~/.zshrc or ~/.bashrc")
		fmt.Println()
		fmt.Println("# Add exitbox to PATH")
		fmt.Println(`export PATH="$HOME/.local/bin:$PATH"`)
		fmt.Println()
		fmt.Println("# Agent aliases")
		for _, name := range agent.AgentNames {
			fmt.Printf("alias %s='exitbox %s'\n", name, name)
		}
	},
}

func init() {
	rootCmd.AddCommand(aliasesCmd)
}
