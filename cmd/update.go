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

	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/cloud-exit/exitbox/internal/update"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ExitBox to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		runUpdate()
	},
}

func runUpdate() {
	fmt.Println("Checking for updates...")

	latest, err := update.GetLatestVersion()
	if err != nil {
		ui.Errorf("Failed to check for updates: %v", err)
	}

	if !update.IsNewer(Version, latest) {
		ui.Successf("ExitBox is up to date (v%s)", Version)
		return
	}

	fmt.Printf("Updating ExitBox: v%s → v%s...\n", Version, latest)

	url := update.BinaryURL(latest)
	if err := update.DownloadAndReplace(url); err != nil {
		ui.Errorf("Update failed: %v", err)
	}

	ui.Successf("ExitBox updated to v%s", latest)
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
