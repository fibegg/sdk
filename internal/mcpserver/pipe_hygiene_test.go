package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

// TestStdoutIsolation verifies that even if a tool handler writes garbage
// to os.Stdout (the exact bug that corrupted Antigravity's JSON-RPC pipe),
// the MCP transport stays clean.
//
// We build the fibe binary, spawn it in mcp serve mode, send a tool call
// that triggers a stdout write inside the process, and then parse every
// line of the server's real stdout. Each line must be valid JSON-RPC.
func TestStdoutIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode: skipping subprocess-based pipe hygiene test")
	}

	// Locate the sdk module root so we can `go build` the fibe binary.
	moduleRoot := findModuleRoot(t)
	bin := filepath.Join(t.TempDir(), "fibe")
	build := exec.Command("go", "build", "-o", bin, "./cmd/fibe")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build fibe: %v\n%s", err, out)
	}

	// fibe_run with a harmless command exercises the cobra path most likely
	// to emit stray fmt.Println: `doctor` prints ASCII art. If stdout
	// isolation works, none of those bytes reach the JSON-RPC pipe.
	cmd := exec.Command(bin, "mcp", "serve")
	cmd.Env = append(os.Environ(), "FIBE_API_KEY=pk_test_invalid_on_purpose")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fibe mcp serve: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	sendJSON := func(payload string) {
		if _, err := fmt.Fprintln(stdin, payload); err != nil {
			t.Fatalf("write to stdin: %v", err)
		}
	}

	// 1. Handshake.
	sendJSON(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}`)
	sendJSON(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	// 2. Call fibe_run doctor — deliberately triggers stray stdout writes
	//    from the cobra command, which bypass cobra's SetOut and hit the
	//    global os.Stdout. The pipe hijack must redirect them.
	sendJSON(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"fibe_run","arguments":{"args":["doctor"]}}}`)
	// 3. Call fibe_auth_set with a clearly bogus key and validate:false so
	//    we skip the ping path that would produce an expected HTTP error.
	sendJSON(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"fibe_auth_set","arguments":{"api_key":"pk_test_totally_invalid","validate":false}}}`)
	// 4. tools/list should still succeed after the prior calls.
	sendJSON(`{"jsonrpc":"2.0","id":4,"method":"tools/list"}`)

	// Drain the first 4 response lines with a hard wall clock. We use a
	// single background goroutine reading sequentially (bufio.Reader is not
	// safe for concurrent reads) and surface lines via a channel.
	type lineMsg struct {
		line []byte
		err  error
	}
	lines := make(chan lineMsg, 16)
	go func() {
		br := bufio.NewReader(stdout)
		for {
			l, err := br.ReadBytes('\n')
			if len(l) > 0 {
				lines <- lineMsg{line: bytes.Clone(l), err: nil}
			}
			if err != nil {
				lines <- lineMsg{err: err}
				return
			}
		}
	}()

	deadline := time.After(6 * time.Second)
	gotIDs := map[float64]bool{}
	for !(gotIDs[1] && gotIDs[2] && gotIDs[3] && gotIDs[4]) {
		select {
		case msg := <-lines:
			if msg.err != nil && len(msg.line) == 0 {
				t.Fatalf("stdout closed before all responses arrived; so far got %v (err=%v)", gotIDs, msg.err)
			}
			line := bytes.TrimSpace(msg.line)
			if len(line) == 0 {
				continue
			}
			var parsed map[string]any
			if jerr := json.Unmarshal(line, &parsed); jerr != nil {
				t.Fatalf("non-JSON on MCP pipe (corruption!): %q err=%v", string(line), jerr)
			}
			if v, ok := parsed["jsonrpc"]; !ok || v != "2.0" {
				t.Fatalf("non-2.0 JSON-RPC line: %q", string(line))
			}
			if id, ok := parsed["id"].(float64); ok {
				gotIDs[id] = true
			}
		case <-deadline:
			t.Fatalf("timeout waiting for JSON-RPC responses; so far got %v", gotIDs)
		}
	}

	for _, id := range []float64{1, 2, 3, 4} {
		if !gotIDs[id] {
			t.Errorf("never received response for id=%v", id)
		}
	}
}

// TestRunCobraCapturesStdout verifies that fibe_run's os.Stdout hijack
// captures direct fmt.Println output into the result, instead of losing it
// to stderr or corrupting the parent pipe.
//
// We use a real *cobra.Command with a RunE that calls fmt.Println directly —
// the exact pattern every cmd_*.go handler uses in this repo.
func TestRunCobraCapturesStdout(t *testing.T) {
	srv := New(Config{APIKey: "pk_test"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	root := &cobra.Command{Use: "fibe"}
	echo := &cobra.Command{
		Use: "echo",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Directly write to os.Stdout — bypasses cobra's SetOut, which is
			// the exact antipattern the hijack has to tolerate.
			fmt.Println("line via fmt.Println")
			return nil
		},
	}
	// fibe_run prepends --output json globally; add a matching persistent
	// flag so cobra doesn't reject the unknown flag.
	root.PersistentFlags().String("output", "", "")
	root.AddCommand(echo)
	srv.cfg.CobraRoot = root

	result, err := srv.runCobra(context.Background(), map[string]any{
		"args": []any{"echo"},
	})
	if err != nil {
		t.Fatalf("runCobra: %v", err)
	}
	m := result.(map[string]any)
	if !strings.Contains(m["stdout"].(string), "line via fmt.Println") {
		t.Errorf("expected captured stdout; got %q", m["stdout"])
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above %s", wd)
		}
		dir = parent
	}
}

// Keep fibe import live (used in other tests in this package; compile-time guard).
var _ = fibe.NewIdempotencyKey
