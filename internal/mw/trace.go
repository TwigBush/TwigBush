// internal/mw/trace.go
package mw

import (
	"net/http"

	"github.com/TwigBush/gnap-go/internal/trace"
)

func Trace() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(trace.Header)
			if id == "" {
				id = trace.NewID()
			}
			ctx := trace.With(r.Context(), id)

			// echo back on response
			w.Header().Set(trace.Header, id)
			w.Header().Set("X-Request-ID", id)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
