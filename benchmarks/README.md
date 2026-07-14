# Benchmark baselines

Run representative allocation and latency benchmarks with:

```sh
go test -run '^$' -bench . -benchmem ./fibe ./internal/mcpserver
```

Record results for the release candidate environment before performance work.
Compare with `benchstat`; update code only when contract fixtures remain
identical. Streaming upload/download memory must remain bounded by chunk size,
and pipeline concurrency is independently capped at eight.

The initial release-candidate reference is in
[`baseline-darwin-arm64.md`](baseline-darwin-arm64.md).
