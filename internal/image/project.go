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

package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/profile"
	proj "github.com/cloud-exit/exitbox/internal/project"
	"github.com/cloud-exit/exitbox/internal/ui"
)

// BuildProject builds the agent project image (with profiles).
func BuildProject(ctx context.Context, rt container.Runtime, agentName, projectDir string) error {
	imageName := proj.ImageName(agentName, projectDir)
	coreImage := fmt.Sprintf("exitbox-%s-core", agentName)
	cmd := container.Cmd(rt)

	// Ensure core image exists
	if err := BuildCore(ctx, rt, agentName, false); err != nil {
		return err
	}

	// Get current profiles
	profiles, _ := profile.GetProjectProfiles(agentName, projectDir)

	// Build composite hash for cache detection
	hashParts := strings.Join(profiles, ",")
	cfg := config.LoadOrDefault()
	if len(cfg.Tools.User) > 0 {
		hashParts += ":" + strings.Join(cfg.Tools.User, ",")
	}

	if rt.ImageExists(imageName) {
		// Check if core is newer
		coreCreated, _ := rt.ImageInspect(coreImage, "{{.Created}}")
		projectCreated, _ := rt.ImageInspect(imageName, "{{.Created}}")
		if coreCreated != "" && projectCreated != "" && coreCreated > projectCreated {
			ui.Info("Core image updated, rebuilding project image...")
		} else {
			cachedHash, _ := rt.ImageInspect(imageName, `{{index .Config.Labels "exitbox.profiles.hash"}}`)
			if cachedHash == hashParts {
				return nil
			}
		}
	}

	ui.Infof("Building %s project image with %s...", agentName, cmd)

	buildCtx := filepath.Join(config.Cache, "build-"+agentName+"-project")
	_ = os.MkdirAll(buildCtx, 0755)

	dockerfilePath := filepath.Join(buildCtx, "Dockerfile")
	var df strings.Builder

	fmt.Fprintf(&df, "FROM %s\n\n", coreImage)

	// Switch to root for package installation (some core images end as non-root)
	df.WriteString("USER root\n\n")

	// Install user tools
	if len(cfg.Tools.User) > 0 {
		fmt.Fprintf(&df, "RUN apk add --no-cache %s\n\n", strings.Join(cfg.Tools.User, " "))
	}

	// Add profile installations
	for _, p := range profiles {
		if !profile.Exists(p) {
			return fmt.Errorf("unknown profile '%s'. Run 'exitbox <agent> profile list' for valid names", p)
		}
		snippet := profile.DockerfileSnippet(p)
		if snippet != "" {
			df.WriteString(snippet)
			df.WriteString("\n")
		}
	}

	// Fix home dir ownership after root package installs
	df.WriteString("RUN chown -R user:user /home/user\n\n")

	// Switch back to non-root user
	df.WriteString("USER user\n")

	// Add label
	fmt.Fprintf(&df, "\nLABEL exitbox.profiles.hash=\"%s\"\n", hashParts)

	_ = os.WriteFile(dockerfilePath, []byte(df.String()), 0644)

	args := buildArgs(cmd)
	args = append(args,
		"-t", imageName,
		"-f", dockerfilePath,
		buildCtx,
	)

	if err := container.BuildInteractive(rt, args); err != nil {
		return fmt.Errorf("failed to build %s project image: %w", agentName, err)
	}

	ui.Successf("%s project image built", agentName)
	return nil
}
