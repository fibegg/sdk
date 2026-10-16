package fibe

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func BenchmarkBuildQuery(b *testing.B) {
	params := &PlaygroundListParams{Page: 3, PerPage: 100, Status: "running", CreatedAfter: "2026-01-02T03:04:05Z"}
	b.ReportAllocs()
	for b.Loop() {
		_ = buildQuery(params)
	}
}

func BenchmarkProjectFields(b *testing.B) {
	value := Playground{ID: 42, Name: "benchmark", Status: "running", EnvOverrides: map[string]string{"APP_ENV": "benchmark"}}
	fields := map[string]bool{"id": true, "name": true, "status": true}
	b.ReportAllocs()
	for b.Loop() {
		_ = ProjectFields(value, fields)
	}
}

func BenchmarkNewIdempotencyKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = NewIdempotencyKey()
	}
}

func BenchmarkDecodeJSONLimitedProjected(b *testing.B) {
	payload := []byte(`{"id":42,"name":"benchmark","status":"running","env_overrides":{"APP_ENV":"benchmark"}}`)
	b.ReportAllocs()
	for b.Loop() {
		var value Playground
		if err := decodeJSONLimitedProjected(context.Background(), bytes.NewReader(payload), maxResponseBody, "benchmark", &value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMultipartStreaming1MiB(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 1<<20)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		body, _ := streamMultipartBody(map[string]string{"name": "benchmark"}, "file", "fixture.bin", bytes.NewReader(payload))
		if _, err := io.Copy(io.Discard, body); err != nil {
			b.Fatal(err)
		}
		if err := body.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileComposeSchemaCached(b *testing.B) {
	schema := []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"services":{"type":"object"}}}`)
	if _, err := compileComposeSchema(schema, "https://example.test/schema.json"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := compileComposeSchema(schema, "https://example.test/schema.json"); err != nil {
			b.Fatal(err)
		}
	}
}
