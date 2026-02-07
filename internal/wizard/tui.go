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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloud-exit/exitbox/internal/config"
)

// Step identifies the current wizard step.
type Step int

const (
	stepWelcome Step = iota
	stepRole
	stepLanguage
	stepTools
	stepAgents
	stepSettings
	stepReview
	stepDone
)

// State holds accumulated user selections across wizard steps.
type State struct {
	Roles          []string
	Languages      []string
	ToolCategories []string
	Agents         []string
	AutoUpdate     bool
	StatusBar      bool
}

// Model is the root bubbletea model for the wizard.
type Model struct {
	step      Step
	state     State
	cursor    int
	checked   map[string]bool
	width     int
	height    int
	cancelled bool
	confirmed bool
}

// NewModel creates a new wizard model with defaults.
func NewModel() Model {
	checked := make(map[string]bool)
	// Default settings to on
	checked["setting:auto_update"] = true
	checked["setting:status_bar"] = true
	return Model{
		step:    stepWelcome,
		checked: checked,
	}
}

// NewModelFromConfig creates a wizard model pre-populated from existing config.
func NewModelFromConfig(cfg *config.Config) Model {
	checked := make(map[string]bool)

	// Pre-check roles
	for _, r := range cfg.Roles {
		checked["role:"+r] = true
	}

	// Pre-check agents
	if cfg.Agents.Claude.Enabled {
		checked["agent:claude"] = true
	}
	if cfg.Agents.Codex.Enabled {
		checked["agent:codex"] = true
	}
	if cfg.Agents.OpenCode.Enabled {
		checked["agent:opencode"] = true
	}

	// Pre-check tool categories from saved selections (or fall back to role inference)
	if len(cfg.ToolCategories) > 0 {
		for _, tc := range cfg.ToolCategories {
			checked["tool:"+tc] = true
		}
	} else {
		for _, roleName := range cfg.Roles {
			if role := GetRole(roleName); role != nil {
				for _, t := range role.ToolCategories {
					checked["tool:"+t] = true
				}
			}
		}
	}

	// Pre-check languages from saved profiles (or fall back to role inference)
	if len(cfg.Profiles) > 0 {
		profileSet := make(map[string]bool, len(cfg.Profiles))
		for _, p := range cfg.Profiles {
			profileSet[p] = true
		}
		for _, l := range AllLanguages {
			if profileSet[l.Profile] {
				checked["lang:"+l.Name] = true
			}
		}
	} else {
		for _, roleName := range cfg.Roles {
			if role := GetRole(roleName); role != nil {
				for _, l := range role.Languages {
					checked["lang:"+l] = true
				}
			}
		}
	}

	// Settings
	checked["setting:auto_update"] = cfg.Settings.AutoUpdate
	checked["setting:status_bar"] = cfg.Settings.StatusBar

	return Model{
		step:    stepWelcome,
		checked: checked,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "q":
			if m.step == stepWelcome {
				m.cancelled = true
				return m, tea.Quit
			}
		}
	}

	switch m.step {
	case stepWelcome:
		return m.updateWelcome(msg)
	case stepRole:
		return m.updateRole(msg)
	case stepLanguage:
		return m.updateLanguage(msg)
	case stepTools:
		return m.updateTools(msg)
	case stepAgents:
		return m.updateAgents(msg)
	case stepSettings:
		return m.updateSettings(msg)
	case stepReview:
		return m.updateReview(msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.step {
	case stepWelcome:
		return m.viewWelcome()
	case stepRole:
		return m.viewRole()
	case stepLanguage:
		return m.viewLanguage()
	case stepTools:
		return m.viewTools()
	case stepAgents:
		return m.viewAgents()
	case stepSettings:
		return m.viewSettings()
	case stepReview:
		return m.viewReview()
	case stepDone:
		return ""
	}
	return ""
}

// Cancelled returns true if the user cancelled the wizard.
func (m Model) Cancelled() bool { return m.cancelled }

// Confirmed returns true if the user confirmed their selections.
func (m Model) Confirmed() bool { return m.confirmed }

// Result returns the final wizard state.
func (m Model) Result() State { return m.state }

// wrapWords joins words with ", " and wraps to maxWidth, indenting
// continuation lines with the given indent string.
func wrapWords(words []string, indent string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	lineLen := len(indent)
	b.WriteString(indent)

	for i, w := range words {
		seg := w
		if i < len(words)-1 {
			seg += ","
		}
		// +1 for the space before the word (except first on line)
		needed := len(seg)
		if lineLen > len(indent) {
			needed++ // space separator
		}

		if lineLen+needed > maxWidth && lineLen > len(indent) {
			b.WriteString("\n")
			b.WriteString(indent)
			lineLen = len(indent)
		}

		if lineLen > len(indent) {
			b.WriteString(" ")
			lineLen++
		}
		b.WriteString(seg)
		lineLen += len(seg)
	}
	return b.String()
}

// --- Welcome Step ---

func (m Model) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "enter" {
			m.step = stepRole
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) viewWelcome() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(logo))
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("Welcome to ExitBox Setup"))
	b.WriteString("\n\n")
	b.WriteString("This wizard will help you configure your development environment.\n")
	b.WriteString("You'll choose your role, languages, tools, and agents.\n\n")
	b.WriteString(helpStyle.Render("Press Enter to start, q to quit"))
	return b.String()
}

