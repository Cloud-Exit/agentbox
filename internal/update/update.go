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

package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const repoOwner = "Cloud-Exit"
const repoName = "ExitBox"

// githubRelease is the subset of the GitHub releases API response we need.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// GetLatestVersion fetches the latest release tag from GitHub.
// Returns the version string without a leading "v" (e.g. "3.3.0").
func GetLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parsing release JSON: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")
	if version == "" {
		return "", fmt.Errorf("empty tag_name in release")
	}
	return version, nil
}

// IsNewer returns true if latest is a newer semver than current.
// Returns false if either version is unparseable (e.g. "dev").
func IsNewer(current, latest string) bool {
	curParts, curOk := parseSemver(current)
	latParts, latOk := parseSemver(latest)
	if !curOk || !latOk {
		return false
	}

	for i := 0; i < 3; i++ {
		if latParts[i] > curParts[i] {
			return true
		}
		if latParts[i] < curParts[i] {
			return false
		}
	}
	return false
}

// parseSemver splits a version string like "3.2.0" into [3, 2, 0].
// Returns false if the format is invalid.
func parseSemver(v string) ([3]int, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var result [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		result[i] = n
	}
	return result, true
}

// BinaryURL returns the download URL for a given version, OS, and architecture.
func BinaryURL(version string) string {
	name := fmt.Sprintf("exitbox-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s",
		repoOwner, repoName, version, name)
}

// BinaryURLFor returns the download URL for a given version with explicit OS/arch.
// Useful for testing.
func BinaryURLFor(version, goos, goarch string) string {
	name := fmt.Sprintf("exitbox-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s",
		repoOwner, repoName, version, name)
}

// DownloadAndReplace downloads the binary from url and replaces the current executable.
func DownloadAndReplace(url string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current executable: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Write to a temp file next to the current binary.
	dir := filepath.Dir(exePath)
	tmpFile, err := os.CreateTemp(dir, "exitbox-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing download: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Set executable permissions.
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic-ish replace: rename current -> .old, rename tmp -> current, remove .old.
	oldPath := exePath + ".old"
	if err := os.Rename(exePath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		// Try to restore the old binary.
		_ = os.Rename(oldPath, exePath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	// Best-effort cleanup of the old binary.
	_ = os.Remove(oldPath)

	return nil
}

// CheckResult holds the result of an async update check.
type CheckResult struct {
	Available bool
	Latest    string
}

// AsyncCheck runs a version check in the background. The returned channel
// receives the result when done. The check respects the given timeout.
func AsyncCheck(current string, timeout time.Duration) <-chan CheckResult {
	ch := make(chan CheckResult, 1)
	go func() {
		done := make(chan struct{})
		var result CheckResult
		go func() {
			latest, err := GetLatestVersion()
			if err == nil && IsNewer(current, latest) {
				result = CheckResult{Available: true, Latest: latest}
			}
			close(done)
		}()

		select {
		case <-done:
			ch <- result
		case <-time.After(timeout):
			ch <- CheckResult{}
		}
		close(ch)
	}()
	return ch
}

// PromptViaTmux shows a tmux popup inside a container asking if the user
// wants to update. containerCmd is "podman" or "docker", containerName is
// the running container's name. Returns true if the user approved.
func PromptViaTmux(containerCmd, containerName, currentVersion, latestVersion string) (bool, error) {
	script := fmt.Sprintf(
		`printf '\n  \033[1;33m[ExitBox]\033[0m Update available!\n\n  Current: v%s\n  Latest:  v%s\n\n  Update after session? [y/N]: '; read ans; [ "$ans" = "y" ] || [ "$ans" = "yes" ]`,
		currentVersion, latestVersion,
	)

	c := exec.Command(containerCmd, "exec", containerName,
		"tmux", "display-popup", "-E", "-w", "50", "-h", "10",
		"sh", "-c", script,
	)

	var stderr bytes.Buffer
	c.Stderr = &stderr
	err := c.Run()

	if err == nil {
		return true, nil // exit 0 = approved
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if stderr.Len() == 0 {
			return false, nil // user dismissed or denied
		}
		return false, fmt.Errorf("popup failed (exit %d): %s", exitErr.ExitCode(), stderr.String())
	}

	return false, fmt.Errorf("popup exec failed: %w", err)
}

// RunUpdatePopup starts a background goroutine that checks for updates and,
// if one is available, waits for the container's tmux to be ready then shows
// a popup. If the user approves, wantUpdate is set to 1 atomically.
// latestVersion is written when an update is found, regardless of approval.
func RunUpdatePopup(containerCmd, containerName, currentVersion string, wantUpdate *atomic.Int32, latestVersion *atomic.Value) {
	go func() {
		// Check for update with a 5s timeout.
		result := <-AsyncCheck(currentVersion, 5*time.Second)
		if !result.Available {
			return
		}
		latestVersion.Store(result.Latest)

		// Wait for tmux to be ready inside the container (poll for up to 30s).
		ready := false
		for i := 0; i < 15; i++ {
			time.Sleep(2 * time.Second)
			c := exec.Command(containerCmd, "exec", containerName,
				"tmux", "list-sessions",
			)
			if c.Run() == nil {
				ready = true
				break
			}
		}
		if !ready {
			return
		}

		// Small delay so the agent has time to render its initial screen.
		time.Sleep(2 * time.Second)

		approved, err := PromptViaTmux(containerCmd, containerName, currentVersion, result.Latest)
		if err == nil && approved {
			wantUpdate.Store(1)
		}
	}()
}
