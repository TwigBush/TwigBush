package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/handlers"
	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/go-chi/chi/v5"
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
	cfg := types.Config{GrantTTLSeconds: 120}
	
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
		store = gnap.NewMemoryStore(cfg)
	}

	grant := handlers.NewGrantHandler(store)
	cont := handlers.NewContinueHandler(store)
	device := handlers.NewDeviceHandler(store)
	// health and version
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/version", handlers.Version)

	// GNAP endpoints
	r.Post("/grants", grant.ServeHTTP)
	r.Post("/continue/{grantId}", cont.ServeHTTP)
	r.Post("/introspect", handlers.Introspect)
	r.Get("/.well-known/jwks.json", handlers.JWKS)
	r.Post("/device/verify/json", device.VerifyJSON)
	r.Post("/device/verify", device.VerifyForm)
	r.Get("/device", device.Page)
	r.Post("/device/consent", device.ConsentForm)
}
