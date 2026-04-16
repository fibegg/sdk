package fibe

import (
	"context"
	"crypto/rand"
	"fmt"
)

type idempotencyKeyCtxKey struct{}

// WithIdempotencyKey returns a context that causes the next request to include
// the given Idempotency-Key header. The API caches the response for 24 hours
// and replays it on duplicate keys (indicated by X-Idempotent-Replayed: true).
//
// Use this for any mutating operation (create, rollout, restart, sync, etc.)
// where a network timeout could leave you unsure whether the action was taken:
//
//	key := fibe.NewIdempotencyKey()
//	ctx := fibe.WithIdempotencyKey(ctx, key)
//	pg, err := client.Playgrounds.Create(ctx, params) // safe to retry
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKeyCtxKey{}, key)
}

// NewIdempotencyKey generates a random idempotency key.
func NewIdempotencyKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func idempotencyKeyFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(idempotencyKeyCtxKey{}).(string); ok {
		return v
	}
	return ""
}