const logo = `  _____      _ _   ____
 | ____|_  _(_) |_| __ )  _____  __
 |  _| \ \/ / | __|  _ \ / _ \ \/ /
 | |___ >  <| | |_| |_) | (_) >  <
 |_____/_/\_\_|\__|____/ \___/_/\_\`

// --- Role Step (multi-select) ---

func (m Model) updateRole(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(Roles)-1 {
				m.cursor++
			}
		case " ", "x":
			k := "role:" + Roles[m.cursor].Name
			m.checked[k] = !m.checked[k]
		case "enter":
			m.state.Roles = nil
			// Pre-check languages and tools from all selected roles
			for _, role := range Roles {
				if m.checked["role:"+role.Name] {
					m.state.Roles = append(m.state.Roles, role.Name)
					for _, l := range role.Languages {
						m.checked["lang:"+l] = true
					}
					for _, t := range role.ToolCategories {
						m.checked["tool:"+t] = true
					}
				}
			}
			m.step = stepLanguage
			m.cursor = 0
		case "esc":
			m.step = stepWelcome
		}
	}
	return m, nil
}

func (m Model) viewRole() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Step 1/6 — What kind of developer are you?"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Select all that apply. Space to toggle.\n"))
	b.WriteString("\n")

	for i, role := range Roles {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		check := "[ ]"
		if m.checked["role:"+role.Name] {
			check = selectedStyle.Render("[x]")
		}
		// Pad name to fixed width before styling to prevent layout shift
		paddedName := fmt.Sprintf("%-15s", role.Name)
		if m.cursor == i {
			paddedName = selectedStyle.Render(paddedName)
		}
		b.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, check, paddedName, dimStyle.Render(role.Description)))
	}

	b.WriteString(helpStyle.Render("\nSpace to toggle, Enter to confirm, Esc to go back"))
	return b.String()
}

// --- Language Step (multi-select) ---

func (m Model) updateLanguage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(AllLanguages)-1 {
				m.cursor++
			}
		case " ", "x":
			k := "lang:" + AllLanguages[m.cursor].Name
			m.checked[k] = !m.checked[k]
		case "enter":
			m.state.Languages = nil
			for _, l := range AllLanguages {
				if m.checked["lang:"+l.Name] {
					m.state.Languages = append(m.state.Languages, l.Name)
				}
			}
			m.step = stepTools
			m.cursor = 0
		case "esc":
			m.step = stepRole
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) viewLanguage() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Step 2/6 — Which languages do you use?"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("These become your default profile, installed in all projects. Space to toggle.\n"))
	b.WriteString("\n")

	for i, lang := range AllLanguages {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		check := "[ ]"
		if m.checked["lang:"+lang.Name] {
			check = selectedStyle.Render("[x]")
		}
		paddedName := fmt.Sprintf("%-15s", lang.Name)
		if m.cursor == i {
			paddedName = selectedStyle.Render(paddedName)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, paddedName))
	}

	b.WriteString(helpStyle.Render("\nSpace to toggle, Enter to confirm, Esc to go back"))
	return b.String()
}

// --- Tools Step (multi-select) ---

func (m Model) updateTools(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(AllToolCategories)-1 {
				m.cursor++
			}
		case " ", "x":
			k := "tool:" + AllToolCategories[m.cursor].Name
			m.checked[k] = !m.checked[k]
		case "enter":
			m.state.ToolCategories = nil
			for _, t := range AllToolCategories {
				if m.checked["tool:"+t.Name] {
					m.state.ToolCategories = append(m.state.ToolCategories, t.Name)
				}
			}
			m.step = stepAgents
			m.cursor = 0
		case "esc":
			m.step = stepLanguage
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) viewTools() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Step 3/6 — Which tool categories do you need?"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Pre-selected based on your role. Space to toggle.\n"))
	b.WriteString("\n")

	for i, cat := range AllToolCategories {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		check := "[ ]"
		if m.checked["tool:"+cat.Name] {
			check = selectedStyle.Render("[x]")
		}
		paddedName := fmt.Sprintf("%-15s", cat.Name)
		if m.cursor == i {
			paddedName = selectedStyle.Render(paddedName)
		}
		// prefix: cursor(2) + check(3) + space(1) + name(15) + space(1) = 22 chars
		wrapped := wrapWords(cat.Packages, "                      ", m.width)
		// First line starts after the padded name, so trim leading indent
		firstLine := strings.TrimLeft(wrapped, " ")
		pkgs := dimStyle.Render(firstLine)
		b.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, check, paddedName, pkgs))
	}

	b.WriteString(helpStyle.Render("\nSpace to toggle, Enter to confirm, Esc to go back"))
	return b.String()
}

// --- Agents Step (multi-select) ---

func (m Model) updateAgents(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(AllAgents)-1 {
				m.cursor++
			}
		case " ", "x":
			k := "agent:" + AllAgents[m.cursor].Name
			m.checked[k] = !m.checked[k]
		case "enter":
			m.state.Agents = nil
			for _, a := range AllAgents {
				if m.checked["agent:"+a.Name] {
					m.state.Agents = append(m.state.Agents, a.Name)
				}
			}
			m.step = stepSettings
			m.cursor = 0
		case "esc":
			m.step = stepTools
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) viewAgents() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Step 4/6 — Which agents do you want to enable?"))
	b.WriteString("\n\n")

	for i, agent := range AllAgents {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		check := "[ ]"
		if m.checked["agent:"+agent.Name] {
			check = selectedStyle.Render("[x]")
		}
		paddedName := fmt.Sprintf("%-18s", agent.DisplayName)
		if m.cursor == i {
			paddedName = selectedStyle.Render(paddedName)
		}
		b.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, check, paddedName, dimStyle.Render(agent.Description)))
	}

	b.WriteString(helpStyle.Render("\nSpace to toggle, Enter to confirm, Esc to go back"))
	return b.String()
}

// --- Settings Step ---

var settingsOptions = []struct {
	Key         string
	Label       string
	Description string
}{
	{"setting:auto_update", "Auto-update agents", "Check for new versions on every launch (slows down startup)"},
	{"setting:status_bar", "Status bar", "Show a status bar with version and agent info during sessions"},
}

func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(settingsOptions)-1 {
				m.cursor++
			}
		case " ", "x":
			k := settingsOptions[m.cursor].Key
			m.checked[k] = !m.checked[k]
		case "enter":
			m.state.AutoUpdate = m.checked["setting:auto_update"]
			m.state.StatusBar = m.checked["setting:status_bar"]
			m.step = stepReview
			m.cursor = 0
		case "esc":
			m.step = stepAgents
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) viewSettings() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Step 5/6 — Settings"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Space to toggle. Use 'exitbox rebuild <agent>' to update manually.\n"))
	b.WriteString("\n")

	for i, opt := range settingsOptions {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		check := "[ ]"
		if m.checked[opt.Key] {
			check = selectedStyle.Render("[x]")
		}
		label := opt.Label
		if m.cursor == i {
			label = selectedStyle.Render(label)
		}
		b.WriteString(fmt.Sprintf("%s%s %-25s %s\n", cursor, check, label, dimStyle.Render(opt.Description)))
	}

	b.WriteString(helpStyle.Render("\nSpace to toggle, Enter to confirm, Esc to go back"))
	return b.String()
}

// --- Review Step ---

func (m Model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter", "y":
			m.confirmed = true
			m.step = stepDone
			return m, tea.Quit
		case "esc":
			m.step = stepSettings
			m.cursor = 0
		case "q", "n":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewReview() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Step 6/6 — Review your configuration"))
	b.WriteString("\n\n")

	if len(m.state.Roles) > 0 {
		b.WriteString(fmt.Sprintf("  Roles:      %s\n", successStyle.Render(strings.Join(m.state.Roles, ", "))))
	} else {
		b.WriteString(fmt.Sprintf("  Roles:      %s\n", dimStyle.Render("none")))
	}

	if len(m.state.Languages) > 0 {
		b.WriteString(fmt.Sprintf("  Languages:  %s\n", selectedStyle.Render(strings.Join(m.state.Languages, ", "))))
	} else {
		b.WriteString(fmt.Sprintf("  Languages:  %s\n", dimStyle.Render("none")))
	}

	if len(m.state.ToolCategories) > 0 {
		b.WriteString(fmt.Sprintf("  Tools:      %s\n", selectedStyle.Render(strings.Join(m.state.ToolCategories, ", "))))
	} else {
		b.WriteString(fmt.Sprintf("  Tools:      %s\n", dimStyle.Render("none")))
	}

	if len(m.state.Agents) > 0 {
		names := make([]string, len(m.state.Agents))
		for i, a := range m.state.Agents {
			for _, opt := range AllAgents {
				if opt.Name == a {
					names[i] = opt.DisplayName
					break
				}
			}
		}
		b.WriteString(fmt.Sprintf("  Agents:     %s\n", selectedStyle.Render(strings.Join(names, ", "))))
	} else {
		b.WriteString(fmt.Sprintf("  Agents:     %s\n", dimStyle.Render("none")))
	}

	autoUpdateStr := successStyle.Render("yes")
	if !m.state.AutoUpdate {
		autoUpdateStr = dimStyle.Render("no")
	}
	statusBarStr := successStyle.Render("yes")
	if !m.state.StatusBar {
		statusBarStr = dimStyle.Render("no")
	}
	b.WriteString(fmt.Sprintf("  Auto-update:  %s\n", autoUpdateStr))
	b.WriteString(fmt.Sprintf("  Status bar:   %s\n", statusBarStr))

	profiles := ComputeProfiles(m.state.Roles, m.state.Languages)
	if len(profiles) > 0 {
		b.WriteString(fmt.Sprintf("\n  Default profile: %s\n", selectedStyle.Render(strings.Join(profiles, ", "))))
		b.WriteString(dimStyle.Render("  (installed in all projects; override per-project with 'exitbox run <agent> profile')"))
		b.WriteString("\n")
	}

	packages := ComputePackages(m.state.ToolCategories)
	if len(packages) > 0 {
		// "  Packages:   " = 14 chars indent
		wrapped := wrapWords(packages, "              ", m.width)
		firstLine := strings.TrimLeft(wrapped, " ")
		b.WriteString(fmt.Sprintf("  Packages:   %s\n", dimStyle.Render(firstLine)))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter to confirm, Esc to go back, q to cancel"))
	return b.String()
}
