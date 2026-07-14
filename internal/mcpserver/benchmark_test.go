package mcpserver

import (
	"context"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func BenchmarkPipelineProjection(b *testing.B) {
	value := map[string]any{"step": map[string]any{"id": float64(42), "name": "benchmark"}}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = projectOnMap(value, "$.step.id")
	}
}

func BenchmarkAuditRedaction(b *testing.B) {
	value := map[string]any{
		"payload": map[string]any{"name": "demo", "api_key": "secret", "nested": []any{map[string]any{"token": "secret"}}},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = redactSensitive(value)
	}
}

func BenchmarkPipelineValidation(b *testing.B) {
	server := New(DefaultConfig())
	server.dispatcher.register(&toolImpl{name: "benchmark", handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		return nil, nil
	}})
	steps := []pipelineStep{
		{ID: "first", Tool: "benchmark", Args: map[string]any{"name": "fixture"}},
		{ID: "parallel", Parallel: []pipelineStep{
			{ID: "left", Tool: "benchmark"},
			{ID: "right", Tool: "benchmark"},
		}},
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := server.validatePipeline(steps); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPipelineCacheRoundTrip(b *testing.B) {
	cache := newPipelineCache(1024, 1<<20)
	value := map[string]any{"status": "success", "result": map[string]any{"id": 42, "name": "benchmark"}}
	b.ReportAllocs()
	for b.Loop() {
		id, _, err := cache.Put("benchmark-session", value)
		if err != nil {
			b.Fatal(err)
		}
		if _, ok := cache.Get("benchmark-session", id); !ok {
			b.Fatal("cache miss")
		}
	}
}
