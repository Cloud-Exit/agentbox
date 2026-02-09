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
	"runtime"
	"time"

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/cloud-exit/exitbox/static"
)

// Version is set from cmd package.
var Version = "3.2.0"

// SessionTools are extra packages requested via --tools for this build only.
var SessionTools []string

// ForceRebuild forces image rebuilds (from --update flag only).
var ForceRebuild bool

// AutoUpdate enables checking for new agent versions on launch.
var AutoUpdate bool

// BuildBase builds the exitbox-base image.
func BuildBase(ctx context.Context, rt container.Runtime, force bool) error {
	imageName := "exitbox-base"
	cmd := container.Cmd(rt)

	if !force && !ForceRebuild && rt.ImageExists(imageName) {
		v, _ := rt.ImageInspect(imageName, `{{index .Config.Labels "exitbox.version"}}`)
		if v == Version {
			return nil
		}
		ui.Infof("Base image version mismatch (%s != %s). Rebuilding...", v, Version)
	}

	ui.Infof("Building base image with %s...", cmd)

	buildCtx := filepath.Join(config.Cache, "build")
	_ = os.MkdirAll(buildCtx, 0755)

	// Copy build files from embedded assets
	dockerfilePath := filepath.Join(buildCtx, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, static.DockerfileBase, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	if err := os.WriteFile(filepath.Join(buildCtx, "docker-entrypoint"), static.DockerEntrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write entrypoint: %w", err)
	}
	if err := os.WriteFile(filepath.Join(buildCtx, ".dockerignore"), static.Dockerignore, 0644); err != nil {
		return fmt.Errorf("failed to write .dockerignore: %w", err)
	}

	// Write pre-built exitbox-allow binary for the container's architecture.
	var allowBin []byte
	switch runtime.GOARCH {
	case "arm64":
		allowBin = static.ExitboxAllowArm64
	default:
		allowBin = static.ExitboxAllowAmd64
	}
	if err := os.WriteFile(filepath.Join(buildCtx, "exitbox-allow"), allowBin, 0755); err != nil {
		ui.Warnf("Failed to write exitbox-allow: %v", err)
	} else {
		extra := "\n# IPC client\nCOPY exitbox-allow /usr/local/bin/exitbox-allow\n"
		if err := appendToFile(dockerfilePath, extra); err != nil {
			ui.Warnf("Failed to append exitbox-allow to Dockerfile: %v", err)
		}
	}

	args := buildArgs(cmd)
	args = append(args,
		"--build-arg", fmt.Sprintf("USER_ID=%d", os.Getuid()),
		"--build-arg", fmt.Sprintf("GROUP_ID=%d", os.Getgid()),
		"--build-arg", "USERNAME=user",
		"--build-arg", fmt.Sprintf("EXITBOX_VERSION=%s", Version),
		"-t", imageName,
		"-f", filepath.Join(buildCtx, "Dockerfile"),
		buildCtx,
	)

	if err := buildImage(rt, args, "Building base image..."); err != nil {
		return fmt.Errorf("failed to build base image: %w", err)
	}

	ui.Success("Base image built")
	return nil
}

// buildImage runs a container build, using a spinner in quiet mode or
// full output in verbose mode. On failure in quiet mode, the captured
// build output is printed to stderr.
func buildImage(rt container.Runtime, args []string, label string) error {
	if ui.Verbose {
		start := time.Now()
		err := container.BuildInteractive(rt, args)
		ui.Infof("Build took %s", formatDuration(time.Since(start)))
		return err
	}
	spin := ui.NewSpinner(label)
	spin.Start()
	output, err := container.BuildQuiet(rt, args)
	elapsed := spin.Stop()
	if err != nil {
		fmt.Fprint(os.Stderr, output)
		return err
	}
	ui.Infof("Build took %s", formatDuration(elapsed))
	return nil
}

// formatDuration formats a duration as a human-friendly string (e.g., "12s", "1m 23s").
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func buildArgs(cmd string) []string {
	var args []string
	if cmd == "podman" {
		args = append(args, "--layers", "--pull=newer")
	} else {
		os.Setenv("DOCKER_BUILDKIT", "1")
		cacheDir := filepath.Join(config.Cache, "buildx")
		_ = os.MkdirAll(cacheDir, 0755)
		args = append(args,
			"--progress=auto",
			"--cache-from", "type=local,src="+cacheDir,
			"--cache-to", "type=local,dest="+cacheDir+",mode=max",
		)
	}
	return args
}

