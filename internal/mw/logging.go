package mw

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/TwigBush/gnap-go/internal/httpx"
	"github.com/TwigBush/gnap-go/internal/trace"
)

type LogOpts struct {
	SampleBodies  bool
	MaxBodyBytes  int
	PollSkipEvery int // e.g., 4 logs only every 4th call on polling endpoints
	SkipPaths     []string
	RedactHeaders []string
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions
}

func isNoisyPath(p string) bool {
	if p == "/healthz" || p == "/version" {
		return true
	}
	// add static prefixes as needed
	return false
}

func isPollingPath(p string) bool {
	return strings.HasPrefix(p, "/continue/")
}

var pollCounter uint64

func Logger(opts LogOpts) func(http.Handler) http.Handler {
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = 2048
	}
	if opts.PollSkipEvery <= 0 {
		opts.PollSkipEvery = 1
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if isPreflight(r) || isNoisyPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// throttle super-chatty polling
			if isPollingPath(r.URL.Path) && opts.PollSkipEvery > 1 {
				pollCounter++
				if pollCounter%uint64(opts.PollSkipEvery) != 0 {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()
			rec := httpx.NewRecorder(w)
			next.ServeHTTP(rec, r)
			dur := time.Since(start)

			// one-liner summary
			slog.Info("req",
				"trace", trace.From(r.Context()),
				"m", r.Method,
				"path", r.URL.Path,
				"status", rec.Status,
				"ms", dur.Milliseconds(),
				"bytes", rec.Bytes,
			)

			// on error, add a compact JSON block with headers/body sample
			if rec.Status >= 400 {
				h := map[string]string{}
				for k, vv := range r.Header {
					if len(vv) == 0 {
						continue
					}
					vl := vv[0]
					if strings.EqualFold(k, "Authorization") || strings.HasPrefix(strings.ToLower(k), "x-api-key") {
						vl = "***redacted***"
					}
					h[k] = vl
				}
				slog.Error("req_detail",
					"trace", trace.From(r.Context()),
					"m", r.Method, "path", r.URL.Path,
					"status", rec.Status, "ms", dur.Milliseconds(),
					"headers", h,
				)
			}
		})
	}
}

//package mw
//
//import (
//	"bytes"
//	"context"
//	"io"
//	"log/slog"
//	"net/http"
//	"os"
//	"strings"
//	"time"
//
//	"github.com/TwigBush/gnap-go/internal/httpx"
//)
//
//type ctxKey int
//
//const reqIDKey ctxKey = 1
//
//// Very small request id generator
//func requestID() string {
//	return time.Now().UTC().Format("20060102T150405.000000000") // good enough
//}
//
//func RequestID(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		id := r.Header.Get("X-Request-ID")
//		if id == "" {
//			id = requestID()
//		}
//		ctx := context.WithValue(r.Context(), reqIDKey, id)
//		w.Header().Set("X-Request-ID", id)
//		next.ServeHTTP(w, r.WithContext(ctx))
//	})
//}
//
//func reqIDFrom(ctx context.Context) string {
//	v := ctx.Value(reqIDKey)
//	if s, ok := v.(string); ok {
//		return s
//	}
//	return ""
//}
//
//// Redact common secrets in headers
//func redactHeader(k, v string) string {
//	k = strings.ToLower(k)
//	if k == "authorization" || strings.HasPrefix(k, "x-api-key") {
//		return "***redacted***"
//	}
//	return v
//}
//
//type LogOptions struct {
//	SampleBodies bool // capture small bodies for debug
//	MaxBodyBytes int  // default 2048
//}
//
//func Logger(opts LogOptions) func(http.Handler) http.Handler {
//	if opts.MaxBodyBytes <= 0 {
//		opts.MaxBodyBytes = 2048
//	}
//	return func(next http.Handler) http.Handler {
//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			start := time.Now()
//			rec := httpx.NewRecorder(w)
//
//			// Build base fields
//			attrs := []any{
//				"req.id", reqIDFrom(r.Context()),
//				"http.method", r.Method,
//				"url.path", r.URL.Path,
//				"remote.addr", r.RemoteAddr,
//				"user.agent", r.UserAgent(),
//			}
//
//			// Optional request body sampling (small, non-binary)
//			var reqBodySample string
//			if opts.SampleBodies && r.Body != nil && r.ContentLength <= int64(opts.MaxBodyBytes) {
//				var buf bytes.Buffer
//				tee := io.TeeReader(r.Body, &buf)
//				body, _ := io.ReadAll(io.LimitReader(tee, int64(opts.MaxBodyBytes)))
//				r.Body = io.NopCloser(&buf)
//				if len(body) > 0 {
//					reqBodySample = string(body)
//				}
//			}
//
//			// Log request start
//			l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo}))
//			hdrs := map[string]string{}
//			for k, vv := range r.Header {
//				if len(vv) > 0 {
//					hdrs[k] = redactHeader(k, vv[0])
//				}
//			}
//			l.Info("http_request", append(attrs, "req.headers", hdrs, "req.body_sample", reqBodySample)...)
//
//			// Serve
//			next.ServeHTTP(rec, r)
//
//			// Log response
//			dur := time.Since(start)
//
//			attrs = append(attrs,
//				"http.status", rec.Status,
//				"http.duration_ms", dur.Milliseconds(),
//				"http.bytes", rec.Bytes,
//			)
//
//			l.Info("http_response", attrs...)
//		})
//	}
//}
