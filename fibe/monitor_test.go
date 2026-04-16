package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMonitor_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/monitor" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if got := query.Get("agent_id"); got != "42,43" {
			t.Fatalf("expected agent_id=42,43, got %q", got)
		}
		if got := query.Get("types"); got != "message,artefact" {
			t.Fatalf("expected types filter, got %q", got)
		}
		if got := query.Get("page"); got != "2" {
			t.Fatalf("expected page=2, got %q", got)
		}
		if got := query.Get("per_page"); got != "10" {
			t.Fatalf("expected per_page=10, got %q", got)
		}
		if query.Get("cursor") != "" || query.Get("after_cursor") != "" {
			t.Fatalf("did not expect cursor params, got cursor=%q after_cursor=%q", query.Get("cursor"), query.Get("after_cursor"))
		}

		json.NewEncoder(w).Encode(monitorListEnvelope{
			Data: []MonitorEvent{
				{
					Type:       MonitorTypeMessage,
					AgentID:    42,
					OccurredAt: "2026-04-16T12:00:00.000000Z",
					ItemID:     "m1",
					Payload:    map[string]any{"body": "hello"},
				},
			},
			Meta: MonitorMeta{Page: 2, PerPage: 10, Total: 11},
		})
	})

	res, err := c.Monitor.List(context.Background(), &MonitorListParams{
		AgentIDs: "42,43",
		Types:    "message,artefact",
		Page:     2,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("Monitor.List: %v", err)
	}
	if len(res.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(res.Data))
	}
	if res.Meta.Page != 2 || res.Meta.PerPage != 10 || res.Meta.Total != 11 {
		t.Fatalf("unexpected meta: %+v", res.Meta)
	}
}

func TestMonitor_FollowStartsFromNowDedupesAndOrdersChronologically(t *testing.T) {
	requests := 0
	firstSince := ""
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/monitor" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("page") != "1" || query.Get("per_page") != "100" {
			t.Fatalf("expected follow to force page=1 per_page=100, got page=%q per_page=%q", query.Get("page"), query.Get("per_page"))
		}

		switch requests {
		case 1:
			firstSince = query.Get("since")
			if _, err := time.Parse(time.RFC3339Nano, firstSince); err != nil {
				t.Fatalf("expected RFC3339 since on first poll, got %q (%v)", firstSince, err)
			}
			json.NewEncoder(w).Encode(monitorListEnvelope{Data: []MonitorEvent{}, Meta: MonitorMeta{Page: 1, PerPage: 100, Total: 0}})
		case 2:
			if query.Get("since") != firstSince {
				t.Fatalf("expected second poll to reuse initial since %q, got %q", firstSince, query.Get("since"))
			}
			json.NewEncoder(w).Encode(monitorListEnvelope{
				Data: []MonitorEvent{
					{Type: MonitorTypeMessage, AgentID: 7, OccurredAt: "2026-04-16T12:00:02.000000Z", ItemID: "m2"},
					{Type: MonitorTypeMessage, AgentID: 7, OccurredAt: "2026-04-16T12:00:01.000000Z", ItemID: "m1"},
				},
				Meta: MonitorMeta{Page: 1, PerPage: 100, Total: 2},
			})
		case 3:
			if query.Get("since") != "2026-04-16T12:00:02.000000Z" {
				t.Fatalf("expected third poll to advance since to latest event, got %q", query.Get("since"))
			}
			json.NewEncoder(w).Encode(monitorListEnvelope{
				Data: []MonitorEvent{
					{Type: MonitorTypeMessage, AgentID: 7, OccurredAt: "2026-04-16T12:00:03.000000Z", ItemID: "m3"},
					{Type: MonitorTypeMessage, AgentID: 7, OccurredAt: "2026-04-16T12:00:02.000000Z", ItemID: "m2"},
				},
				Meta: MonitorMeta{Page: 1, PerPage: 100, Total: 2},
			})
		default:
			json.NewEncoder(w).Encode(monitorListEnvelope{Data: []MonitorEvent{}, Meta: MonitorMeta{Page: 1, PerPage: 100, Total: 0}})
		}
	})

	events, errs := c.Monitor.Follow(context.Background(), nil, &MonitorFollowOptions{
		PollInterval: time.Millisecond,
		Duration:     250 * time.Millisecond,
		MaxEvents:    3,
	})

	got, err := collectMonitorFollow(events, errs)
	if err != nil {
		t.Fatalf("Monitor.Follow: %v", err)
	}

	want := []string{"m1", "m2", "m3"}
	if len(got) != len(want) {
		t.Fatalf("expected %d events, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i].ItemID != want[i] {
			t.Fatalf("expected order %v, got %+v", want, got)
		}
	}
}

func TestMonitor_FollowRespectsExplicitSince(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("since"); got != "2026-04-16T12:00:00Z" {
			t.Fatalf("expected explicit since to be forwarded, got %q", got)
		}
		json.NewEncoder(w).Encode(monitorListEnvelope{
			Data: []MonitorEvent{
				{Type: MonitorTypeMessage, AgentID: 7, OccurredAt: "2026-04-16T12:00:01.000000Z", ItemID: "m1"},
			},
			Meta: MonitorMeta{Page: 1, PerPage: 100, Total: 1},
		})
	})

	events, errs := c.Monitor.Follow(context.Background(), &MonitorListParams{
		Since: "2026-04-16T12:00:00Z",
	}, &MonitorFollowOptions{
		PollInterval: time.Millisecond,
		MaxEvents:    1,
	})

	got, err := collectMonitorFollow(events, errs)
	if err != nil {
		t.Fatalf("Monitor.Follow: %v", err)
	}
	if len(got) != 1 || got[0].ItemID != "m1" {
		t.Fatalf("unexpected events: %+v", got)
	}
}

func TestMonitor_FollowStopsOnOverflow(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(monitorListEnvelope{
			Data: []MonitorEvent{
				{Type: MonitorTypeMessage, AgentID: 7, OccurredAt: "2026-04-16T12:00:01.000000Z", ItemID: "m1"},
			},
			Meta: MonitorMeta{Page: 1, PerPage: 100, Total: 101},
		})
	})

	events, errs := c.Monitor.Follow(context.Background(), nil, &MonitorFollowOptions{
		PollInterval: time.Millisecond,
		Duration:     100 * time.Millisecond,
	})

	got, err := collectMonitorFollow(events, errs)
	if err == nil {
		t.Fatal("expected overflow error")
	}
	if len(got) != 0 {
		t.Fatalf("expected no events on overflow, got %+v", got)
	}
	if !strings.Contains(err.Error(), "narrow filters") || !strings.Contains(err.Error(), "poll more frequently") {
		t.Fatalf("expected actionable overflow error, got %v", err)
	}
}

func collectMonitorFollow(events <-chan MonitorEvent, errs <-chan error) ([]MonitorEvent, error) {
	var out []MonitorEvent
	for events != nil || errs != nil {
		select {
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			out = append(out, ev)
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return out, err
			}
		}
	}
	return out, nil
}
