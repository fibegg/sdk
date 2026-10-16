package mcpserver

import (
	"encoding/json"
	"math"
	"testing"
)

func FuzzArgumentCoercion(f *testing.F) {
	for _, seed := range []float64{0, 1, -1, math.MaxInt64, math.MaxFloat64, math.NaN(), math.Inf(1)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value float64) {
		args := map[string]any{"value": value}
		_, _ = argInt64(args, "value")
		var target struct {
			Value int64 `json:"value"`
		}
		_ = bindArgs(args, &target)
	})
}

func FuzzPipelineProjection(f *testing.F) {
	f.Add(`{"step":{"id":1}}`, "$.step.id")
	f.Add(`[]`, "$")
	f.Fuzz(func(t *testing.T, raw, path string) {
		if len(raw) > 1<<20 || len(path) > 4096 {
			t.Skip()
		}
		var value any
		if json.Unmarshal([]byte(raw), &value) != nil {
			return
		}
		_, _ = projectOnMap(value, path)
	})
}
