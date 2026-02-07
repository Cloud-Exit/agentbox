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
)

// Step identifies the current wizard step.
type Step int

const (
	stepWelcome Step = iota
	stepRole
	stepLanguage
	stepTools
	stepAgents
	stepReview
	stepDone
)

// State holds accumulated user selections across wizard steps.
type State struct {
	Roles          []string
	Languages      []string
	ToolCategories []string
	Agents         []string
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

// NewModel creates a new wizard model.
func NewModel() Model {
	return Model{
		step:    stepWelcome,
		checked: make(map[string]bool),
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
	b.WriteString(titleStyle.Render("Step 1/5 — What kind of developer are you?"))
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
		name := role.Name
		if m.cursor == i {
			name = selectedStyle.Render(name)
		}
		b.WriteString(fmt.Sprintf("%s%s %-15s %s\n", cursor, check, name, dimStyle.Render(role.Description)))
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
	b.WriteString(titleStyle.Render("Step 2/5 — Which languages do you use?"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Pre-selected based on your role. Space to toggle.\n"))
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
		name := lang.Name
		if m.cursor == i {
			name = selectedStyle.Render(name)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, name))
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
	b.WriteString(titleStyle.Render("Step 3/5 — Which tool categories do you need?"))
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
		name := cat.Name
		if m.cursor == i {
			name = selectedStyle.Render(name)
		}
		pkgs := dimStyle.Render(strings.Join(cat.Packages, ", "))
		b.WriteString(fmt.Sprintf("%s%s %-15s %s\n", cursor, check, name, pkgs))
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
			m.step = stepReview
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
	b.WriteString(titleStyle.Render("Step 4/5 — Which agents do you want to enable?"))
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
		name := agent.DisplayName
		if m.cursor == i {
			name = selectedStyle.Render(name)
		}
		b.WriteString(fmt.Sprintf("%s%s %-18s %s\n", cursor, check, name, dimStyle.Render(agent.Description)))
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
			m.step = stepAgents
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
	b.WriteString(titleStyle.Render("Step 5/5 — Review your configuration"))
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

	profiles := ComputeProfiles(m.state.Roles, m.state.Languages)
	if len(profiles) > 0 {
		b.WriteString(fmt.Sprintf("\n  Profiles:   %s\n", dimStyle.Render(strings.Join(profiles, ", "))))
	}

	packages := ComputePackages(m.state.ToolCategories)
	if len(packages) > 0 {
		b.WriteString(fmt.Sprintf("  Packages:   %s\n", dimStyle.Render(strings.Join(packages, ", "))))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter to confirm, Esc to go back, q to cancel"))
	return b.String()
}
