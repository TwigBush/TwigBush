package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/TwigBush/gnap-go/internal/handlers"
	mw2 "github.com/TwigBush/gnap-go/internal/mw"
	"github.com/TwigBush/gnap-go/internal/playground"
	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/TwigBush/gnap-go/internal/version"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Options struct {
	EnableCORS bool
	DevNoStore bool
}

type Deps struct {
	Store types.Store
}

func BuildASRouter(d Deps, opts Options, mw ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	if os.Getenv("TWIGBUSH_ENV") == "local" || os.Getenv("TWIGBUSH_ENV") == "dev" {
		// temporary fix to prevent caching
		r.Use(mw2.NoStore)
	}

	// baseline
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8088", "*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	for _, m := range mw {
		r.Use(m)
	}

	// tracing + logger
	r.Use(mw2.Trace())
	r.Use(mw2.Logger(mw2.LogOpts{
		PollSkipEvery: 4, // sample /continue
		SkipPaths:     []string{"/healthz", "/version"},
		RedactHeaders: []string{"Authorization"},
	}))

	grant := handlers.NewGrantHandler(d.Store)
	cont := handlers.NewContinueHandler(d.Store)
	device := handlers.NewDeviceHandler(d.Store)

	r.Get("/healthz", healthCheckHandler)
	r.Get("/version", handlers.VersionHandler)

	r.Post("/grants", grant.ServeHTTP)
	r.Options("/grants", GrantDiscoveryHandler)
	r.Post("/continue/{grantId}", cont.ServeHTTP)
	r.Post("/introspect", handlers.Introspect)
	r.Get("/.well-known/jwks.json", handlers.JWKS)

	r.Post("/device/verify/json", device.VerifyJSON)
	r.Post("/device/verify", device.VerifyForm)
	r.Get("/device", device.Page)
	r.Post("/device/consent", device.ConsentForm)

	return r
}

func BuildPlaygroundRouter(d Deps, opts Options) http.Handler {
	r := chi.NewRouter()
	if opts.DevNoStore {
		r.Use(mw2.NoStore) // stops UI caching in dev
	}

	// Log everything except the SSE stream itself
	r.Use(mw2.Logger(mw2.LogOpts{
		SkipPaths: []string{"/events"},
	}))
	sse := playground.NewSSEHub()
	r.Get("/events", sse.ServeHTTP)
	playground.MountUI(r)
	playground.MountDebug(r, d.Store)
	return r
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"version": version.Version,
	})
}
