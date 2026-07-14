package fibe

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

type idempotencyKeyCtxKey struct{}

var idempotencyFallbackCounter atomic.Uint64

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
	return newIdempotencyKey(rand.Reader, time.Now())
}

func newIdempotencyKey(random io.Reader, now time.Time) string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(random, b); err == nil {
		return hex.EncodeToString(b)
	}

	seed := fmt.Sprintf("%d:%d:%d", os.Getpid(), now.UnixNano(), idempotencyFallbackCounter.Add(1))
	sum := sha256.Sum256([]byte(seed + ":" + strconv.FormatInt(now.Unix(), 10)))
	return hex.EncodeToString(sum[:16])
}

func idempotencyKeyFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(idempotencyKeyCtxKey{}).(string); ok {
		return v
	}
	return ""
}
