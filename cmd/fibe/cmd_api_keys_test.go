package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestKeyGranularScopeArguments(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		want  map[string][]int64
	}{
		{"omitted", nil, nil},
		{"explicit empty", []string{"--granular-scope", "agents:read="}, map[string][]int64{"agents:read": {}}},
		{"multiple IDs", []string{"--granular-scope", "agents:read=12,15"}, map[string][]int64{"agents:read": {12, 15}}},
		{"independent scopes", []string{"--granular-scope", "agents:read=12", "--granular-scope", "monitor:read="}, map[string][]int64{"agents:read": {12}, "monitor:read": {}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupAuthTest(t)
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				var body struct {
					APIKey struct {
						Granular map[string][]int64 `json:"granular_scopes"`
					} `json:"api_key"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body.APIKey.Granular, tc.want) {
					t.Errorf("restrictions=%#v want=%#v", body.APIKey.Granular, tc.want)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":1,"label":"test","scopes":["agents:read"],"granular_scopes":{},"token":"synthetic"}`))
			}))
			defer server.Close()
			t.Setenv("FIBE_DOMAIN", server.URL)
			t.Setenv("FIBE_API_KEY", "synthetic")
			cmd := RootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(append([]string{"--output", "json", "keys", "create", "--label", "test", "--scope", "agents:read", "--scope", "monitor:read"}, tc.flags...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("request missing")
			}
		})
	}
}
