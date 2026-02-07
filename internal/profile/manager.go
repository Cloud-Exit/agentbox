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

package profile

import (
	"os"
	"path/filepath"

	"github.com/cloud-exit/exitbox/internal/config"
)

// ProjectProfilesPath returns the path to profiles.yaml for an agent in a project.
func ProjectProfilesPath(agent, projectDir string) string {
	return filepath.Join(projectDir, ".exitbox", agent, "profiles.yaml")
}

// GetProjectProfiles returns the current profiles for an agent in a project.
func GetProjectProfiles(agent, projectDir string) ([]string, error) {
	path := ProjectProfilesPath(agent, projectDir)
	pp, err := config.LoadProjectProfiles(path)
	if err != nil {
		return nil, nil // no profiles configured
	}
	return pp.Profiles, nil
}

// AddProjectProfile adds a profile for an agent in a project.
func AddProjectProfile(agent, projectDir, name string) error {
	if !Exists(name) {
		return &InvalidProfileError{Name: name}
	}

	path := ProjectProfilesPath(agent, projectDir)
	os.MkdirAll(filepath.Dir(path), 0755)

	pp, _ := config.LoadProjectProfiles(path)
	if pp == nil {
		pp = &config.ProjectProfiles{}
	}

	// Check if already present
	for _, p := range pp.Profiles {
		if p == name {
			return nil
		}
	}

	pp.Profiles = append(pp.Profiles, name)
	return config.SaveProjectProfiles(pp, path)
}

// RemoveProjectProfile removes a profile for an agent in a project.
func RemoveProjectProfile(agent, projectDir, name string) error {
	path := ProjectProfilesPath(agent, projectDir)
	pp, err := config.LoadProjectProfiles(path)
	if err != nil {
		return nil
	}

	var filtered []string
	for _, p := range pp.Profiles {
		if p != name {
			filtered = append(filtered, p)
		}
	}
	pp.Profiles = filtered
	return config.SaveProjectProfiles(pp, path)
}

// InvalidProfileError is returned when an invalid profile name is used.
type InvalidProfileError struct {
	Name string
}

func (e *InvalidProfileError) Error() string {
	return "unknown profile: " + e.Name + ". Run 'exitbox <agent> profile list' for valid names."
}
