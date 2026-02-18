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

package redactor

import (
	"testing"
)

func TestRedactor_AddSecret(t *testing.T) {
	r := New()
	r.AddSecret("my-secret-value", "MY_SECRET")

	if r.Count() != 1 {
		t.Errorf("expected 1 secret, got %d", r.Count())
	}
}

func TestRedactor_Filter(t *testing.T) {
	r := New()
	secret := "super-secret-token-123"
	r.AddSecret(secret, "API_KEY")

	input := []byte("Authorization: Bearer " + secret + "\n")
	output := string(r.Filter(input))

	expected := "Authorization: Bearer <redacted>\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestRedactor_Filter_MultipleSecrets(t *testing.T) {
	r := New()
	r.AddSecret("secret1", "KEY1")
	r.AddSecret("secret2", "KEY2")

	input := []byte("key1=secret1; key2=secret2")
	output := string(r.Filter(input))

	expected := "key1=<redacted>; key2=<redacted>"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestRedactor_Filter_NoSecrets(t *testing.T) {
	r := New()

	input := []byte("normal output without secrets")
	output := string(r.Filter(input))

	if output != string(input) {
		t.Errorf("expected %q, got %q", input, output)
	}
}

func TestRedactor_Filter_EmptySecret(t *testing.T) {
	r := New()
	r.AddSecret("", "EMPTY_KEY")

	input := []byte("test output")
	output := string(r.Filter(input))

	if output != string(input) {
		t.Errorf("expected %q, got %q", input, output)
	}
}

func TestRedactor_Clear(t *testing.T) {
	r := New()
	r.AddSecret("secret", "KEY")

	if r.Count() != 1 {
		t.Errorf("expected 1 secret before clear, got %d", r.Count())
	}

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("expected 0 secrets after clear, got %d", r.Count())
	}
}

func TestRedactor_Filter_PartialMatch(t *testing.T) {
	r := New()
	r.AddSecret("secret", "KEY")

	input := []byte("mysecretvalue")
	output := string(r.Filter(input))

	// Partial matches ARE replaced - this is expected behavior
	expected := "my<redacted>value"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestRedactor_Filter_Newlines(t *testing.T) {
	r := New()
	r.AddSecret("secret", "KEY")

	input := []byte("line1\nsecret\nline3")
	output := string(r.Filter(input))

	expected := "line1\n<redacted>\nline3"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestRedactor_NewWithProvider(t *testing.T) {
	secrets := make(map[string]string)
	provider := func() map[string]string {
		cp := make(map[string]string, len(secrets))
		for k, v := range secrets {
			cp[k] = v
		}
		return cp
	}

	r := NewWithProvider(provider)

	// Initially no secrets
	input := []byte("test output")
	output := string(r.Filter(input))
	if output != string(input) {
		t.Errorf("expected %q, got %q", input, output)
	}

	// Add a secret dynamically
	secrets["dynamic-secret"] = "DYNAMIC_KEY"

	// Now it should be redacted
	input2 := []byte("Authorization: dynamic-secret")
	output2 := string(r.Filter(input2))
	expected2 := "Authorization: <redacted>"
	if output2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, output2)
	}
}
