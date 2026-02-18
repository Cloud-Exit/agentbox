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
	"bytes"
	"sync"
)

// SecretProvider is a function that returns the current map of secrets to redact.
type SecretProvider func() map[string]string

// Redactor filters output to replace known secret values with <redacted>.
type Redactor struct {
	mu       sync.RWMutex
	secrets  map[string]string // value -> key name
	provider SecretProvider    // optional dynamic provider
}

// New creates a new Redactor instance.
func New() *Redactor {
	return &Redactor{
		secrets: make(map[string]string),
	}
}

// NewWithProvider creates a Redactor that fetches secrets dynamically.
func NewWithProvider(provider SecretProvider) *Redactor {
	return &Redactor{
		provider: provider,
	}
}

// AddSecret registers a secret value to be redacted from output.
// The key name is used for logging/debugging purposes only.
func (r *Redactor) AddSecret(value, keyName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value != "" {
		r.secrets[value] = keyName
	}
}

// getSecrets returns the current secrets, using provider if available.
func (r *Redactor) getSecrets() map[string]string {
	if r.provider != nil {
		return r.provider()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.secrets
}

// Filter replaces all known secrets in the input with <redacted>.
func (r *Redactor) Filter(input []byte) []byte {
	secrets := r.getSecrets()

	if len(secrets) == 0 {
		return input
	}

	output := input
	for secret := range secrets {
		if len(secret) == 0 {
			continue
		}
		output = bytes.ReplaceAll(output, []byte(secret), []byte("<redacted>"))
	}
	return output
}

// Clear removes all registered secrets.
func (r *Redactor) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.secrets = make(map[string]string)
}

// Count returns the number of registered secrets.
func (r *Redactor) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.secrets)
}
