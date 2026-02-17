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

package generate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// ServerConfig holds the user-provided server configuration.
type ServerConfig struct {
	ProviderID   string // slug, e.g. "local"
	ProviderName string // display name, e.g. "Qwen3.5-397B (local)"
	BaseURL      string // e.g. "http://localhost:8080/v1"
	APIKey       string // may be empty or vault reference
	ModelID      string // e.g. "qwen3.5-397b"
	ModelName    string // e.g. "Qwen3.5-397B-A17B"
	VaultKeyName string // non-empty when key is stored in vault
}

// ModelInfo describes a model returned by the /models endpoint.
type ModelInfo struct {
	ID string `json:"id"`
}

// modelsResponse is the OpenAI-compatible /models response.
type modelsResponse struct {
	Data []ModelInfo `json:"data"`
}

// TestServer sends GET {baseURL}/models and returns discovered models.
func TestServer(baseURL, apiKey string) ([]ModelInfo, error) {
	url := baseURL + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("server returned no models")
	}

	return result.Data, nil
}

// GenerateOpenCode produces an OpenCode config map.
func GenerateOpenCode(cfg ServerConfig) map[string]interface{} {
	return map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]interface{}{
			cfg.ProviderID: map[string]interface{}{
				"npm":  "@ai-sdk/openai-compatible",
				"name": cfg.ProviderName,
				"options": map[string]interface{}{
					"baseURL": cfg.BaseURL,
				},
				"models": map[string]interface{}{
					cfg.ModelID: map[string]interface{}{
						"name": cfg.ModelName,
					},
				},
			},
		},
		"model": cfg.ProviderID + "/" + cfg.ModelID,
	}
}

// GenerateClaude produces a Claude Code settings.json config map.
func GenerateClaude(cfg ServerConfig) map[string]interface{} {
	m := map[string]interface{}{
		"apiBaseUrl": cfg.BaseURL,
		"model":      cfg.ModelID,
	}
	if cfg.APIKey != "" && cfg.VaultKeyName == "" {
		m["apiKey"] = cfg.APIKey
	}
	return m
}

// GenerateCodex produces a Codex config.json config map.
func GenerateCodex(cfg ServerConfig) map[string]interface{} {
	return map[string]interface{}{
		"model":    cfg.ProviderID + "/" + cfg.ModelID,
		"provider": cfg.BaseURL,
	}
}

// MergeJSON deep-merges generated values into existing, returning a new map.
// Generated values override existing at leaf level; existing keys not in
// generated are preserved.
func MergeJSON(existing, generated map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy existing keys.
	for k, v := range existing {
		result[k] = v
	}

	// Merge generated keys.
	for k, genVal := range generated {
		existVal, exists := result[k]
		if !exists {
			result[k] = genVal
			continue
		}

		// If both are maps, recurse.
		existMap, existIsMap := existVal.(map[string]interface{})
		genMap, genIsMap := genVal.(map[string]interface{})
		if existIsMap && genIsMap {
			result[k] = MergeJSON(existMap, genMap)
		} else {
			result[k] = genVal
		}
	}

	return result
}

// WriteConfig reads an existing config file (if present), deep-merges the
// generated data into it, and writes the result back with 2-space indentation.
func WriteConfig(path string, data map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	existing := make(map[string]interface{})
	if raw, err := os.ReadFile(path); err == nil {
		if jsonErr := json.Unmarshal(raw, &existing); jsonErr != nil {
			// Existing file is not valid JSON; overwrite it.
			existing = make(map[string]interface{})
		}
	}

	merged := MergeJSON(existing, data)

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	out = append(out, '\n')

	return os.WriteFile(path, out, 0644)
}

// ConfigPath returns the config file path for an agent within a workspace profile dir.
func ConfigPath(agentDir, agentName string) string {
	switch agentName {
	case "opencode":
		return filepath.Join(agentDir, ".config", "opencode", "opencode.json")
	case "claude":
		return filepath.Join(agentDir, ".claude", "settings.json")
	case "codex":
		return filepath.Join(agentDir, ".codex", "config.json")
	}
	return ""
}

// ExtractConfigHosts reads the agent config file and extracts any non-local
// server hosts (with port) that the agent is configured to connect to.
func ExtractConfigHosts(agentDir, agentName string) []string {
	path := ConfigPath(agentDir, agentName)
	if path == "" {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}

	var urls []string
	switch agentName {
	case "opencode":
		urls = extractOpenCodeURLs(data)
	case "claude":
		urls = extractClaudeURLs(data)
	case "codex":
		urls = extractCodexURLs(data)
	}

	return deduplicateHosts(urls)
}

// extractOpenCodeURLs walks provider.*.options.baseURL in an OpenCode config.
func extractOpenCodeURLs(data map[string]interface{}) []string {
	providers, ok := data["provider"].(map[string]interface{})
	if !ok {
		return nil
	}
	var urls []string
	for _, pv := range providers {
		provider, ok := pv.(map[string]interface{})
		if !ok {
			continue
		}
		opts, ok := provider["options"].(map[string]interface{})
		if !ok {
			continue
		}
		if baseURL, ok := opts["baseURL"].(string); ok && baseURL != "" {
			urls = append(urls, baseURL)
		}
	}
	return urls
}

// extractClaudeURLs reads the top-level apiBaseUrl from a Claude config.
func extractClaudeURLs(data map[string]interface{}) []string {
	if baseURL, ok := data["apiBaseUrl"].(string); ok && baseURL != "" {
		return []string{baseURL}
	}
	return nil
}

// extractCodexURLs reads the top-level provider URL from a Codex config.
func extractCodexURLs(data map[string]interface{}) []string {
	if provider, ok := data["provider"].(string); ok && provider != "" {
		return []string{provider}
	}
	return nil
}

// deduplicateHosts extracts host:port from URLs and returns a unique list,
// skipping localhost and 127.0.0.1.
func deduplicateHosts(urls []string) []string {
	seen := make(map[string]struct{})
	var hosts []string
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := u.Host
		if host == "" {
			continue
		}
		// Skip loopback addresses.
		hostname := u.Hostname()
		if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}
	return hosts
}
