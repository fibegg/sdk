package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// fakeFibeExecutable records the argv and session key a fibe_run subprocess receives.
func fakeFibeExecutable(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake CLI is a POSIX shell script")
	}
	path := filepath.Join(t.TempDir(), "fibe")
	script := "#!/bin/sh\nfor a in \"$@\"; do printf 'arg=%s\\n' \"$a\"; done\nprintf 'key=%s\\n' \"$" + RunAPIKeyEnv + "\"\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake CLI: %v", err)
	}
	return path
}

func TestFibeRunSubprocessUsesSessionCredentials(t *testing.T) {
	const domain = "http://127.0.0.1:65535"
	cases := []struct {
		name      string
		cfg       Config
		session   func(*Server, context.Context)
		args      []any
		wantFlags []string
		wantKey   string
	}{
		{
			name:      "server --api-key travels through env",
			cfg:       Config{APIKey: "pk_server", Domain: domain, Profile: "default", AuthSource: "--api-key flag"},
			wantFlags: []string{"--domain", domain},
			wantKey:   "pk_server",
		},
		{
			name:      "server profile passes the profile name only",
			cfg:       Config{APIKey: "pk_profile", Domain: domain, Profile: "staging", AuthSource: "profile staging"},
			wantFlags: []string{"--domain", domain, "--profile", "staging"},
		},
		{
			name:      "fibe_auth_set key overrides the server profile",
			cfg:       Config{APIKey: "pk_profile", Domain: domain, Profile: "staging", AuthSource: "profile staging"},
			session:   func(s *Server, ctx context.Context) { s.setSessionAuth(ctx, "pk_session", "") },
			wantFlags: []string{"--domain", domain},
			wantKey:   "pk_session",
		},
		{
			name:      "fibe_auth_use profile overrides the server key",
			cfg:       Config{APIKey: "pk_server", Domain: domain, Profile: "default", AuthSource: "--api-key flag"},
			session:   func(s *Server, ctx context.Context) { s.setSessionProfile(ctx, "prod", "pk_prod", domain) },
			wantFlags: []string{"--domain", domain, "--profile", "prod"},
		},
		{
			name:      "caller auth flags win",
			cfg:       Config{APIKey: "pk_server", Domain: domain, Profile: "default", AuthSource: "--api-key flag"},
			args:      []any{"--profile", "other"},
			wantFlags: []string{"--domain", domain},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(RunAPIKeyEnv, "pk_stale_parent")
			root := &cobra.Command{Use: "fibe"}
			root.PersistentFlags().String("profile", "", "")
			root.AddCommand(&cobra.Command{Use: "status", RunE: func(*cobra.Command, []string) error { return nil }, Annotations: map[string]string{"fibe.gg/safety": "read-only"}})
			tc.cfg.ToolSet = "full"
			tc.cfg.CobraRoot = root
			tc.cfg.CobraExecutable = fakeFibeExecutable(t)
			srv := New(tc.cfg)
			if err := srv.RegisterAll(); err != nil {
				t.Fatalf("RegisterAll: %v", err)
			}
			ctx := context.Background()
			if tc.session != nil {
				tc.session(srv, ctx)
			}

			out, err := srv.dispatcher.dispatch(ctx, "fibe_run", map[string]any{"args": append([]any{"status"}, tc.args...)})
			if err != nil {
				t.Fatalf("dispatch: %v", err)
			}
			result := out.(map[string]any)
			var gotArgs []string
			var gotKey string
			for _, line := range strings.Split(strings.TrimSpace(result["stdout"].(string)), "\n") {
				if arg, ok := strings.CutPrefix(line, "arg="); ok {
					gotArgs = append(gotArgs, arg)
				} else if key, ok := strings.CutPrefix(line, "key="); ok {
					gotKey = key
				}
			}
			userArgs := []string{"--output", "json", "status"}
			for _, arg := range tc.args {
				userArgs = append(userArgs, arg.(string))
			}
			if want := append(append([]string(nil), tc.wantFlags...), userArgs...); !slices.Equal(gotArgs, want) {
				t.Fatalf("argv = %q, want %q", gotArgs, want)
			}
			if gotKey != tc.wantKey {
				t.Fatalf("%s = %q, want %q", RunAPIKeyEnv, gotKey, tc.wantKey)
			}
			if echoed := result["args"].([]string); !slices.Equal(echoed, userArgs) {
				t.Fatalf("result args = %q, want caller args only %q", echoed, userArgs)
			}
			for _, arg := range gotArgs {
				if strings.HasPrefix(arg, "pk_") {
					t.Fatalf("API key leaked into argv: %q", gotArgs)
				}
			}
		})
	}
}
