package fibe

import (
	"container/list"
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	monitorSeenCap        = 4096
	monitorBackoffMaxStep = 60 * time.Second
)

type MonitorService struct {
	client *Client
}

// List returns a single page of monitor events ordered newest first.
func (s *MonitorService) List(ctx context.Context, params *MonitorListParams) (*MonitorListResult, error) {
	path := "/api/monitor" + buildQuery(params)
	var env monitorListEnvelope
	if err := s.client.do(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &MonitorListResult{Data: env.Data, Meta: env.Meta}, nil
}

// Follow polls GET /api/monitor with a rolling "since" watermark.
//
// The returned error channel is used for unrecoverable failures such as
// follow-window overflow (more matching events than one poll can safely hold).
// Transient request failures are retried with exponential backoff.
func (s *MonitorService) Follow(ctx context.Context, params *MonitorListParams, opts *MonitorFollowOptions) (<-chan MonitorEvent, <-chan error) {
	events := make(chan MonitorEvent, 64)
	errs := make(chan error, 1)

	if opts == nil {
		opts = &MonitorFollowOptions{}
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}

	req := MonitorListParams{}
	if params != nil {
		req = *params
	}
	req.Page = 1
	req.PerPage = 100
	if req.Since == "" {
		req.Since = time.Now().UTC().Format(time.RFC3339Nano)
	}

	streamCtx := ctx
	var cancel context.CancelFunc
	if opts.Duration > 0 {
		streamCtx, cancel = context.WithTimeout(ctx, opts.Duration)
	}

	go func() {
		defer close(events)
		defer close(errs)
		if cancel != nil {
			defer cancel()
		}

		emitted := 0
		seen := newBoundedSet(monitorSeenCap)
		failures := 0

		for {
			if streamCtx.Err() != nil {
				return
			}

			batch, err := s.List(streamCtx, &req)
			if err != nil {
				failures++
				select {
				case <-streamCtx.Done():
					return
				case <-time.After(backoff(opts.PollInterval, failures)):
					continue
				}
			}
			failures = 0

			if batch.Meta.Total > req.PerPage {
				sendMonitorFollowError(streamCtx, errs, fmt.Errorf(
					"monitor follow overflow: got %d matching events in one poll window (per_page=%d); narrow filters or poll more frequently",
					batch.Meta.Total,
					req.PerPage,
				))
				return
			}

			latestOccurredAt := ""
			for idx := len(batch.Data) - 1; idx >= 0; idx-- {
				ev := batch.Data[idx]
				key := monitorEventKey(ev)
				if seen.contains(key) {
					continue
				}
				seen.add(key)
				latestOccurredAt = ev.OccurredAt
				select {
				case events <- ev:
					emitted++
				case <-streamCtx.Done():
					return
				}
				if opts.MaxEvents > 0 && emitted >= opts.MaxEvents {
					return
				}
			}

			if latestOccurredAt != "" {
				req.Since = latestOccurredAt
			}

			select {
			case <-streamCtx.Done():
				return
			case <-time.After(opts.PollInterval):
			}
		}
	}()

	return events, errs
}

func backoff(base time.Duration, failures int) time.Duration {
	if base <= 0 {
		base = 2 * time.Second
	}
	step := base
	for i := 1; i < failures && step < monitorBackoffMaxStep; i++ {
		step *= 2
	}
	if step > monitorBackoffMaxStep {
		step = monitorBackoffMaxStep
	}
	return step
}

func sendMonitorFollowError(ctx context.Context, errs chan<- error, err error) {
	if err == nil {
		return
	}
	select {
	case errs <- err:
	case <-ctx.Done():
	}
}

func monitorEventKey(ev MonitorEvent) string {
	return fmt.Sprintf("%s|%d|%s|%s", ev.Type, ev.AgentID, ev.ItemID, ev.OccurredAt)
}

// boundedSet is a fixed-capacity, insertion-ordered set used to dedup
// event identities across a long-running Follow loop without unbounded growth.
type boundedSet struct {
	cap   int
	order *list.List
	index map[string]*list.Element
}

func newBoundedSet(capacity int) *boundedSet {
	if capacity <= 0 {
		capacity = 1024
	}
	return &boundedSet{
		cap:   capacity,
		order: list.New(),
		index: make(map[string]*list.Element, capacity),
	}
}

func (b *boundedSet) contains(key string) bool {
	_, ok := b.index[key]
	return ok
}

func (b *boundedSet) add(key string) {
	if _, ok := b.index[key]; ok {
		return
	}
	elem := b.order.PushBack(key)
	b.index[key] = elem
	for b.order.Len() > b.cap {
		front := b.order.Front()
		if front == nil {
			break
		}
		b.order.Remove(front)
		delete(b.index, front.Value.(string))
	}
}
