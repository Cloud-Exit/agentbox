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

package run

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/cloud-exit/exitbox/internal/ui"
)

// IDERelay bridges a Unix domain socket to the host IDE's TCP WebSocket port.
type IDERelay struct {
	Port       string
	SocketPath string
	LockDir    string // temp dir with rewritten lock file
	listener   net.Listener
	cancel     context.CancelFunc
}

// ideLockFile represents the JSON lock file written by the IDE extension.
type ideLockFile struct {
	WorkspaceFolders []string `json:"workspaceFolders"`
	PID              int      `json:"pid"`
	IDEName          string   `json:"ideName"`
	Transport        string   `json:"transport"`
	AuthToken        string   `json:"authToken"`
}

// DetectIDE checks whether IDE integration should be enabled.
// It returns the port and true only when the agent is "claude" and
// the CLAUDE_CODE_SSE_PORT env var contains a valid port number.
func DetectIDE(agent string) (string, bool) {
	if agent != "claude" {
		return "", false
	}
	port := os.Getenv("CLAUDE_CODE_SSE_PORT")
	if port == "" {
		return "", false
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", false
	}
	return port, true
}

// StartIDERelay creates ide.sock in the IPC socket directory and starts a
// goroutine that accepts Unix connections and relays them to the IDE's TCP
// port on the host (127.0.0.1:<port>).
func StartIDERelay(ipcSocketDir, port string) *IDERelay {
	socketPath := filepath.Join(ipcSocketDir, "ide.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		ui.Warnf("Failed to create IDE relay socket: %v", err)
		return nil
	}
	// Allow non-root container user to connect (matches host.sock pattern).
	if err := os.Chmod(socketPath, 0666); err != nil {
		ui.Warnf("Failed to chmod IDE relay socket: %v", err)
		listener.Close()
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	relay := &IDERelay{
		Port:       port,
		SocketPath: socketPath,
		listener:   listener,
		cancel:     cancel,
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go relayConn(ctx, conn, port)
		}
	}()

	return relay
}

// relayConn handles a single Unix connection by dialing the IDE's TCP port
// and performing bidirectional copy.
func relayConn(ctx context.Context, unixConn net.Conn, port string) {
	defer unixConn.Close()

	tcpConn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		return
	}
	defer tcpConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// When the parent context is cancelled, close both connections to
	// unblock any in-progress io.Copy calls.
	go func() {
		<-ctx.Done()
		unixConn.Close()
		tcpConn.Close()
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(tcpConn, unixConn)
		// Signal the other direction that reads are done.
		if tc, ok := tcpConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(unixConn, tcpConn)
		// Signal the other direction that reads are done.
		if uc, ok := unixConn.(*net.UnixConn); ok {
			_ = uc.CloseWrite()
		}
	}()

	wg.Wait()
}

// PrepareIDELockFile reads the IDE lock file from the host, rewrites the PID
// to 1 (the container init process, always valid due to --init), and writes
// the modified file to a temp directory. Returns the temp directory path,
// or "" on failure.
func PrepareIDELockFile(port string) string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	srcPath := filepath.Join(home, ".claude", "ide", port+".lock")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return ""
	}

	var lock ideLockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return ""
	}

	lock.PID = 1

	rewritten, err := json.Marshal(lock)
	if err != nil {
		return ""
	}

	tmpDir, err := os.MkdirTemp("", "exitbox-ide-lock-*")
	if err != nil {
		return ""
	}

	ideDir := filepath.Join(tmpDir, "ide")
	if err := os.MkdirAll(ideDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return ""
	}

	dstPath := filepath.Join(ideDir, port+".lock")
	if err := os.WriteFile(dstPath, rewritten, 0644); err != nil {
		os.RemoveAll(tmpDir)
		return ""
	}

	return tmpDir
}

// ContainerArgs returns the container engine flags needed to enable IDE
// integration inside the container.
func (r *IDERelay) ContainerArgs() []string {
	if r == nil {
		return nil
	}
	args := []string{
		"-e", "CLAUDE_CODE_SSE_PORT=" + r.Port,
		"-e", "ENABLE_IDE_INTEGRATION=true",
		"-e", "EXITBOX_IDE_PORT=" + r.Port,
	}
	if r.LockDir != "" {
		args = append(args, "-v", filepath.Join(r.LockDir, "ide")+":/home/user/.claude/ide:ro")
	}
	return args
}

// StopIDERelay safely shuts down the relay. It is nil-safe.
func StopIDERelay(r *IDERelay) {
	if r == nil {
		return
	}
	if r.cancel != nil {
		r.cancel()
	}
	if r.listener != nil {
		r.listener.Close()
	}
	if r.LockDir != "" {
		os.RemoveAll(r.LockDir)
	}
}
