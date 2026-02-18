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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTestServer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/models" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"model-a"},{"id":"model-b"}]}`))
		}))
		defer srv.Close()

		models, err := TestServer(srv.URL, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(models))
		}
		if models[0].ID != "model-a" {
			t.Errorf("expected model-a, got %s", models[0].ID)
		}
	})

	t.Run("with api key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-key" {
				t.Errorf("expected Bearer test-key, got %s", auth)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"model-x"}]}`))
		}))
		defer srv.Close()

		models, err := TestServer(srv.URL, "test-key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 1 || models[0].ID != "model-x" {
			t.Errorf("unexpected models: %v", models)
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		}))
		defer srv.Close()

		_, err := TestServer(srv.URL, "")
		if err == nil {
			t.Fatal("expected error for 500 response")
		}
	})

	t.Run("empty models", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[]}`))
		}))
		defer srv.Close()

		_, err := TestServer(srv.URL, "")
		if err == nil {
			t.Fatal("expected error for empty models")
		}
	})
}

func TestGenerateOpenCode(t *testing.T) {
	t.Run("without compaction", func(t *testing.T) {
		cfg := ServerConfig{
			ProviderID:   "local",
			ProviderName: "Local Server",
			BaseURL:      "http://localhost:8080/v1",
			ModelID:      "qwen3",
			ModelName:    "Qwen3",
		}

		result := GenerateOpenCode(cfg)

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if parsed["$schema"] != "https://opencode.ai/config.json" {
			t.Error("missing or wrong $schema")
		}

		providers, ok := parsed["provider"].(map[string]interface{})
		if !ok {
			t.Fatal("missing provider")
		}
		local, ok := providers["local"].(map[string]interface{})
		if !ok {
			t.Fatal("missing local provider")
		}
		if local["name"] != "Local Server" {
			t.Errorf("expected Local Server, got %v", local["name"])
		}

		if parsed["model"] != "local/qwen3" {
			t.Errorf("expected local/qwen3, got %v", parsed["model"])
		}

		if _, ok := parsed["compaction"]; ok {
			t.Error("compaction should not be present when disabled")
		}
	})

	t.Run("with compaction", func(t *testing.T) {
		cfg := ServerConfig{
			ProviderID:   "local",
			ProviderName: "Local Server",
			BaseURL:      "http://localhost:8080/v1",
			ModelID:      "qwen3",
			ModelName:    "Qwen3",
			Compaction:   true,
		}

		result := GenerateOpenCode(cfg)

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		comp, ok := parsed["compaction"].(map[string]interface{})
		if !ok {
			t.Fatal("missing compaction")
		}
		if comp["auto"] != true {
			t.Errorf("expected compaction.auto=true, got %v", comp["auto"])
		}
		if comp["prune"] != true {
			t.Errorf("expected compaction.prune=true, got %v", comp["prune"])
		}
	})
}

func TestGenerateClaude(t *testing.T) {
	t.Run("with api key no vault", func(t *testing.T) {
		cfg := ServerConfig{
			BaseURL: "http://localhost:8080/v1",
			ModelID: "gpt-4",
			APIKey:  "sk-test",
		}
		result := GenerateClaude(cfg)

		if result["apiBaseUrl"] != "http://localhost:8080/v1" {
			t.Errorf("wrong apiBaseUrl: %v", result["apiBaseUrl"])
		}
		if result["model"] != "gpt-4" {
			t.Errorf("wrong model: %v", result["model"])
		}
		if result["apiKey"] != "sk-test" {
			t.Errorf("wrong apiKey: %v", result["apiKey"])
		}
	})

	t.Run("with vault key", func(t *testing.T) {
		cfg := ServerConfig{
			BaseURL:      "http://localhost:8080/v1",
			ModelID:      "gpt-4",
			APIKey:       "sk-test",
			VaultKeyName: "API_KEY",
		}
		result := GenerateClaude(cfg)

		if _, ok := result["apiKey"]; ok {
			t.Error("apiKey should not be set when vault key is used")
		}
	})

	t.Run("no api key", func(t *testing.T) {
		cfg := ServerConfig{
			BaseURL: "http://localhost:8080/v1",
			ModelID: "gpt-4",
		}
		result := GenerateClaude(cfg)

		if _, ok := result["apiKey"]; ok {
			t.Error("apiKey should not be set when empty")
		}
	})
}

