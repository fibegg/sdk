// Package fibetest provides testing utilities for downstream consumers of the
// Fibe Go SDK. It allows developers to test their own systems without
// requiring an active internet connection to Fibe's production servers.
package fibetest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

// MockServer implements an httptest.Server that returns mock payloads
// for standard Fibe REST API calls. 
type MockServer struct {
	server *httptest.Server
	Mux    *http.ServeMux

	// Interceptors allow you to override specific routes
	Interceptors map[string]http.HandlerFunc
}

// NewMockServer boots an in-memory HTTP server attached to localhost.
// Use the returned URL to configure your Fibe Client:
//
//	mock := fibetest.NewMockServer()
//	defer mock.Close()
//	client := fibe.NewClient(fibe.WithDomain(mock.Domain()))
//
func NewMockServer() *MockServer {
	m := &MockServer{
		Mux:          http.NewServeMux(),
		Interceptors: make(map[string]http.HandlerFunc),
	}

	m.Mux.HandleFunc("/", m.handleDefault)
	m.server = httptest.NewServer(m.Mux)
	return m
}

// Close shuts down the mock server.
func (m *MockServer) Close() {
	m.server.Close()
}

// URL returns the full mock server HTTP URL.
func (m *MockServer) URL() string {
	return m.server.URL
}

// Domain returns the domain portion of the URL, suitable for `fibe.WithDomain()`.
func (m *MockServer) Domain() string {
	return strings.TrimPrefix(m.server.URL, "http://")
}

// handleDefault provides basic success responses for standard endpoints if no
// interceptor is configured.
func (m *MockServer) handleDefault(w http.ResponseWriter, r *http.Request) {
	if interceptor, ok := m.Interceptors[r.URL.Path]; ok {
		interceptor(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", "req_mock_123")

	switch {
	case r.URL.Path == "/api/status" || r.URL.Path == "/api/me":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"id": 1, "username": "mock_user", "email": "mock@example.com"})

	case strings.HasSuffix(r.URL.Path, "/playgrounds") && r.Method == "GET":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []fibe.Playground{{ID: 42, Name: "mock-pg", Status: "running"}},
			"meta": map[string]int{"page": 1, "per_page": 25, "total": 1},
		})

	case strings.HasPrefix(r.URL.Path, "/api/playgrounds/") && r.Method == "GET":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fibe.Playground{ID: 42, Name: "mock-pg", Status: "running"})

	default:
		// Generic fallback success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
