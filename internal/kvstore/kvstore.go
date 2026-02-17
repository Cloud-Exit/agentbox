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

// Package kvstore provides a generic BadgerDB key-value store wrapper.
package kvstore

import (
	"errors"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("key not found")

// Options configures a Store.
type Options struct {
	Dir      string // on-disk directory (ignored when InMemory is true)
	InMemory bool   // use in-memory storage (for tests)
	ReadOnly bool   // open in read-only mode (no directory lock acquired)
}

// Store wraps a Badger database.
type Store struct {
	db *badger.DB
}

// Open creates or opens a BadgerDB store. If the WAL is corrupted (e.g. from
// an unclean container shutdown), it automatically recovers by opening in
// write mode first to allow truncation, then re-opening in the requested mode.
func Open(opts Options) (*Store, error) {
	bopts := badgerOptions(opts)

	db, err := badger.Open(bopts)
	if err != nil && !opts.InMemory && needsTruncation(err) {
		// WAL has incomplete entries. Open briefly in write mode so BadgerDB
		// can truncate the corrupted WAL, then re-open in the requested mode.
		recoveryOpts := badgerOptions(Options{Dir: opts.Dir})
		rdb, rerr := badger.Open(recoveryOpts)
		if rerr != nil {
			return nil, err // return original error if recovery fails
		}
		if cerr := rdb.Close(); cerr != nil {
			return nil, cerr
		}
		// Retry the original open.
		db, err = badger.Open(bopts)
	}
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}

	// Run value log GC on open to reclaim space from previous runs.
	if !opts.ReadOnly && !opts.InMemory {
		s.runGC()
	}

	return s, nil
}

// badgerOptions builds BadgerDB options from our Options.
func badgerOptions(opts Options) badger.Options {
	bopts := badger.DefaultOptions(opts.Dir)
	bopts.Logger = nil // suppress badger logs
	if opts.InMemory {
		bopts = badger.DefaultOptions("").WithInMemory(true)
		bopts.Logger = nil
	}
	if opts.ReadOnly {
		bopts = bopts.WithReadOnly(true).WithBypassLockGuard(true)
	}
	return bopts
}

// needsTruncation checks if a BadgerDB open error indicates WAL truncation is needed.
func needsTruncation(err error) bool {
	return strings.Contains(err.Error(), "Log truncate required") ||
		strings.Contains(err.Error(), "MANIFEST has unsupported version")
}

// Get retrieves the value for a key. Returns ErrNotFound if the key does not exist.
func (s *Store) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrNotFound
			}
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	return val, err
}

// Set stores a key-value pair.
func (s *Store) Set(key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// SetWithTTL stores a key-value pair that expires after the given duration.
func (s *Store) SetWithTTL(key, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// Delete removes a key.
func (s *Store) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// Iterate calls fn for every key with the given prefix.
// Iteration stops early if fn returns a non-nil error.
func (s *Store) Iterate(prefix []byte, fn func(key, value []byte) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			if err := fn(k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// RunGC triggers value log garbage collection to reclaim disk space.
// It runs multiple rounds until Badger reports nothing left to collect.
func (s *Store) RunGC() {
	s.runGC()
}

// Close runs value log GC and then closes the underlying database.
func (s *Store) Close() error {
	s.runGC()
	return s.db.Close()
}

// runGC runs value log garbage collection in a loop until no more space
// can be reclaimed. The 0.5 discard ratio means a vlog file is rewritten
// when at least 50% of its space is reclaimable.
func (s *Store) runGC() {
	for {
		if s.db.RunValueLogGC(0.5) != nil {
			return
		}
	}
}
