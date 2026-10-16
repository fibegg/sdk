# Performance baseline: darwin/arm64

Recorded 2026-07-13 on an Apple M5 Pro with Go 1.26.5:

```sh
go test -run '^$' -bench . -benchtime=200ms -count=3 -benchmem ./fibe ./internal/mcpserver ./internal/localconversations
```

The table records the middle observation from three runs. Use the raw command
and `benchstat` for release comparisons; CPU frequency and background load make
single-machine latency values directional rather than universal.

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| BuildQuery | 1,119 | 843 | 25 |
| ProjectFields | 3,424 | 3,686 | 66 |
| NewIdempotencyKey | 214.6 | 48 | 2 |
| DecodeJSONLimitedProjected | 989.1 | 2,824 | 23 |
| MultipartStreaming1MiB | 85,020 | 2,603 | 49 |
| CompileComposeSchemaCached | 49.47 | 0 | 0 |
| PipelineProjection | 671.4 | 1,994 | 26 |
| AuditRedaction | 440.9 | 1,064 | 9 |
| PipelineValidation | 324.4 | 152 | 9 |
| PipelineCacheRoundTrip | 5,952 | 634 | 16 |
| WalkFileCandidates1000 | 4,754,052 | 915,707 | 6,053 |

Multipart reported 12.3 GiB/s while keeping allocations independent of the
1 MiB payload size. The file-scan benchmark covers 1,000 regular files.
