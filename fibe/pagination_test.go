package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIterator_CollectsAllPages(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		switch page {
		case 1:
			json.NewEncoder(w).Encode(listEnv([]Playground{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}))
		case 2:
			json.NewEncoder(w).Encode(listEnv([]Playground{{ID: 3, Name: "c"}}))
		case 3:
			json.NewEncoder(w).Encode(listEnv([]Playground{}))
		}
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("test"), WithBaseURL(srv.URL), WithMaxRetries(0))
	iter := newIterator[Playground](context.Background(), c, "/api/playgrounds", 2)

	all, err := iter.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}
}

func TestIterator_StopsOnEmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("test"), WithBaseURL(srv.URL), WithMaxRetries(0))
	iter := newIterator[Playground](context.Background(), c, "/api/playgrounds", 25)

	all, err := iter.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 items, got %d", len(all))
	}
}

func TestIterator_PropagatesErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: struct {
				Code    string         `json:"code"`
				Message string         `json:"message"`
				Details map[string]any `json:"details,omitempty"`
			}{Code: ErrCodeInternalError, Message: "boom"},
		})
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("test"), WithBaseURL(srv.URL), WithMaxRetries(0))
	iter := newIterator[Playground](context.Background(), c, "/api/playgrounds", 25)

	_, err := iter.Collect()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIterator_ManualIteration(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			json.NewEncoder(w).Encode(listEnv([]Playground{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}))
		} else {
			json.NewEncoder(w).Encode(listEnv([]Playground{}))
		}
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("test"), WithBaseURL(srv.URL), WithMaxRetries(0))
	iter := newIterator[Playground](context.Background(), c, "/api/playgrounds", 25)

	count := 0
	for iter.Next() {
		pg := iter.Current()
		count++
		if pg.ID == 0 {
			t.Error("expected non-zero ID")
		}
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 items, got %d", count)
	}
}

func TestIterator_TracksTotal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnvelope[Playground]{
			Data: []Playground{{ID: 1}},
			Meta: ListMeta{Page: 1, PerPage: 25, Total: 100},
		})
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("test"), WithBaseURL(srv.URL), WithMaxRetries(0))
	iter := newIterator[Playground](context.Background(), c, "/api/playgrounds", 25)

	iter.Next()
	_ = iter.Current()

	if iter.Total() != 100 {
		t.Errorf("expected total 100, got %d", iter.Total())
	}
}
