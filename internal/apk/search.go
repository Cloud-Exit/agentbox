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

// Package apk provides Alpine package index search functionality.
package apk

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloud-exit/exitbox/internal/config"
)

// Package represents an Alpine package entry.
type Package struct {
	Name        string
	Description string
}

// cacheMaxAge is the maximum age of the cached APKINDEX before re-downloading.
const cacheMaxAge = 24 * time.Hour

// repos are the Alpine repositories to search.
var repos = []string{"main", "community"}

// alpineVersion is the Alpine release branch to search.
const alpineVersion = "v3.21"

// indexURL returns the APKINDEX.tar.gz URL for a given repository.
func indexURL(repo string) string {
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/%s/%s/x86_64/APKINDEX.tar.gz", alpineVersion, repo)
}

// cacheDir returns the directory for cached APKINDEX files.
func cacheDir() string {
	return filepath.Join(config.Cache, "apkindex")
}

// cacheFile returns the path to a cached repo index file.
func cacheFile(repo string) string {
	return filepath.Join(cacheDir(), repo+".txt")
}

// isCacheFresh returns true if the cache file exists and is less than cacheMaxAge old.
func isCacheFresh(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < cacheMaxAge
}

// downloadIndex fetches APKINDEX.tar.gz and extracts the APKINDEX text.
func downloadIndex(repo string) ([]byte, error) {
	url := indexURL(repo)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decompressing %s: %w", repo, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar %s: %w", repo, err)
		}
		if hdr.Name == "APKINDEX" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("reading APKINDEX from %s: %w", repo, err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("APKINDEX not found in %s archive", repo)
}

// ensureCache downloads and caches the index for a repo if not fresh.
func ensureCache(repo string) error {
	path := cacheFile(repo)
	if isCacheFresh(path) {
		return nil
	}

	data, err := downloadIndex(repo)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing cache %s: %w", path, err)
	}

	return nil
}

// parseIndex parses the APKINDEX text format into Package entries.
// Entries are separated by blank lines. P: = package name, T: = description.
func parseIndex(data []byte) []Package {
	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	var name, desc string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if name != "" {
				packages = append(packages, Package{Name: name, Description: desc})
			}
			name = ""
			desc = ""
			continue
		}
		if strings.HasPrefix(line, "P:") {
			name = strings.TrimPrefix(line, "P:")
		} else if strings.HasPrefix(line, "T:") {
			desc = strings.TrimPrefix(line, "T:")
		}
	}
	// Handle last entry without trailing blank line
	if name != "" {
		packages = append(packages, Package{Name: name, Description: desc})
	}

	return packages
}

// LoadIndex downloads (if needed) and parses the full Alpine package index.
func LoadIndex() ([]Package, error) {
	var all []Package
	for _, repo := range repos {
		if err := ensureCache(repo); err != nil {
			return nil, fmt.Errorf("loading %s index: %w", repo, err)
		}

		data, err := os.ReadFile(cacheFile(repo))
		if err != nil {
			return nil, fmt.Errorf("reading cache for %s: %w", repo, err)
		}

		all = append(all, parseIndex(data)...)
	}

	// Deduplicate by name (main takes precedence over community)
	seen := make(map[string]bool, len(all))
	deduped := make([]Package, 0, len(all))
	for _, p := range all {
		if !seen[p.Name] {
			seen[p.Name] = true
			deduped = append(deduped, p)
		}
	}

	return deduped, nil
}

// Search filters packages by case-insensitive substring match on name and description.
// Returns at most maxResults results.
func Search(index []Package, query string, maxResults int) []Package {
	if query == "" {
		return nil
	}

	q := strings.ToLower(query)
	var results []Package

	// Prioritize name matches first, then description-only matches
	var descOnly []Package
	for _, p := range index {
		if len(results)+len(descOnly) >= maxResults*2 {
			break
		}
		nameLower := strings.ToLower(p.Name)
		if strings.Contains(nameLower, q) {
			results = append(results, p)
		} else if strings.Contains(strings.ToLower(p.Description), q) {
			descOnly = append(descOnly, p)
		}
	}

	results = append(results, descOnly...)
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results
}

// FetchAndSearch handles download/cache/parse/search in one call.
// It loads the index if needed and searches for the query.
func FetchAndSearch(query string) ([]Package, error) {
	index, err := LoadIndex()
	if err != nil {
		return nil, err
	}
	return Search(index, query, 50), nil
}
