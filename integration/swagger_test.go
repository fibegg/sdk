package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestSwagger_Endpoint(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set")
	}

	t.Run("returns swagger spec", func(t *testing.T) {
		t.Parallel()
		base := c.BaseURL()
		req, err := http.NewRequestWithContext(ctx(), "GET", base+"/api/swagger", nil)
		requireNoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		requireNoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		requireNoError(t, err)

		var spec map[string]any
		err = json.Unmarshal(body, &spec)
		requireNoError(t, err, "swagger response should be valid JSON")

		if _, ok := spec["paths"]; !ok {
			if _, ok := spec["openapi"]; !ok {
				if _, ok := spec["swagger"]; !ok {
					t.Error("expected swagger/openapi spec with 'paths', 'openapi', or 'swagger' key")
				}
			}
		}
	})

	t.Run("returns 401 without auth", func(t *testing.T) {
		t.Parallel()
		base := c.BaseURL()
		req, err := http.NewRequestWithContext(ctx(), "GET", base+"/api/swagger", nil)
		requireNoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		requireNoError(t, err)
		resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
		}
	})
}
