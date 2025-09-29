package main

import (
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/handlers"
	"github.com/TwigBush/gnap-go/internal/playground"
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
	cfg := gnap.Config{GrantTTLSeconds: 120}
	var store gnap.Store
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

	// Core endpoints
	grant := handlers.NewGrantHandler(store)
	cont := handlers.NewContinueHandler(store)
	device := handlers.NewDeviceHandler(store)

	r.Post("/grant", grant.ServeHTTP)
	r.Post("/continue/{grantId}", cont.ServeHTTP)
	r.Get("/device", device.Page)
	r.Post("/device/verify", device.VerifyJSON)

	r.Post("/device/consent", device.ConsentForm)

	// Demo-only pieces
	sse := playground.NewSSEHub()
	r.Get("/events", sse.ServeHTTP)
	playground.MountUI(r)
	playground.MountDebug(r, store) // so Approve/Deny buttons work
}