func TestGenerateCodex(t *testing.T) {
	cfg := ServerConfig{
		ProviderID: "local",
		BaseURL:    "http://localhost:8080/v1",
		ModelID:    "gpt-4",
	}
	result := GenerateCodex(cfg)

	if result["model"] != "local/gpt-4" {
		t.Errorf("wrong model: %v", result["model"])
	}
	if result["provider"] != "http://localhost:8080/v1" {
		t.Errorf("wrong provider: %v", result["provider"])
	}
}

func TestMergeJSON(t *testing.T) {
	t.Run("basic merge", func(t *testing.T) {
		existing := map[string]interface{}{
			"a": "old",
			"b": "keep",
		}
		generated := map[string]interface{}{
			"a": "new",
			"c": "added",
		}
		result := MergeJSON(existing, generated)

		if result["a"] != "new" {
			t.Errorf("expected new, got %v", result["a"])
		}
		if result["b"] != "keep" {
			t.Errorf("expected keep, got %v", result["b"])
		}
		if result["c"] != "added" {
			t.Errorf("expected added, got %v", result["c"])
		}
	})

	t.Run("deep merge maps", func(t *testing.T) {
		existing := map[string]interface{}{
			"provider": map[string]interface{}{
				"existing-provider": map[string]interface{}{
					"name": "Existing",
				},
			},
		}
		generated := map[string]interface{}{
			"provider": map[string]interface{}{
				"new-provider": map[string]interface{}{
					"name": "New",
				},
			},
		}
		result := MergeJSON(existing, generated)

		providers, ok := result["provider"].(map[string]interface{})
		if !ok {
			t.Fatal("provider should be a map")
		}
		if _, ok := providers["existing-provider"]; !ok {
			t.Error("existing-provider should be preserved")
		}
		if _, ok := providers["new-provider"]; !ok {
			t.Error("new-provider should be added")
		}
	})

	t.Run("scalar overrides map", func(t *testing.T) {
		existing := map[string]interface{}{
			"key": map[string]interface{}{"nested": true},
		}
		generated := map[string]interface{}{
			"key": "scalar",
		}
		result := MergeJSON(existing, generated)
		if result["key"] != "scalar" {
			t.Errorf("expected scalar, got %v", result["key"])
		}
	})

	t.Run("does not mutate inputs", func(t *testing.T) {
		existing := map[string]interface{}{"a": "1"}
		generated := map[string]interface{}{"b": "2"}
		_ = MergeJSON(existing, generated)

		if _, ok := existing["b"]; ok {
			t.Error("existing map was mutated")
		}
		if _, ok := generated["a"]; ok {
			t.Error("generated map was mutated")
		}
	})
}

func TestWriteConfig(t *testing.T) {
	t.Run("new file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "config.json")

		data := map[string]interface{}{
			"key": "value",
		}
		if err := WriteConfig(path, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed["key"] != "value" {
			t.Errorf("expected value, got %v", parsed["key"])
		}
	})

	t.Run("merge with existing", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")

		// Write initial config.
		initial := []byte(`{"existing": "preserved", "override": "old"}`)
		if err := os.WriteFile(path, initial, 0644); err != nil {
			t.Fatalf("failed to write initial: %v", err)
		}

		data := map[string]interface{}{
			"override": "new",
			"added":    "fresh",
		}
		if err := WriteConfig(path, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed["existing"] != "preserved" {
			t.Errorf("expected preserved, got %v", parsed["existing"])
		}
		if parsed["override"] != "new" {
			t.Errorf("expected new, got %v", parsed["override"])
		}
		if parsed["added"] != "fresh" {
			t.Errorf("expected fresh, got %v", parsed["added"])
		}
	})
}

