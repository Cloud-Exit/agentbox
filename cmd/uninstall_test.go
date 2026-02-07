// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/cloud-exit/exitbox/internal/container"
)

// testRuntime implements container.Runtime for testing.
type testRuntime struct {
	images  map[string]bool
	removed []string
}

func newTestRuntime(images ...string) *testRuntime {
	r := &testRuntime{images: make(map[string]bool)}
	for _, img := range images {
		r.images[img] = true
	}
	return r
}

func (r *testRuntime) Name() string                                        { return "docker" }
func (r *testRuntime) Build(_ context.Context, _ []string) error           { return nil }
func (r *testRuntime) Run(_ context.Context, _ []string) (int, error)      { return 0, nil }
func (r *testRuntime) Exec(_ context.Context, _ string, _ []string) error  { return nil }
func (r *testRuntime) ImageExists(img string) bool                         { return r.images[img] }
func (r *testRuntime) ImageInspect(_, _ string) (string, error)            { return "", nil }

func (r *testRuntime) ImageList(filter string) ([]string, error) {
	// Simple glob matching: support "exitbox-*" and "exitbox-claude-*" patterns
	prefix := strings.TrimSuffix(filter, "*")
	var result []string
	for img := range r.images {
		if strings.HasPrefix(img, prefix) {
			result = append(result, img)
		}
	}
	return result, nil
}

func (r *testRuntime) ImageRemove(img string) error {
	r.removed = append(r.removed, img)
	delete(r.images, img)
	return nil
}

func (r *testRuntime) PS(_, _ string) ([]string, error)       { return nil, nil }
func (r *testRuntime) Stop(_ string) error                    { return nil }
func (r *testRuntime) Remove(_ string) error                  { return nil }
func (r *testRuntime) NetworkCreate(_ string, _ bool) error   { return nil }
func (r *testRuntime) NetworkExists(_ string) bool            { return false }
func (r *testRuntime) NetworkConnect(_, _ string) error       { return nil }
func (r *testRuntime) NetworkInspect(_, _ string) (string, error) { return "", nil }
func (r *testRuntime) IsRootless() bool                       { return false }

// Verify testRuntime implements container.Runtime.
var _ container.Runtime = (*testRuntime)(nil)

func TestRemoveAgentImages(t *testing.T) {
	rt := newTestRuntime(
		"exitbox-claude-core",
		"exitbox-claude-project1",
		"exitbox-claude-project2",
		"exitbox-codex-core",
	)

	removeAgentImages(rt, "claude")

	// All claude images should be removed
	if len(rt.removed) < 3 {
		t.Errorf("expected at least 3 images removed, got %d: %v", len(rt.removed), rt.removed)
	}

	// codex image should still exist
	if !rt.images["exitbox-codex-core"] {
		t.Error("codex image should not have been removed")
	}
}

func TestCleanImagesAll(t *testing.T) {
	rt := newTestRuntime(
		"exitbox-claude-core",
		"exitbox-codex-core",
		"exitbox-base",
		"exitbox-squid",
		"exitbox-claude-project1",
	)

	cleanImages(rt, "all")

	if len(rt.images) != 0 {
		t.Errorf("expected all images removed, got %d remaining: %v", len(rt.images), rt.images)
	}
}
