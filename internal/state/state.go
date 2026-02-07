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

package state

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-exit/exitbox/internal/config"
)

// IsFirstInstall returns true if this is the first installation.
func IsFirstInstall() bool {
	_, err := os.Stat(filepath.Join(config.Home, ".installed"))
	return os.IsNotExist(err)
}

// MarkInstalled marks the installation as complete.
func MarkInstalled() error {
	return os.WriteFile(filepath.Join(config.Home, ".installed"), []byte{}, 0644)
}

// GetInstalledVersion reads the saved version.
func GetInstalledVersion() string {
	data, err := os.ReadFile(filepath.Join(config.Home, ".version"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SaveVersion saves the current version.
func SaveVersion(version string) error {
	return os.WriteFile(filepath.Join(config.Home, ".version"), []byte(version+"\n"), 0644)
}
