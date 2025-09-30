package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/TwigBush/gnap-go/internal/di"
	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/handlers"
	"github.com/TwigBush/gnap-go/internal/mw"
	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func defaultDataDir() string {
	// Respect explicit override first
	if v := os.Getenv("TWIGBUSH_DATA_DIR"); v != "" {
		return v
	}
	// Resolve HOME cross-platform
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// Fallback to current working dir if HOME is unknown
		return filepath.Join(".", ".twigbush", "data")
	}
	return filepath.Join(home, ".twigbush", "data")
}

func registerRoutes(r chi.Router) {
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3001", "*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	cfg := types.Config{GrantTTLSeconds: 120}
	authz := di.ProvideAuthorizer()
	var store types.Store
	switch os.Getenv("TWIGBUSH_STORE") {
	case "fs":
		dir := defaultDataDir()
		fsStore, err := gnap.NewFileStore(dir, cfg)
		if err != nil {
			panic(err)
		}
		store = fsStore
	default:
		dir := defaultDataDir()
		fsStore, err := gnap.NewFileStore(dir, cfg)
		if err != nil {
			panic(err)
		}
		store = fsStore
	}

	r.Use(mw.Trace()) // from earlier
	r.Use(mw.Logger(mw.LogOpts{
		PollSkipEvery: 4, // log every 4th /continue call
	}))

	//r.Use(mw.RequestID)
	//r.Use(mw.Logger(mw.LogOptions{
	//	// default no body sampling
	//	SampleBodies: os.Getenv("LOG_SAMPLE_BODIES") == "1",
	//	MaxBodyBytes: 2048,
	//}))

	grant := handlers.NewGrantHandler(store)
	cont := handlers.NewContinueHandler(store)
	device := handlers.NewDeviceHandler(store)
	// health and version
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/version", handlers.Version)

	// GNAP endpoints
	r.Post("/grants", grant.ServeHTTP)

	r.Post("/continue/{grantId}", cont.ServeHTTP)

	r.Post("/introspect", handlers.Introspect(authz))
	r.Get("/.well-known/jwks.json", handlers.JWKS)
	r.Post("/device/verify/json", device.VerifyJSON)
	r.Post("/device/verify", device.VerifyForm)
	r.Get("/device", device.Page)
	r.Post("/device/consent", device.ConsentForm)
}
