package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestWithFields_FiltersSingleObject(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Playground{
			ID:     42,
			Name:   "my-pg",
			Status: "running",
		})
	})

	ctx := WithFields(context.Background(), "id", "name")
	pg, err := c.Playgrounds.Get(ctx, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pg.ID != 42 {
		t.Errorf("expected ID 42, got %d", pg.ID)
	}
	if pg.Name != "my-pg" {
		t.Errorf("expected name 'my-pg', got %q", pg.Name)
	}
	if pg.Status != "" {
		t.Errorf("expected status to be filtered out (zero value), got %q", pg.Status)
	}
}

func TestWithFields_FiltersListData(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Playground{
			{ID: 1, Name: "a", Status: "running"},
			{ID: 2, Name: "b", Status: "pending"},
		}))
	})

	ctx := WithFields(context.Background(), "id", "status")
	result, err := c.Playgrounds.List(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, pg := range result.Data {
		if pg.Name != "" {
			t.Errorf("expected name to be filtered out, got %q", pg.Name)
		}
		if pg.ID == 0 {
			t.Error("expected ID to be preserved")
		}
		if pg.Status == "" {
			t.Error("expected status to be preserved")
		}
	}
}

func TestWithFields_NoFieldsMeansNoFiltering(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Playground{
			ID:     1,
			Name:   "test",
			Status: "running",
		})
	})

	pg, err := c.Playgrounds.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.Name == "" || pg.Status == "" {
		t.Error("all fields should be present without WithFields")
	}
}

func TestWithFields_UnknownFieldsSilentlyIgnored(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Playground{ID: 1, Name: "test"})
	})

	ctx := WithFields(context.Background(), "id", "nonexistent_field")
	pg, err := c.Playgrounds.Get(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 1 {
		t.Error("id should be preserved")
	}
	if pg.Name != "" {
		t.Error("name should be filtered out")
	}
}

func TestProjectFields_Direct(t *testing.T) {
	pg := Playground{ID: 1, Name: "test", Status: "running"}
	fields := map[string]bool{"id": true, "name": true}

	filtered := ProjectFields(pg, fields)
	if filtered.ID != 1 {
		t.Error("expected ID preserved")
	}
	if filtered.Name != "test" {
		t.Error("expected name preserved")
	}
	if filtered.Status != "" {
		t.Errorf("expected status stripped, got %q", filtered.Status)
	}
}

func TestProjectFields_NilFields(t *testing.T) {
	pg := Playground{ID: 1, Name: "test", Status: "running"}
	filtered := ProjectFields(pg, nil)
	if filtered.Status != "running" {
		t.Error("nil fields should return original value")
	}
}

func TestToYAML_Basic(t *testing.T) {
	pg := Playground{ID: 1, Name: "test", Status: "running"}
	y := ToYAML(pg)

	if !strings.Contains(y, "id: 1") {
		t.Error("expected 'id: 1' in YAML output")
	}
	if !strings.Contains(y, "name: test") {
		t.Error("expected 'name: test' in YAML output")
	}
	if !strings.Contains(y, "status: running") {
		t.Error("expected 'status: running' in YAML output")
	}
	if strings.Contains(y, "{") {
		t.Error("YAML output should not contain JSON braces")
	}
}

func TestToYAML_UsesJSONKeys(t *testing.T) {
	pg := Playground{ID: 1, PlayspecName: strPtr("my-spec")}
	y := ToYAML(pg)

	if !strings.Contains(y, "playspec_name:") {
		t.Errorf("expected snake_case JSON key 'playspec_name', got:\n%s", y)
	}
}

func TestToYAML_List(t *testing.T) {
	items := []Playground{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
	}
	y := ToYAML(items)

	if !strings.Contains(y, "id: 1") {
		t.Errorf("expected 'id: 1' in YAML list output, got:\n%s", y)
	}
	if !strings.Contains(y, "name: a") {
		t.Errorf("expected 'name: a' in YAML list output, got:\n%s", y)
	}
	if !strings.HasPrefix(y, "-") {
		t.Errorf("expected YAML list to start with '-', got:\n%s", y)
	}
}

func TestWithFields_CombinedWithList(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Agent{
			{ID: 1, Name: "agent-1", Provider: "github", Authenticated: true, ProviderLabel: "GitHub"},
		}))
	})

	ctx := WithFields(context.Background(), "id", "name")
	result, err := c.Agents.List(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := result.Data[0]
	if a.ID != 1 {
		t.Error("expected ID preserved")
	}
	if a.Name != "agent-1" {
		t.Error("expected name preserved")
	}
	if a.Provider != "" {
		t.Errorf("expected provider filtered, got %q", a.Provider)
	}
	if a.ProviderLabel != "" {
		t.Errorf("expected provider_label filtered, got %q", a.ProviderLabel)
	}
}

func TestWithFields_DoesNotBreakErrors(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: struct {
				Code    string         `json:"code"`
				Message string         `json:"message"`
				Details map[string]any `json:"details,omitempty"`
			}{Code: ErrCodeNotFound, Message: "not found"},
		})
	})

	ctx := WithFields(context.Background(), "id", "name")
	_, err := c.Playgrounds.Get(ctx, 999)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := err.(*APIError)
	if apiErr.Code != ErrCodeNotFound {
		t.Errorf("expected NOT_FOUND, got %q", apiErr.Code)
	}
}

func TestWithFields_DoesNotBreak204(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	ctx := WithFields(context.Background(), "id")
	err := c.Playgrounds.Delete(ctx, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOutput_YAMLviaToYAML(t *testing.T) {
	secret := Secret{Key: "DB_URL", Value: strPtr("postgres://localhost/db")}
	y := ToYAML(secret)

	if strings.Contains(y, "\"") {
		t.Error("YAML should not have JSON-style quotes")
	}
	if !strings.Contains(y, "key: DB_URL") {
		t.Errorf("expected 'key: DB_URL', got:\n%s", y)
	}
}
