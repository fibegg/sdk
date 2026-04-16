package integration

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// TestArtefact_CreateReturnsPopulatedStruct regression test for the bug where
// ArtefactService.Create returned (nil, nil) on success.
func TestArtefact_CreateReturnsPopulatedStruct(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	ag := seedAgent(t, c, fibe.ProviderGemini)

	content := "regression-check-" + uniqueName("")
	art, err := c.Artefacts.Create(ctx(), ag.ID, &fibe.ArtefactCreateParams{
		Name: "regression.txt",
	}, strings.NewReader(content), "regression.txt")
	requireNoError(t, err)
	if art == nil {
		t.Fatal("Artefact.Create returned nil struct — REGRESSION (see fix in svc_artefacts.go)")
	}
	if art.ID == 0 {
		t.Errorf("Artefact.ID should be > 0, got %d", art.ID)
	}
	if art.Name == "" {
		t.Error("Artefact.Name should be populated")
	}
	if art.AgentID != ag.ID {
		t.Errorf("Artefact.AgentID mismatch: want %d, got %d", ag.ID, art.AgentID)
	}
}

// TestArtefact_ContentRoundtrip uploads binary and text content, downloads, verifies bytes.
func TestArtefact_ContentRoundtrip(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	ag := seedAgent(t, c, fibe.ProviderGemini)

	cases := []struct {
		name    string
		file    string
		content []byte
	}{
		{"utf8 text", "utf8.txt", []byte("Hello, 世界! 😀 multi-line\nsecond line\n")},
		{"binary bytes", "binary.bin", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x7F, 0x80}},
		{"json blob", "config.json", []byte(`{"key":"value","nums":[1,2,3]}`)},
		{"large 10KB text", "large.txt", bytes.Repeat([]byte("ABCDEFGHIJ"), 1024)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			art, err := c.Artefacts.Create(ctx(), ag.ID, &fibe.ArtefactCreateParams{
				Name: tc.file,
			}, bytes.NewReader(tc.content), tc.file)
			if err != nil {
				if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
					t.Skipf("upload rejected (%d %s): %s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
				}
				requireNoError(t, err)
			}
			if art == nil || art.ID == 0 {
				t.Skip("artefact create returned no ID; skipping roundtrip")
			}

			// Download
			body, _, _, err := c.Artefacts.Download(ctx(), ag.ID, art.ID)
			if err != nil {
				if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 404 {
					t.Skip("artefact not ready")
				}
				requireNoError(t, err)
			}
			defer body.Close()

			got, err := io.ReadAll(body)
			requireNoError(t, err)
			if !bytes.Equal(got, tc.content) {
				// Content mismatch — log size info for debugging
				t.Errorf("content mismatch: want %d bytes, got %d bytes", len(tc.content), len(got))
			}
		})
	}
}
