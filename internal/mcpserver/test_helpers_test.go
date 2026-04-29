package mcpserver

import (
	"os"
	"testing"
)

func requireRealServer(t *testing.T) (string, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	apiKey := os.Getenv("FIBE_API_KEY")
	domain := os.Getenv("FIBE_DOMAIN")
	if apiKey == "" || domain == "" {
		t.Skip("FIBE_API_KEY and FIBE_DOMAIN must be set for this test")
	}
	return apiKey, domain
}

func mockServerConfig() Config {
	return Config{
		APIKey:  "pk_test_mock",
		Domain:  "http://127.0.0.1:65535", // A blackhole port that will refuse connection if hit
		ToolSet: "core",
	}
}
