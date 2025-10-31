package mw

import (
	"context"
	"net/http"
)

type rsContextKey struct{}

type RSIdentity struct {
	ID    string // your canonical RS id (use keyid or your own mapping)
	KeyID string
	Alg   string
}

func WithRSIdentity(r *http.Request, rs RSIdentity) *http.Request {
	ctx := context.WithValue(r.Context(), rsContextKey{}, rs)
	return r.WithContext(ctx)
}

func RSIdentityFromContext(r *http.Request) (RSIdentity, bool) {
	v := r.Context().Value(rsContextKey{})
	if v == nil {
		return RSIdentity{}, false
	}
	id, _ := v.(RSIdentity)
	return id, true
}
