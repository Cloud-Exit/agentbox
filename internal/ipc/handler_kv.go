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

package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/kvstore"
)

// KVHandlerConfig holds dependencies for KV IPC handlers.
type KVHandlerConfig struct {
	WorkspaceName string
	// OpenFunc overrides kvstore.Open for testing.
	OpenFunc func(opts kvstore.Options) (*kvstore.Store, error)
}

// openKVForRequest opens the KV store for a single request.
// The caller must close the returned store.
func openKVForRequest(workspace string, openFn func(kvstore.Options) (*kvstore.Store, error)) (*kvstore.Store, error) {
	if openFn == nil {
		openFn = kvstore.Open
	}
	dir := config.KVDir(workspace)
	store, err := openFn(kvstore.Options{Dir: dir})
	if err != nil {
		return nil, fmt.Errorf("open kv store: %w", err)
	}
	return store, nil
}

// NewKVGetHandler returns a HandlerFunc for "kv_get" requests.
func NewKVGetHandler(cfg KVHandlerConfig) HandlerFunc {
	return func(req *Request) (interface{}, error) {
		var payload KVGetRequest
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return KVGetResponse{Error: "invalid payload"}, nil
		}

		key := strings.TrimSpace(payload.Key)
		if key == "" {
			return KVGetResponse{Error: "empty key"}, nil
		}

		store, err := openKVForRequest(cfg.WorkspaceName, cfg.OpenFunc)
		if err != nil {
			return KVGetResponse{Error: err.Error()}, nil
		}
		defer store.Close()

		val, err := store.Get([]byte(key))
		if err != nil {
			if errors.Is(err, kvstore.ErrNotFound) {
				return KVGetResponse{Found: false}, nil
			}
			return KVGetResponse{Error: fmt.Sprintf("get: %v", err)}, nil
		}

		return KVGetResponse{Value: string(val), Found: true}, nil
	}
}

// NewKVSetHandler returns a HandlerFunc for "kv_set" requests.
func NewKVSetHandler(cfg KVHandlerConfig) HandlerFunc {
	return func(req *Request) (interface{}, error) {
		var payload KVSetRequest
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return KVSetResponse{Error: "invalid payload"}, nil
		}

		key := strings.TrimSpace(payload.Key)
		if key == "" {
			return KVSetResponse{Error: "empty key"}, nil
		}

		store, err := openKVForRequest(cfg.WorkspaceName, cfg.OpenFunc)
		if err != nil {
			return KVSetResponse{Error: err.Error()}, nil
		}
		defer store.Close()

		if err := store.Set([]byte(key), []byte(payload.Value)); err != nil {
			return KVSetResponse{Error: fmt.Sprintf("set: %v", err)}, nil
		}

		return KVSetResponse{}, nil
	}
}

// NewKVDeleteHandler returns a HandlerFunc for "kv_delete" requests.
func NewKVDeleteHandler(cfg KVHandlerConfig) HandlerFunc {
	return func(req *Request) (interface{}, error) {
		var payload KVDeleteRequest
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return KVDeleteResponse{Error: "invalid payload"}, nil
		}

		key := strings.TrimSpace(payload.Key)
		if key == "" {
			return KVDeleteResponse{Error: "empty key"}, nil
		}

		store, err := openKVForRequest(cfg.WorkspaceName, cfg.OpenFunc)
		if err != nil {
			return KVDeleteResponse{Error: err.Error()}, nil
		}
		defer store.Close()

		if err := store.Delete([]byte(key)); err != nil {
			return KVDeleteResponse{Error: fmt.Sprintf("delete: %v", err)}, nil
		}

		return KVDeleteResponse{}, nil
	}
}

// NewKVListHandler returns a HandlerFunc for "kv_list" requests.
func NewKVListHandler(cfg KVHandlerConfig) HandlerFunc {
	return func(req *Request) (interface{}, error) {
		var payload KVListRequest
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return KVListResponse{Error: "invalid payload"}, nil
		}

		store, err := openKVForRequest(cfg.WorkspaceName, cfg.OpenFunc)
		if err != nil {
			return KVListResponse{Error: err.Error()}, nil
		}
		defer store.Close()

		var entries []KVListEntry
		err = store.Iterate([]byte(payload.Prefix), func(key, value []byte) error {
			entries = append(entries, KVListEntry{
				Key:   string(key),
				Value: string(value),
			})
			return nil
		})
		if err != nil {
			return KVListResponse{Error: fmt.Sprintf("list: %v", err)}, nil
		}

		return KVListResponse{Entries: entries}, nil
	}
}
