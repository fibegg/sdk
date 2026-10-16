package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func TestWaitDeadlineAndCancellationCoverBlockedRequests(t *testing.T) {
	for _, resource := range []string{"playground", "task", "repository"} {
		for _, finalRead := range []bool{false, true} {
			if resource == "repository" && finalRead {
				continue
			}
			for _, cancelled := range []bool{false, true} {
				t.Run(resource+map[bool]string{false: "-status", true: "-final"}[finalRead]+map[bool]string{false: "-deadline", true: "-cancel"}[cancelled], func(t *testing.T) {
					setupAuthTest(t)
					started := make(chan struct{}, 1)
					release := make(chan struct{})
					api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if finalRead && strings.HasSuffix(r.URL.Path, "/status") {
							if resource == "task" {
								_, _ = w.Write([]byte(`{"id":8,"status":"completed"}`))
							} else {
								_, _ = w.Write([]byte(`{"id":8,"status":"running","services":[{"name":"web","status":"running","health":"healthy"}]}`))
							}
							return
						}
						select {
						case started <- struct{}{}:
						default:
						}
						select {
						case <-r.Context().Done():
						case <-release:
						}
					}))
					defer api.Close()
					defer close(release)
					t.Setenv("FIBE_DOMAIN", api.URL)
					t.Setenv("FIBE_API_KEY", "synthetic")
					parent, cancel := context.WithCancel(context.Background())
					defer cancel()
					cmd := waitCmd()
					cmd.SilenceUsage = true
					cmd.SetOut(&bytes.Buffer{})
					cmd.SetContext(parent)
					cmd.SetErr(&bytes.Buffer{})
					args := []string{resource, "fixture", "--timeout=80ms", "--interval=1h"}
					if resource == "task" {
						args = append(args, "--status=completed", "--full")
					}
					cmd.SetArgs(args)
					done := make(chan error, 1)
					go func() { done <- cmd.Execute() }()
					select {
					case <-started:
					case <-time.After(time.Second):
						t.Fatal("request never started")
					}
					if cancelled {
						cancel()
					}
					select {
					case err := <-done:
						want := context.DeadlineExceeded
						if cancelled {
							want = context.Canceled
						}
						if !errors.Is(err, want) {
							t.Fatalf("error=%v want %v", err, want)
						}
					case <-time.After(time.Second):
						cancel()
						t.Fatal("blocked request exceeded wait bound")
					}
				})
			}
		}
	}
}

func TestCreatedPlaygroundWaitBoundsBlockedRequest(t *testing.T) {
	release := make(chan struct{})
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer api.Close()
	defer close(release)
	c := fibe.NewClient(fibe.WithDisableAutoConfig(), fibe.WithBaseURL(api.URL), fibe.WithAPIKey("synthetic"), fibe.WithMaxRetries(0))
	start := time.Now()
	_, err := waitForPlayground(context.Background(), c, 8, "running", 60*time.Millisecond, time.Hour, nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("error=%v elapsed=%v", err, time.Since(start))
	}
}

type waitStatusSignal struct {
	once   sync.Once
	status chan struct{}
}

func (w *waitStatusSignal) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "status: pending") {
		w.once.Do(func() { close(w.status) })
	}
	return len(p), nil
}

func TestWaitCancellationInterruptsPollingSleep(t *testing.T) {
	setupAuthTest(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"id":8,"status":"pending"}`)) }))
	defer api.Close()
	t.Setenv("FIBE_DOMAIN", api.URL)
	t.Setenv("FIBE_API_KEY", "synthetic")
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &waitStatusSignal{status: make(chan struct{})}
	cmd := waitCmd()
	cmd.SetContext(parent)
	cmd.SetErr(writer)
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"playground", "fixture", "--interval=1h", "--timeout=2h"})
	done := make(chan error, 1)
	go func() { done <- cmd.Execute() }()
	select {
	case <-writer.status:
	case <-time.After(time.Second):
		t.Fatal("status not received")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("polling sleep ignored cancellation")
	}
}
