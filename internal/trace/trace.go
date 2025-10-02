// internal/trace/trace.go
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey int

const key ctxKey = 1
const Header = "TRACE_ID"

func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key, id)
}

func From(ctx context.Context) string {
	if v := ctx.Value(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
