package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestMCPHelpWhileRunSubprocessIsActive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	t.Setenv("FIBE_TEST_MCP_CHILD_MARKER", marker)
	exe := filepath.Join(dir, "fibe")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n: > \"$FIBE_TEST_MCP_CHILD_MARKER\"\nexec sleep 60\n"), 0o700))

	cfg := DefaultConfig()
	cfg.CobraExecutable = exe
	cfg.CobraRoot = &cobra.Command{Use: "fibe"}
	cfg.CobraRoot.AddCommand(&cobra.Command{
		Use: "wait", Short: "Wait for work",
		Annotations: map[string]string{"fibe.gg/safety": "read-only"},
		Run:         func(*cobra.Command, []string) {},
	})
	cfg.CobraRoot.AddCommand(&cobra.Command{
		Use: "destroy", Short: "Destroy work",
		Annotations: map[string]string{"fibe.gg/safety": "destructive"},
		Run:         func(*cobra.Command, []string) {},
	})
	s := New(cfg)
	require.NoError(t, s.RegisterAll())
	ctx, cancel := context.WithCancel(context.Background())
	childDone := make(chan struct{})
	go func() {
		defer close(childDone)
		_, _ = s.dispatcher.dispatch(ctx, "fibe_run", map[string]any{"args": []any{"wait"}})
	}()
	defer func() {
		cancel()
		<-childDone
	}()
	require.Eventually(t, func() bool {
		_, err := os.Stat(marker)
		return err == nil
	}, 8*time.Second, 5*time.Millisecond)

	type reply struct {
		help         any
		err          error
		rpc          any
		confirmation error
	}
	result := make(chan reply, 1)
	go func() {
		help, err := s.dispatcher.dispatch(ctx, "fibe_help", map[string]any{"path": "wait"})
		rpc := s.mcp.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"fibe://help/wait"}}`))
		_, confirmation := s.dispatcher.dispatch(ctx, "fibe_run", map[string]any{"args": []any{"destroy"}})
		result <- reply{help: help, err: err, rpc: rpc, confirmation: confirmation}
	}()
	select {
	case out := <-result:
		require.NoError(t, out.err)
		require.Equal(t, "Wait for work", out.help.(map[string]any)["short"])
		serialized, err := json.Marshal(out.rpc)
		require.NoError(t, err)
		require.Contains(t, string(serialized), "Wait for work")
		var confirm *confirmRequiredError
		require.ErrorAs(t, out.confirmation, &confirm)
		select {
		case <-childDone:
			t.Fatal("CLI subprocess finished before the help and confirmation replies")
		default:
		}
	case <-time.After(8 * time.Second):
		t.Fatal("help waited for a running CLI subprocess")
	}
}
