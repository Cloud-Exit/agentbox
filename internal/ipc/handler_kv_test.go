package ipc

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"

	"github.com/cloud-exit/exitbox/internal/kvstore"
)

// openTestKVDir returns a temp directory and an OpenFunc that opens a real
// on-disk BadgerDB in that directory. Each handler call opens/closes the DB,
// and data persists between opens because it's on disk.
func openTestKVDir(t *testing.T) (string, func(kvstore.Options) (*kvstore.Store, error)) {
	t.Helper()
	dir := t.TempDir()
	openFn := func(_ kvstore.Options) (*kvstore.Store, error) {
		return kvstore.Open(kvstore.Options{Dir: dir})
	}
	return dir, openFn
}

// seedKV opens the store at dir, writes key-value pairs, and closes it.
func seedKV(t *testing.T, dir string, pairs map[string]string) {
	t.Helper()
	store, err := kvstore.Open(kvstore.Options{Dir: dir})
	if err != nil {
		t.Fatalf("open seed store: %v", err)
	}
	for k, v := range pairs {
		if err := store.Set([]byte(k), []byte(v)); err != nil {
			store.Close()
			t.Fatalf("seed Set(%s): %v", k, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close seed store: %v", err)
	}
}

func TestKVGetFound(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	dir, openFn := openTestKVDir(t)
	seedKV(t, dir, map[string]string{"mykey": "myvalue"})

	srv.Handle("kv_get", NewKVGetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVGet(t, srv, "mykey")
	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
	if !resp.Found {
		t.Error("expected found=true")
	}
	if resp.Value != "myvalue" {
		t.Errorf("value = %q, want %q", resp.Value, "myvalue")
	}
}

func TestKVGetNotFound(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	_, openFn := openTestKVDir(t)

	srv.Handle("kv_get", NewKVGetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVGet(t, srv, "missing")
	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
	if resp.Found {
		t.Error("expected found=false for missing key")
	}
}

func TestKVGetEmptyKey(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	_, openFn := openTestKVDir(t)

	srv.Handle("kv_get", NewKVGetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVGet(t, srv, "")
	if resp.Error == "" {
		t.Error("expected error for empty key")
	}
}

func TestKVSetAndGet(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	_, openFn := openTestKVDir(t)

	srv.Handle("kv_get", NewKVGetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Handle("kv_set", NewKVSetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	setResp := sendKVSet(t, srv, "newkey", "newvalue")
	if setResp.Error != "" {
		t.Errorf("set error: %s", setResp.Error)
	}

	getResp := sendKVGet(t, srv, "newkey")
	if !getResp.Found {
		t.Error("expected key to be found after set")
	}
	if getResp.Value != "newvalue" {
		t.Errorf("value = %q, want %q", getResp.Value, "newvalue")
	}
}

func TestKVSetEmptyKey(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	_, openFn := openTestKVDir(t)

	srv.Handle("kv_set", NewKVSetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVSet(t, srv, "", "value")
	if resp.Error == "" {
		t.Error("expected error for empty key")
	}
}

func TestKVDelete(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	dir, openFn := openTestKVDir(t)
	seedKV(t, dir, map[string]string{"delme": "val"})

	srv.Handle("kv_get", NewKVGetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Handle("kv_delete", NewKVDeleteHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	delResp := sendKVDelete(t, srv, "delme")
	if delResp.Error != "" {
		t.Errorf("delete error: %s", delResp.Error)
	}

	getResp := sendKVGet(t, srv, "delme")
	if getResp.Found {
		t.Error("expected key to be gone after delete")
	}
}

func TestKVDeleteEmptyKey(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	_, openFn := openTestKVDir(t)

	srv.Handle("kv_delete", NewKVDeleteHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVDelete(t, srv, "")
	if resp.Error == "" {
		t.Error("expected error for empty key")
	}
}

func TestKVList(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	dir, openFn := openTestKVDir(t)
	seedKV(t, dir, map[string]string{
		"session:claude:proj1:.name":  "my-session",
		"session:claude:proj1:.token": "abc123",
		"session:codex:proj1:.name":   "other",
		"unrelated:key":               "val",
	})

	srv.Handle("kv_list", NewKVListHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVList(t, srv, "session:claude:")
	if resp.Error != "" {
		t.Errorf("list error: %s", resp.Error)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}
}

func TestKVListAll(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	dir, openFn := openTestKVDir(t)
	seedKV(t, dir, map[string]string{"a": "1", "b": "2"})

	srv.Handle("kv_list", NewKVListHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	resp := sendKVList(t, srv, "")
	if resp.Error != "" {
		t.Errorf("list error: %s", resp.Error)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}
}

func TestKVOpenPerRequest(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	dir := t.TempDir()
	var openCalls int
	openFn := func(_ kvstore.Options) (*kvstore.Store, error) {
		openCalls++
		return kvstore.Open(kvstore.Options{Dir: dir})
	}

	srv.Handle("kv_get", NewKVGetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Handle("kv_set", NewKVSetHandler(KVHandlerConfig{
		WorkspaceName: "test",
		OpenFunc:      openFn,
	}))
	srv.Start()

	sendKVSet(t, srv, "k", "v")
	sendKVGet(t, srv, "k")
	sendKVGet(t, srv, "k")

	if openCalls != 3 {
		t.Errorf("OpenFunc called %d times, expected 3 (once per request)", openCalls)
	}
}

// Test helpers

func sendKVGet(t *testing.T, srv *Server, key string) KVGetResponse {
	t.Helper()
	conn, err := net.Dial("unix", srv.socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	payload, err := json.Marshal(KVGetRequest{Key: key})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := Request{Type: "kv_get", ID: "test", Payload: payload}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err = conn.Write(append(data, '\n')); err != nil {
		t.Fatalf("Write: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	raw, err := json.Marshal(resp.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var result KVGetResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return result
}

func sendKVSet(t *testing.T, srv *Server, key, value string) KVSetResponse {
	t.Helper()
	conn, err := net.Dial("unix", srv.socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	payload, err := json.Marshal(KVSetRequest{Key: key, Value: value})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := Request{Type: "kv_set", ID: "test", Payload: payload}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err = conn.Write(append(data, '\n')); err != nil {
		t.Fatalf("Write: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	raw, err := json.Marshal(resp.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var result KVSetResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return result
}

func sendKVDelete(t *testing.T, srv *Server, key string) KVDeleteResponse {
	t.Helper()
	conn, err := net.Dial("unix", srv.socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	payload, err := json.Marshal(KVDeleteRequest{Key: key})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := Request{Type: "kv_delete", ID: "test", Payload: payload}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err = conn.Write(append(data, '\n')); err != nil {
		t.Fatalf("Write: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	raw, err := json.Marshal(resp.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var result KVDeleteResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return result
}

func sendKVList(t *testing.T, srv *Server, prefix string) KVListResponse {
	t.Helper()
	conn, err := net.Dial("unix", srv.socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	payload, err := json.Marshal(KVListRequest{Prefix: prefix})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := Request{Type: "kv_list", ID: "test", Payload: payload}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err = conn.Write(append(data, '\n')); err != nil {
		t.Fatalf("Write: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	raw, err := json.Marshal(resp.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var result KVListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return result
}
