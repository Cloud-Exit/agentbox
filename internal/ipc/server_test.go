package ipc

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
)

func TestServerRoundTrip(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	srv.Handle("echo", func(req *Request) (interface{}, error) {
		var payload map[string]string
		json.Unmarshal(req.Payload, &payload)
		return payload, nil
	})
	srv.Start()

	// Connect and send a request.
	conn, err := net.Dial("unix", srv.socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	req := Request{
		Type: "echo",
		ID:   "test-1",
	}
	payload, _ := json.Marshal(map[string]string{"msg": "hello"})
	req.Payload = payload
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Type != "echo" {
		t.Errorf("type = %q, want echo", resp.Type)
	}
	if resp.ID != "test-1" {
		t.Errorf("id = %q, want test-1", resp.ID)
	}

	// Decode payload.
	raw, _ := json.Marshal(resp.Payload)
	var got map[string]string
	json.Unmarshal(raw, &got)
	if got["msg"] != "hello" {
		t.Errorf("payload msg = %q, want hello", got["msg"])
	}
}

func TestServerUnknownType(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()
	srv.Start()

	conn, err := net.Dial("unix", srv.socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	req := Request{Type: "nonexistent", ID: "test-2"}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	json.Unmarshal(scanner.Bytes(), &resp)

	raw, _ := json.Marshal(resp.Payload)
	var payload AllowDomainResponse
	json.Unmarshal(raw, &payload)

	if payload.Error == "" {
		t.Error("expected error for unknown type")
	}
}
