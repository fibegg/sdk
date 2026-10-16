package fibetest

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
)

func TestMockServerRoutesAndInterceptorPrecedence(t *testing.T) {
	mock := NewMockServer()
	t.Cleanup(mock.Close)
	mock.Interceptors["/api/status"] = func(w http.ResponseWriter, _ *http.Request) {
		writeMockJSON(w, map[string]string{"status": "intercepted"})
	}
	response, err := http.Get(mock.URL() + "/api/status")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "intercepted" {
		t.Fatalf("interceptor did not run first: %#v", body)
	}
}

func TestMockServerRecordsConcurrentRequests(t *testing.T) {
	mock := NewMockServer()
	t.Cleanup(mock.Close)
	const count = 32
	var group sync.WaitGroup
	for range count {
		group.Add(1)
		go func() {
			defer group.Done()
			response, err := http.Get(mock.URL() + "/api/status")
			if err == nil {
				response.Body.Close()
			}
		}()
	}
	group.Wait()
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.seen) != count {
		t.Fatalf("recorded %d requests, want %d", len(mock.seen), count)
	}
}

func TestMockServerStructuredNotFound(t *testing.T) {
	mock := NewMockServer()
	t.Cleanup(mock.Close)
	response, err := http.Get(mock.URL() + "/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "MOCK_ROUTE_NOT_FOUND" {
		t.Fatalf("code = %q", body.Error.Code)
	}
}

func TestMockServerCloseIsIdempotent(t *testing.T) {
	mock := NewMockServer()
	mock.Close()
	mock.Close()
	if _, err := http.Get(mock.URL() + "/api/status"); err == nil {
		t.Fatal("request unexpectedly succeeded after shutdown")
	}
}
