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

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/container"
	"github.com/cloud-exit/exitbox/internal/ui"
)

// BuildSquid builds the exitbox-squid proxy image.
func BuildSquid(ctx context.Context, rt container.Runtime, force bool) error {
	imageName := "exitbox-squid"
	cmd := container.Cmd(rt)

	if !force && rt.ImageExists(imageName) {
		v, _ := rt.ImageInspect(imageName, `{{index .Config.Labels "exitbox.version"}}`)
		if v == Version {
			return nil
		}
		ui.Infof("Squid image version mismatch (%s != %s). Rebuilding...", v, Version)
	}

	ui.Info("Building Squid proxy image...")

	buildCtx := filepath.Join(config.Cache, "build-squid")
	os.MkdirAll(buildCtx, 0755)

	dockerfile := fmt.Sprintf(`FROM alpine:3.21
ARG EXITBOX_VERSION
RUN apk add --no-cache squid socat ripgrep python3
RUN mkdir -p /etc/squid
LABEL exitbox.version="%s"
CMD ["squid", "-N", "-d", "1", "-f", "/etc/squid/squid.conf"]
`, Version)

	os.WriteFile(filepath.Join(buildCtx, "Dockerfile"), []byte(dockerfile), 0644)

	args := buildArgs(cmd)
	args = append(args,
		"--build-arg", fmt.Sprintf("EXITBOX_VERSION=%s", Version),
		"-t", imageName,
		buildCtx,
	)

	if err := container.BuildInteractive(rt, args); err != nil {
		return fmt.Errorf("failed to build Squid image: %w", err)
	}

	return nil
}
