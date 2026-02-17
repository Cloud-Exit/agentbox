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

// exitbox-kv is a standalone binary for accessing a KV store from inside an
// ExitBox container. It communicates with the host via a Unix domain socket
// using JSON-lines protocol.
//
// Usage:
//
//	exitbox-kv get <KEY>              # prints value to stdout
//	exitbox-kv set <KEY> <VALUE>      # stores a value
//	exitbox-kv delete <KEY>           # removes a key
//	exitbox-kv list [PREFIX]          # prints matching keys, one per line
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type request struct {
	Type    string      `json:"type"`
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

type response struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type kvGetPayload struct {
	Key string `json:"key"`
}

type kvGetResponse struct {
	Value string `json:"value,omitempty"`
	Found bool   `json:"found"`
	Error string `json:"error,omitempty"`
}

type kvSetPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type kvSetResponse struct {
	Error string `json:"error,omitempty"`
}

type kvDeletePayload struct {
	Key string `json:"key"`
}

type kvDeleteResponse struct {
	Error string `json:"error,omitempty"`
}

type kvListPayload struct {
	Prefix string `json:"prefix"`
}

type kvListEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type kvListResponse struct {
	Entries []kvListEntry `json:"entries,omitempty"`
	Error   string        `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "get":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: exitbox-kv get <KEY>")
			os.Exit(1)
		}
		cmdGet(os.Args[2])
	case "set":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: exitbox-kv set <KEY> <VALUE>")
			os.Exit(1)
		}
		cmdSet(os.Args[2], os.Args[3])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: exitbox-kv delete <KEY>")
			os.Exit(1)
		}
		cmdDelete(os.Args[2])
	case "list":
		prefix := ""
		if len(os.Args) >= 3 {
			prefix = os.Args[2]
		}
		cmdList(prefix)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  exitbox-kv get <KEY>              # prints value to stdout")
	fmt.Fprintln(os.Stderr, "  exitbox-kv set <KEY> <VALUE>      # stores a value")
	fmt.Fprintln(os.Stderr, "  exitbox-kv delete <KEY>           # removes a key")
	fmt.Fprintln(os.Stderr, "  exitbox-kv list [PREFIX]          # prints matching keys, one per line")
}

func cmdGet(key string) {
	resp, err := sendRequest("kv_get", kvGetPayload{Key: key})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var payload kvGetResponse
	if err := json.Unmarshal(resp.Payload, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if payload.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", payload.Error)
		os.Exit(1)
	}
	if !payload.Found {
		os.Exit(1)
	}
	fmt.Print(payload.Value)
}

func cmdSet(key, value string) {
	resp, err := sendRequest("kv_set", kvSetPayload{Key: key, Value: value})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var payload kvSetResponse
	if err := json.Unmarshal(resp.Payload, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if payload.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", payload.Error)
		os.Exit(1)
	}
}

func cmdDelete(key string) {
	resp, err := sendRequest("kv_delete", kvDeletePayload{Key: key})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var payload kvDeleteResponse
	if err := json.Unmarshal(resp.Payload, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if payload.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", payload.Error)
		os.Exit(1)
	}
}

func cmdList(prefix string) {
	resp, err := sendRequest("kv_list", kvListPayload{Prefix: prefix})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var payload kvListResponse
	if err := json.Unmarshal(resp.Payload, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if payload.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", payload.Error)
		os.Exit(1)
	}
	for _, e := range payload.Entries {
		fmt.Println(e.Key)
	}
}

func sendRequest(reqType string, payload interface{}) (*response, error) {
	socketPath := os.Getenv("EXITBOX_IPC_SOCKET")
	if socketPath == "" {
		socketPath = "/run/exitbox/host.sock"
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("IPC socket not available. KV store requires the IPC server to be running")
	}
	defer conn.Close()

	id := randomID()
	req := request{
		Type:    reqType,
		ID:      id,
		Payload: payload,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return nil, scanErr
		}
		return nil, fmt.Errorf("no response from host")
	}

	var resp response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
