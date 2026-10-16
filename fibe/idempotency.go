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

// WithIdempotencyKey adds an API-cached, 24-hour replay key to the next request.
// Use it when a mutating request may need a safe retry after a network failure.
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
