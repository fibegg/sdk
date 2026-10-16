package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestResourceDeleteReturnsBackendOutcome(t *testing.T) {
	for _, resource := range []string{"playground", "template_version"} {
		for _, status := range []int{http.StatusNoContent, http.StatusUnauthorized, http.StatusUnprocessableEntity} {
			t.Run(resource+http.StatusText(status), func(t *testing.T) {
				var calls atomic.Int32
				api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != http.MethodDelete {
						t.Errorf("unexpected %s", r.Method)
					}
					w.WriteHeader(status)
					if status != http.StatusNoContent {
						_, _ = w.Write([]byte(`{"error":"synthetic delete rejection"}`))
					}
				}))
				defer api.Close()
				srv := New(Config{APIKey: "synthetic", Domain: api.URL, ToolSet: "core", Yolo: true})
				if err := srv.RegisterAll(); err != nil {
					t.Fatal(err)
				}
				args := map[string]any{"resource": resource, "id_or_name": "demo"}
				if resource == "template_version" {
					args = map[string]any{"resource": resource, "template_id_or_name": "demo", "id": float64(9)}
				}
				result, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_delete", args)
				if status == http.StatusNoContent && err != nil {
					t.Fatal(err)
				}
				if status != http.StatusNoContent && (err == nil || result != nil) {
					t.Fatalf("false deletion: %#v %v", result, err)
				}
				if calls.Load() != 1 {
					t.Fatalf("delete submitted %d times", calls.Load())
				}
			})
		}
	}
}

type deleteFailureTransport struct {
	calls int
	err   error
}

func (d *deleteFailureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	d.calls++
	return nil, d.err
}

func TestResourceDeletePropagatesTransportFailure(t *testing.T) {
	for _, resource := range []string{"playground", "template_version"} {
		srv := New(Config{ToolSet: "core"})
		if err := srv.RegisterAll(); err != nil {
			t.Fatal(err)
		}
		failure := errors.New("synthetic delete transport failure")
		transport := &deleteFailureTransport{err: failure}
		client := fibe.NewClient(fibe.WithDisableAutoConfig(), fibe.WithBaseURL("http://fixture.invalid"), fibe.WithAPIKey("synthetic"), fibe.WithMaxRetries(0), fibe.WithHTTPClient(&http.Client{Transport: transport}))
		args := map[string]any{"resource": resource, "id_or_name": "fixture"}
		if resource == "template_version" {
			args = map[string]any{"resource": resource, "template_id_or_name": "fixture", "id": float64(9)}
		}
		result, err := srv.dispatcher.tools["fibe_resource_delete"].handler(context.Background(), client, args)
		if !errors.Is(err, failure) || result != nil || transport.calls != 1 {
			t.Fatalf("%s result=%#v error=%v calls=%d", resource, result, err, transport.calls)
		}
	}
}