func TestExtractConfigHosts(t *testing.T) {
	t.Run("opencode single provider", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{
			"provider": {
				"local": {
					"options": {
						"baseURL": "http://10.10.10.185:8080/v1"
					}
				}
			}
		}`
		if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "opencode")
		if len(hosts) != 1 || hosts[0] != "10.10.10.185:8080" {
			t.Errorf("expected [10.10.10.185:8080], got %v", hosts)
		}
	})

	t.Run("opencode multiple providers", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{
			"provider": {
				"local": {
					"options": {
						"baseURL": "http://10.10.10.185:8080/v1"
					}
				},
				"remote": {
					"options": {
						"baseURL": "http://192.168.1.100:9090/v1"
					}
				}
			}
		}`
		if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "opencode")
		if len(hosts) != 2 {
			t.Errorf("expected 2 hosts, got %v", hosts)
		}
	})

	t.Run("claude", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{"apiBaseUrl": "http://10.10.10.185:8080/v1", "model": "gpt-4"}`
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "claude")
		if len(hosts) != 1 || hosts[0] != "10.10.10.185:8080" {
			t.Errorf("expected [10.10.10.185:8080], got %v", hosts)
		}
	})

	t.Run("codex", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".codex")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{"provider": "http://10.10.10.185:8080/v1", "model": "local/gpt-4"}`
		if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "codex")
		if len(hosts) != 1 || hosts[0] != "10.10.10.185:8080" {
			t.Errorf("expected [10.10.10.185:8080], got %v", hosts)
		}
	})

	t.Run("skips localhost", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{"apiBaseUrl": "http://localhost:8080/v1"}`
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "claude")
		if len(hosts) != 0 {
			t.Errorf("expected no hosts for localhost, got %v", hosts)
		}
	})

	t.Run("skips 127.0.0.1", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{"apiBaseUrl": "http://127.0.0.1:8080/v1"}`
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "claude")
		if len(hosts) != 0 {
			t.Errorf("expected no hosts for 127.0.0.1, got %v", hosts)
		}
	})

	t.Run("no config file", func(t *testing.T) {
		dir := t.TempDir()
		hosts := ExtractConfigHosts(dir, "claude")
		if len(hosts) != 0 {
			t.Errorf("expected no hosts for missing file, got %v", hosts)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "claude")
		if len(hosts) != 0 {
			t.Errorf("expected no hosts for empty config, got %v", hosts)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "claude")
		if len(hosts) != 0 {
			t.Errorf("expected no hosts for invalid JSON, got %v", hosts)
		}
	})

	t.Run("unknown agent", func(t *testing.T) {
		dir := t.TempDir()
		hosts := ExtractConfigHosts(dir, "unknown")
		if len(hosts) != 0 {
			t.Errorf("expected no hosts for unknown agent, got %v", hosts)
		}
	})

	t.Run("deduplicates same host", func(t *testing.T) {
		dir := t.TempDir()
		configDir := filepath.Join(dir, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		data := `{
			"provider": {
				"a": {"options": {"baseURL": "http://10.10.10.185:8080/v1"}},
				"b": {"options": {"baseURL": "http://10.10.10.185:8080/v2"}}
			}
		}`
		if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}

		hosts := ExtractConfigHosts(dir, "opencode")
		if len(hosts) != 1 {
			t.Errorf("expected 1 deduplicated host, got %v", hosts)
		}
	})
}

func TestConfigPath(t *testing.T) {
	tests := []struct {
		agent    string
		expected string
	}{
		{"opencode", filepath.Join("/base", ".config", "opencode", "opencode.json")},
		{"claude", filepath.Join("/base", ".claude", "settings.json")},
		{"codex", filepath.Join("/base", ".codex", "config.json")},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			got := ConfigPath("/base", tt.agent)
			if got != tt.expected {
				t.Errorf("ConfigPath(%q) = %q, want %q", tt.agent, got, tt.expected)
			}
		})
	}
}
