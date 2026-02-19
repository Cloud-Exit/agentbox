package run

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectIDE_ClaudeWithPort(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SSE_PORT", "12345")
	port, ok := DetectIDE("claude")
	if !ok {
		t.Fatal("expected ok=true for claude with valid port")
	}
	if port != "12345" {
		t.Errorf("port = %q, want %q", port, "12345")
	}
}

func TestDetectIDE_NotClaude(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SSE_PORT", "12345")
	_, ok := DetectIDE("codex")
	if ok {
		t.Error("expected ok=false for non-claude agent")
	}
}

func TestDetectIDE_NoEnvVar(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SSE_PORT", "")
	_, ok := DetectIDE("claude")
	if ok {
		t.Error("expected ok=false when CLAUDE_CODE_SSE_PORT is empty")
	}
}

func TestDetectIDE_InvalidPort(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SSE_PORT", "notanumber")
	_, ok := DetectIDE("claude")
	if ok {
		t.Error("expected ok=false for non-numeric port")
	}
}

func TestDetectIDE_PortOutOfRange(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SSE_PORT", "99999")
	_, ok := DetectIDE("claude")
	if ok {
		t.Error("expected ok=false for port > 65535")
	}
}

func TestDetectIDE_PortZero(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SSE_PORT", "0")
	_, ok := DetectIDE("claude")
	if ok {
		t.Error("expected ok=false for port 0")
	}
}

func TestPrepareIDELockFile_RewritesPID(t *testing.T) {
	// Create a fake ~/.claude/ide/<port>.lock
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	ideDir := filepath.Join(homeDir, ".claude", "ide")
	if err := os.MkdirAll(ideDir, 0755); err != nil {
		t.Fatal(err)
	}

	lock := ideLockFile{
		WorkspaceFolders: []string{"/home/user/project"},
		PID:              42,
		IDEName:          "VS Code",
		Transport:        "ws",
		AuthToken:        "test-auth-token-uuid",
	}
	data, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ideDir, "12345.lock"), data, 0644); err != nil {
		t.Fatal(err)
	}

	tmpDir := PrepareIDELockFile("12345")
	if tmpDir == "" {
		t.Fatal("expected non-empty tmpDir")
	}
	defer os.RemoveAll(tmpDir)

	// Read the rewritten lock file
	rewrittenPath := filepath.Join(tmpDir, "ide", "12345.lock")
	rewrittenData, err := os.ReadFile(rewrittenPath)
	if err != nil {
		t.Fatal(err)
	}

	var rewritten ideLockFile
	if err := json.Unmarshal(rewrittenData, &rewritten); err != nil {
		t.Fatal(err)
	}

	if rewritten.PID != 1 {
		t.Errorf("PID = %d, want 1", rewritten.PID)
	}
	if rewritten.AuthToken != "test-auth-token-uuid" {
		t.Errorf("AuthToken = %q, want %q", rewritten.AuthToken, "test-auth-token-uuid")
	}
	if rewritten.IDEName != "VS Code" {
		t.Errorf("IDEName = %q, want %q", rewritten.IDEName, "VS Code")
	}
	if len(rewritten.WorkspaceFolders) != 1 || rewritten.WorkspaceFolders[0] != "/home/user/project" {
		t.Errorf("WorkspaceFolders = %v, want [/home/user/project]", rewritten.WorkspaceFolders)
	}
}

func TestPrepareIDELockFile_MissingFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	result := PrepareIDELockFile("99999")
	if result != "" {
		t.Errorf("expected empty string for missing lock file, got %q", result)
		os.RemoveAll(result)
	}
}

func TestContainerArgs_ContainsExpectedVars(t *testing.T) {
	lockDir := t.TempDir()
	ideDir := filepath.Join(lockDir, "ide")
	if err := os.MkdirAll(ideDir, 0755); err != nil {
		t.Fatal(err)
	}

	relay := &IDERelay{
		Port:    "12345",
		LockDir: lockDir,
	}

	args := relay.ContainerArgs()
	argStr := strings.Join(args, " ")

	expected := []string{
		"CLAUDE_CODE_SSE_PORT=12345",
		"ENABLE_IDE_INTEGRATION=true",
		"EXITBOX_IDE_PORT=12345",
		filepath.Join(lockDir, "ide") + ":/home/user/.claude/ide:ro",
	}
	for _, want := range expected {
		if !strings.Contains(argStr, want) {
			t.Errorf("ContainerArgs missing %q in %q", want, argStr)
		}
	}
}

func TestContainerArgs_NilRelay(t *testing.T) {
	var relay *IDERelay
	args := relay.ContainerArgs()
	if args != nil {
		t.Errorf("expected nil args for nil relay, got %v", args)
	}
}

func TestContainerArgs_NoLockDir(t *testing.T) {
	relay := &IDERelay{
		Port: "12345",
	}
	args := relay.ContainerArgs()
	argStr := strings.Join(args, " ")
	if strings.Contains(argStr, "/home/user/.claude/ide") {
		t.Error("ContainerArgs should not contain lock mount when LockDir is empty")
	}
	if !strings.Contains(argStr, "EXITBOX_IDE_PORT=12345") {
		t.Error("ContainerArgs should still contain EXITBOX_IDE_PORT")
	}
}

func TestStopIDERelay_NilSafe(t *testing.T) {
	// Should not panic
	StopIDERelay(nil)
}

func TestStopIDERelay_CleansUp(t *testing.T) {
	tmpDir := t.TempDir()
	lockDir, err := os.MkdirTemp(tmpDir, "lock-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create a real listener to test close
	socketPath := filepath.Join(tmpDir, "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}

	relay := &IDERelay{
		Port:       "12345",
		SocketPath: socketPath,
		LockDir:    lockDir,
		listener:   listener,
		cancel:     func() {},
	}

	StopIDERelay(relay)

	if _, err := os.Stat(lockDir); !os.IsNotExist(err) {
		t.Error("expected LockDir to be removed")
	}
}

func TestStartIDERelay_CreatesSocket(t *testing.T) {
	// Start a TCP server to act as the IDE
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcpListener.Close()

	_, portStr, _ := net.SplitHostPort(tcpListener.Addr().String())

	socketDir := t.TempDir()
	relay := StartIDERelay(socketDir, portStr)
	if relay == nil {
		t.Fatal("expected non-nil relay")
	}
	defer StopIDERelay(relay)

	// Verify the socket file was created
	socketPath := filepath.Join(socketDir, "ide.sock")
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("socket file not created: %v", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		t.Error("expected file to be a socket")
	}
}
