package fibe

import (
	"context"
	"time"
)

// LogLine is a single streamed log line. Source mirrors the `source` field
// on PlaygroundLogs ("cache" / "live"), which remains useful for callers
// distinguishing replayed history from live tail.
type LogLine struct {
	Text   string
	Source string
}

// LogsStreamOptions controls LogsStream polling behavior.
type LogsStreamOptions struct {
	// Tail is the initial number of lines to fetch. Defaults to 50.
	Tail int

	// PollInterval is the gap between /logs fetches. Defaults to 2s.
	PollInterval time.Duration

	// MaxLines caps the total number of lines sent on the channel. 0 = unlimited.
	MaxLines int
}

// LogsStream returns a channel that emits new log lines from a playground
// service as they arrive. The channel is closed when ctx is cancelled, the
// server returns an error, or MaxLines is reached.
//
// The initial batch (the last `Tail` lines) is replayed on the channel
// first — callers that only want "new" lines after connect can discard
// lines where Source == "cache".
//
// Implementation detail: the SDK does not have a true streaming logs
// endpoint today, so this polls /logs on an interval and emits any new
// tail entries (de-duplicated by content). Once the server gains an SSE
// endpoint this function will be swapped to consume it without callers
// needing to change.
func (s *PlaygroundService) LogsStream(ctx context.Context, id int64, service string, opts *LogsStreamOptions) <-chan LogLine {
	if opts == nil {
		opts = &LogsStreamOptions{}
	}
	if opts.Tail <= 0 {
		opts.Tail = 50
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}

	ch := make(chan LogLine, 64)
	go func() {
		defer close(ch)

		var lastSig string
		var sent int
		first := true

		for {
			tail := opts.Tail
			logs, err := s.Logs(ctx, id, service, &tail)
			if err != nil {
				return
			}

			// Find the first line we haven't seen yet.
			start := 0
			if lastSig != "" {
				for i, line := range logs.Lines {
					if line == lastSig {
						start = i + 1
					}
				}
			}

			for _, line := range logs.Lines[start:] {
				source := logs.Source
				if first {
					// Mark replayed lines as cache on the first pass so
					// callers can distinguish them from "live" additions.
					if source == "" {
						source = "cache"
					}
				}
				select {
				case ch <- LogLine{Text: line, Source: source}:
					sent++
					lastSig = line
					if opts.MaxLines > 0 && sent >= opts.MaxLines {
						return
					}
				case <-ctx.Done():
					return
				}
			}
			first = false

			select {
			case <-ctx.Done():
				return
			case <-time.After(opts.PollInterval):
			}
		}
	}()

	return ch
}

// LogsStream is the equivalent method on TrickService. Tricks share the
// playground log endpoint so the implementation delegates to PlaygroundService.
func (s *TrickService) LogsStream(ctx context.Context, id int64, service string, opts *LogsStreamOptions) <-chan LogLine {
	return s.client.Playgrounds.LogsStream(ctx, id, service, opts)
}
