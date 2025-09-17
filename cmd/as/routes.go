package main

import (
	"net/http"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/handlers"
	"github.com/go-chi/chi/v5"
)

func registerRoutes(r chi.Router) {
	store := gnap.NewMemoryStore(gnap.Config{GrantTTLSeconds: 120})
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
