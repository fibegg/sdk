package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func newPipelineHardeningServer(t *testing.T, cfg Config) *Server {
	t.Helper()
	if cfg.ToolSet == "" {
		cfg.ToolSet = "full"
	}
	if cfg.APIKey == "" {
		cfg.APIKey = "test"
	}
	srv := New(cfg)
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	return srv
}

func TestPipelinePreflightRejectsDuplicateIDsBeforeSideEffects(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 10})
	var calls atomic.Int32
	srv.dispatcher.register(&toolImpl{name: "test_side_effect", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		calls.Add(1)
		return map[string]any{"ok": true}, nil
	}})
	_, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{
		map[string]any{"id": "same", "tool": "test_side_effect"},
		map[string]any{"id": "same", "tool": "test_side_effect"},
	}})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight allowed %d side effects", calls.Load())
	}
}

func TestPipelinePreflightCountsNestedSteps(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 2})
	var calls atomic.Int32
	srv.dispatcher.register(&toolImpl{name: "test_count", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		calls.Add(1)
		return nil, nil
	}})
	_, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{
		map[string]any{"parallel": []any{
			map[string]any{"id": "a", "tool": "test_count"},
			map[string]any{"id": "b", "tool": "test_count"},
			map[string]any{"id": "c", "tool": "test_count"},
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "max steps") {
		t.Fatalf("error = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight allowed %d side effects", calls.Load())
	}
}

func TestPipelineRejectsParallelSiblingDependency(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 4})
	srv.dispatcher.register(&toolImpl{name: "test_echo", tier: tierMeta, handler: func(_ context.Context, _ *fibe.Client, args map[string]any) (any, error) { return args, nil }})
	_, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{
		map[string]any{"parallel": []any{
			map[string]any{"id": "a", "tool": "test_echo", "args": map[string]any{"value": 1}},
			map[string]any{"id": "b", "tool": "test_echo", "args": map[string]any{"value": "$.a.value"}},
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "parallel sibling") {
		t.Fatalf("error = %v", err)
	}
}

func TestPipelineBoundsConcurrentToolCalls(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 10, PipelineMaxConcurrency: 2})
	var current atomic.Int32
	var maximum atomic.Int32
	srv.dispatcher.register(&toolImpl{name: "test_slow", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		now := current.Add(1)
		for {
			old := maximum.Load()
			if now <= old || maximum.CompareAndSwap(old, now) {
				break
			}
		}
		defer current.Add(-1)
		time.Sleep(15 * time.Millisecond)
		return map[string]any{"ok": true}, nil
	}})
	parallel := make([]any, 5)
	for i := range parallel {
		parallel[i] = map[string]any{"id": fmt.Sprintf("step_%d", i), "tool": "test_slow"}
	}
	result, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{map[string]any{"parallel": parallel}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]any)["status"] != "completed" {
		t.Fatalf("result = %#v", result)
	}
	if maximum.Load() > 2 {
		t.Fatalf("maximum concurrency = %d", maximum.Load())
	}
}

func TestPipelineParallelFailureIsDeterministicByDeclarationOrder(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 4, PipelineMaxConcurrency: 4})
	srv.dispatcher.register(&toolImpl{name: "test_fail_slow", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		time.Sleep(20 * time.Millisecond)
		return nil, errors.New("first declared failure")
	}})
	srv.dispatcher.register(&toolImpl{name: "test_fail_fast", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		return nil, errors.New("second declared failure")
	}})
	result, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{
		map[string]any{"parallel": []any{
			map[string]any{"id": "first", "tool": "test_fail_slow"},
			map[string]any{"id": "second", "tool": "test_fail_fast"},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	errorInfo := result.(map[string]any)["error"].(map[string]any)
	if !strings.Contains(errorInfo["message"].(string), "first declared failure") {
		t.Fatalf("error = %#v", errorInfo)
	}
}

func TestPipelineForEachUsesDistinctDeterministicIdempotencyKeys(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 10, PipelineMaxIterations: 10})
	srv.dispatcher.register(&toolImpl{name: "test_items", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		return map[string]any{"items": []any{"a", "b"}}, nil
	}})
	var mu sync.Mutex
	keys := map[string]string{}
	srv.dispatcher.register(&toolImpl{name: "test_capture_iteration", tier: tierMeta, handler: func(ctx context.Context, _ *fibe.Client, args map[string]any) (any, error) {
		item := args["item"].(string)
		mu.Lock()
		keys[item] = idempotencyKeyFromCtxForTest(ctx)
		mu.Unlock()
		return map[string]any{"item": item}, nil
	}})
	run := func() {
		result, err := srv.runPipeline(context.Background(), map[string]any{
			"idempotency_key": "pipeline-key",
			"steps": []any{
				map[string]any{"id": "seed", "tool": "test_items"},
				map[string]any{"id": "fan", "for_each": "$.seed.items", "as": "item", "steps": []any{
					map[string]any{"id": "capture", "tool": "test_capture_iteration", "args": map[string]any{"item": "$.item"}},
				}},
			},
		})
		if err != nil || result.(map[string]any)["status"] != "completed" {
			t.Fatalf("result=%#v err=%v", result, err)
		}
	}
	run()
	firstA, firstB := keys["a"], keys["b"]
	if firstA == "" || firstB == "" || firstA == firstB {
		t.Fatalf("iteration keys = %#v", keys)
	}
	keys = map[string]string{}
	run()
	if keys["a"] != firstA || keys["b"] != firstB {
		t.Fatalf("iteration keys are not deterministic: %#v", keys)
	}
}

func TestPipelineRuntimeStepLimitCoversForEachExpansion(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 2, PipelineMaxIterations: 10})
	srv.dispatcher.register(&toolImpl{name: "test_many_items", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		return map[string]any{"items": []any{1, 2}}, nil
	}})
	srv.dispatcher.register(&toolImpl{name: "test_noop", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) { return nil, nil }})
	result, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{
		map[string]any{"id": "seed", "tool": "test_many_items"},
		map[string]any{"id": "fan", "for_each": "$.seed.items", "as": "item", "steps": []any{
			map[string]any{"id": "noop", "tool": "test_noop"},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["status"] != "partial" || !strings.Contains(response["error"].(map[string]any)["message"].(string), "max steps") {
		t.Fatalf("response = %#v", response)
	}
}

func TestPipelineProjectionErrorsAreNotSilentlyIgnored(t *testing.T) {
	srv := newPipelineHardeningServer(t, Config{PipelineMaxSteps: 2})
	srv.dispatcher.register(&toolImpl{name: "test_object", tier: tierMeta, handler: func(context.Context, *fibe.Client, map[string]any) (any, error) {
		return map[string]any{"value": 1}, nil
	}})
	result, err := srv.runPipeline(context.Background(), map[string]any{"steps": []any{
		map[string]any{"id": "object", "tool": "test_object", "output_path": "$.missing"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["status"] != "partial" || !strings.Contains(response["error"].(map[string]any)["message"].(string), "output_path") {
		t.Fatalf("response = %#v", response)
	}
}
