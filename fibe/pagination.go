package fibe

import (
	"context"
	"fmt"
	"net/http"
)

// ListMeta contains pagination metadata returned by all list endpoints.
type ListMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

// listEnvelope is the standard response wrapper for all list endpoints.
// Every list endpoint returns {"data": [...], "meta": {"page", "per_page", "total"}}.
type listEnvelope[T any] struct {
	Data []T      `json:"data"`
	Meta ListMeta `json:"meta"`
}

// ListResult is a single page of results with pagination metadata.
type ListResult[T any] struct {
	Data []T
	Meta ListMeta
}

// doList performs a GET request to a list endpoint and decodes the
// standard {data, meta} envelope.
func doList[T any](c *Client, ctx context.Context, path string) (*ListResult[T], error) {
	var env listEnvelope[T]
	if err := c.do(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &ListResult[T]{Data: env.Data, Meta: env.Meta}, nil
}

// Iterator pages through a list endpoint automatically.
type Iterator[T any] struct {
	client  *Client
	path    string
	page    int
	perPage int
	items   []T
	index   int
	total   int64
	done    bool
	err     error
	ctx     context.Context
}

func newIterator[T any](ctx context.Context, client *Client, path string, perPage int) *Iterator[T] {
	if perPage <= 0 {
		perPage = 25
	}
	return &Iterator[T]{
		client:  client,
		path:    path,
		page:    1,
		perPage: perPage,
		ctx:     ctx,
	}
}

func (it *Iterator[T]) Next() bool {
	if it.err != nil || it.done {
		return false
	}

	if it.index < len(it.items) {
		return true
	}

	if it.page > 1 && len(it.items) == 0 {
		it.done = true
		return false
	}

	path := fmt.Sprintf("%s?page=%d&per_page=%d", it.path, it.page, it.perPage)

	var env listEnvelope[T]
	if err := it.client.do(it.ctx, http.MethodGet, path, nil, &env); err != nil {
		it.err = err
		return false
	}

	it.total = env.Meta.Total

	if len(env.Data) == 0 {
		it.done = true
		return false
	}

	it.items = env.Data
	it.index = 0
	it.page++

	return true
}

func (it *Iterator[T]) Current() T {
	item := it.items[it.index]
	it.index++
	return item
}

func (it *Iterator[T]) Err() error {
	return it.err
}

// Total returns the total count from the API. Only valid after at least one Next() call.
func (it *Iterator[T]) Total() int64 {
	return it.total
}

func (it *Iterator[T]) Collect() ([]T, error) {
	var all []T
	for it.Next() {
		all = append(all, it.Current())
	}
	return all, it.Err()
}
