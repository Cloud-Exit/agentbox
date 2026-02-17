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

package kvstore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(Options{InMemory: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

func TestSetAndGet(t *testing.T) {
	s := openTestStore(t)

	if err := s.Set([]byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("Get = %q, want %q", got, "v1")
	}
}

func TestGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.Get([]byte("missing"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	s := openTestStore(t)

	if err := s.Set([]byte("del"), []byte("val")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Delete([]byte("del")); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get([]byte("del"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
}

func TestSetWithTTL(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetWithTTL([]byte("ttl"), []byte("expires"), 1*time.Hour); err != nil {
		t.Fatalf("SetWithTTL: %v", err)
	}

	got, err := s.Get([]byte("ttl"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "expires" {
		t.Errorf("Get = %q, want %q", got, "expires")
	}
}

func TestIterate(t *testing.T) {
	s := openTestStore(t)

	for _, kv := range []struct{ k, v string }{
		{"req:ws1:001", "a"},
		{"req:ws1:002", "b"},
		{"req:ws2:001", "c"},
		{"other:key", "d"},
	} {
		if err := s.Set([]byte(kv.k), []byte(kv.v)); err != nil {
			t.Fatalf("Set(%s): %v", kv.k, err)
		}
	}

	var keys []string
	err := s.Iterate([]byte("req:ws1:"), func(key, value []byte) error {
		keys = append(keys, string(key))
		return nil
	})
	if err != nil {
		t.Fatalf("Iterate: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Iterate(req:ws1:) returned %d keys, want 2: %v", len(keys), keys)
	}
}

func TestIterateAll(t *testing.T) {
	s := openTestStore(t)

	for _, kv := range []struct{ k, v string }{
		{"req:a", "1"},
		{"req:b", "2"},
		{"req:c", "3"},
	} {
		if err := s.Set([]byte(kv.k), []byte(kv.v)); err != nil {
			t.Fatalf("Set(%s): %v", kv.k, err)
		}
	}

	var count int
	err := s.Iterate([]byte("req:"), func(key, value []byte) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("Iterate: %v", err)
	}
	if count != 3 {
		t.Errorf("Iterate(req:) count = %d, want 3", count)
	}
}

func TestIterateStopsOnError(t *testing.T) {
	s := openTestStore(t)

	if err := s.Set([]byte("a"), []byte("1")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Set([]byte("b"), []byte("2")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	stopErr := errors.New("stop")
	var count int
	err := s.Iterate(nil, func(key, value []byte) error {
		count++
		return stopErr
	})
	if !errors.Is(err, stopErr) {
		t.Errorf("Iterate error = %v, want stopErr", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (should stop after first)", count)
	}
}

func TestLargeDataExpansion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large data test in short mode")
	}

	dir := t.TempDir()
	s, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Write ~10 MB of data: 1000 keys × 10 KB random (incompressible) values.
	const numKeys = 1000
	const valSize = 10 * 1024 // 10 KB
	value := make([]byte, valSize)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	for i := 0; i < numKeys; i++ {
		key := []byte(fmt.Sprintf("session:agent:proj:%04d", i))
		if err := s.Set(key, value); err != nil {
			t.Fatalf("Set key %d: %v", i, err)
		}
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Measure on-disk size.
	var totalBytes int64
	err = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		totalBytes += info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}

	totalMB := float64(totalBytes) / (1024 * 1024)
	t.Logf("Stored %d keys × %d KB = %.1f MB logical data", numKeys, valSize/1024, float64(numKeys*valSize)/(1024*1024))
	t.Logf("On-disk size: %.1f MB (%d bytes) in %s", totalMB, totalBytes, dir)

	// The store must hold at least 10 MB of data (not capped at 1 MB).
	if totalBytes < 10*1024*1024 {
		t.Errorf("on-disk size = %d bytes (%.1f MB), expected >= 10 MB", totalBytes, totalMB)
	}

	// Reopen and verify a sample of keys survived.
	s2, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer func() {
		if err := s2.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	for _, i := range []int{0, 499, 999} {
		key := []byte(fmt.Sprintf("session:agent:proj:%04d", i))
		got, err := s2.Get(key)
		if err != nil {
			t.Errorf("Get key %d after reopen: %v", i, err)
			continue
		}
		if len(got) != valSize {
			t.Errorf("key %d: value len = %d, want %d", i, len(got), valSize)
		}
	}
}

func TestOpenDisk(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Set([]byte("persist"), []byte("yes")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen and read.
	s2, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer func() {
		if err := s2.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	got, err := s2.Get([]byte("persist"))
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if string(got) != "yes" {
		t.Errorf("Get = %q, want %q", got, "yes")
	}
}
