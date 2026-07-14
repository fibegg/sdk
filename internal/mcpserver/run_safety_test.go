package mcpserver

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestFibeRunUsesCommandSafetyMetadata(t *testing.T) {
	root := &cobra.Command{Use: "fibe"}
	status := &cobra.Command{Use: "status", RunE: func(*cobra.Command, []string) error { return nil }, Annotations: map[string]string{"fibe.gg/safety": "read-only"}}
	restart := &cobra.Command{Use: "restart", RunE: func(*cobra.Command, []string) error { return nil }, Annotations: map[string]string{"fibe.gg/safety": "destructive"}}
	unknown := &cobra.Command{Use: "unknown", RunE: func(*cobra.Command, []string) error { return nil }}
	root.AddCommand(status, restart, unknown)
	srv := New(Config{CobraRoot: root})

	for _, tc := range []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{name: "status"},
		{name: "restart", want: true},
		{name: "unknown", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := srv.fibeRunRequiresConfirm(map[string]any{"args": []any{tc.name}})
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("requires=%v err=%v", got, err)
			}
		})
	}
}
