package fibe

import (
	"context"
	"time"
)

// LogLine is a single streamed log line kept for compatibility with older
// callers. New code should prefer LogStreamEvent.
type LogLine struct {
	Text   string
	Source string
}

type LogStreamEvent struct {
	Type       string    `json:"type"`
	Line       string    `json:"line,omitempty"`
	Stream     string    `json:"stream,omitempty"`
	Service    string    `json:"service,omitempty"`
	Timestamp  string    `json:"timestamp,omitempty"`
	Status     string    `json:"status,omitempty"`
	Message    string    `json:"message,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
}

// LogsStreamOptions controls LogsStream polling behavior.
type LogsStreamOptions struct {
	// Tail is the initial number of lines to fetch. Defaults to 50.
	Tail int

	// PollInterval is retained for compatibility with polling-backed SDK
	// versions. ActionCable-backed streams ignore it.
	PollInterval time.Duration

	// MaxLines caps the total number of log lines sent on the channel. 0 = unlimited.
	MaxLines int
}

func (s *PlaygroundService) LogsStream(ctx context.Context, id int64, service string, opts *LogsStreamOptions) <-chan LogLine {
	return s.LogsStreamByIdentifier(ctx, int64Identifier(id), service, opts)
}

func (s *PlaygroundService) LogsStreamByIdentifier(ctx context.Context, identifier string, service string, opts *LogsStreamOptions) <-chan LogLine {
	ch := make(chan LogLine, 64)
	go func() {
		defer close(ch)
		events, errs := s.LogStreamByIdentifier(ctx, identifier, service, opts)
		for events != nil || errs != nil {
			select {
			case ev, ok := <-events:
				if !ok {
					events = nil
					continue
				}
				if ev.Type != "log" {
					continue
				}
				source := ev.Stream
				if source == "" {
					source = "live"
				}
				select {
				case ch <- LogLine{Text: ev.Line, Source: source}:
				case <-ctx.Done():
					return
				}
			case _, ok := <-errs:
				if !ok {
					errs = nil
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func (s *PlaygroundService) LogStream(ctx context.Context, id int64, service string, opts *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error) {
	return s.logStreamByID(ctx, id, service, opts)
}

func (s *PlaygroundService) LogStreamByIdentifier(ctx context.Context, identifier string, service string, opts *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error) {
	events := make(chan LogStreamEvent, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		playground, err := s.GetByIdentifier(ctx, identifier)
		if err != nil {
			sendLogStreamError(ctx, errs, err)
			return
		}
		streamEvents(ctx, s.client.Cable.SubscribeLogStream, playground.ID, service, normalizedLogsStreamOptions(opts), events, errs)
	}()
	return events, errs
}

func (s *PlaygroundService) logStreamByID(ctx context.Context, id int64, service string, opts *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error) {
	events := make(chan LogStreamEvent, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		streamEvents(ctx, s.client.Cable.SubscribeLogStream, id, service, normalizedLogsStreamOptions(opts), events, errs)
	}()
	return events, errs
}

func (s *TrickService) LogsStream(ctx context.Context, id int64, service string, opts *LogsStreamOptions) <-chan LogLine {
	return s.LogsStreamByIdentifier(ctx, int64Identifier(id), service, opts)
}

func (s *TrickService) LogsStreamByIdentifier(ctx context.Context, identifier string, service string, opts *LogsStreamOptions) <-chan LogLine {
	return s.client.Playgrounds.LogsStreamByIdentifier(ctx, identifier, service, opts)
}

func (s *TrickService) LogStream(ctx context.Context, id int64, service string, opts *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error) {
	return s.client.Playgrounds.logStreamByID(ctx, id, service, opts)
}

func (s *TrickService) LogStreamByIdentifier(ctx context.Context, identifier string, service string, opts *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error) {
	events := make(chan LogStreamEvent, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		trick, err := s.GetByIdentifier(ctx, identifier)
		if err != nil {
			sendLogStreamError(ctx, errs, err)
			return
		}
		streamEvents(ctx, s.client.Cable.SubscribeLogStream, trick.ID, service, normalizedLogsStreamOptions(opts), events, errs)
	}()
	return events, errs
}

func normalizedLogsStreamOptions(opts *LogsStreamOptions) *LogsStreamOptions {
	if opts == nil {
		return &LogsStreamOptions{}
	}
	return &LogsStreamOptions{
		Tail:         opts.Tail,
		PollInterval: opts.PollInterval,
		MaxLines:     opts.MaxLines,
	}
}

func streamEvents(ctx context.Context, subscribe func(context.Context, int64, string, *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error), id int64, service string, opts *LogsStreamOptions, out chan<- LogStreamEvent, errs chan<- error) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, streamErrs := subscribe(streamCtx, id, service, opts)
	lineCount := 0
	for events != nil || streamErrs != nil {
		select {
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			select {
			case out <- ev:
			case <-streamCtx.Done():
				return
			}
			if ev.Type == "log" {
				lineCount++
				if opts.MaxLines > 0 && lineCount >= opts.MaxLines {
					return
				}
			}
		case err, ok := <-streamErrs:
			if !ok {
				streamErrs = nil
				continue
			}
			if err != nil {
				sendLogStreamError(streamCtx, errs, err)
				return
			}
		case <-streamCtx.Done():
			return
		}
	}
}

func sendLogStreamError(ctx context.Context, errs chan<- error, err error) {
	if err == nil {
		return
	}
	select {
	case errs <- err:
	case <-ctx.Done():
	}
}
