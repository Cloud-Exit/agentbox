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

package wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloud-exit/exitbox/internal/config"
)

// Run executes the setup wizard TUI and writes config files on completion.
// Returns nil on success, error if cancelled or write fails.
func Run() error {
	model := NewModel()
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	wm := finalModel.(Model)
	if wm.Cancelled() {
		return fmt.Errorf("setup cancelled")
	}
	if !wm.Confirmed() {
		return fmt.Errorf("setup cancelled")
	}

	return applyResult(wm.Result())
}

func applyResult(state State) error {
	cfg := &config.Config{
		Version: 1,
		Roles:   state.Roles,
	}

	for _, name := range state.Agents {
		cfg.SetAgentEnabled(name, true)
	}

	cfg.Tools.User = ComputePackages(state.ToolCategories)

	for _, b := range ComputeBinaries(state.ToolCategories) {
		cfg.Tools.Binaries = append(cfg.Tools.Binaries, config.BinaryConfig{
			Name:       b.Name,
			URLPattern: b.URLPattern,
		})
	}

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	al := config.DefaultAllowlist()
	if err := config.SaveAllowlist(al); err != nil {
		return fmt.Errorf("saving allowlist: %w", err)
	}

	config.EnsureDirs()

	return nil
}
