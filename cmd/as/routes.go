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
	// health and version
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/version", handlers.Version)

	// GNAP endpoints
	r.Post("/grant", grant.ServeHTTP)      // parse, verify proof, consult policy, issue or interact
	r.Post("/continue", handlers.Continue) // finish interactions, mint token
	r.Post("/introspect", handlers.Introspect)
	r.Get("/.well-known/jwks.json", handlers.JWKS)
}
