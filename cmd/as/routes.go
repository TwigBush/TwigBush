package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/TwigBush/gnap-go/internal/handlers"
)

func registerRoutes(r chi.Router) {
	// health and version
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/version", handlers.Version)

	// GNAP endpoints
	r.Post("/grant", handlers.Grant)       // parse, verify proof, consult policy, issue or interact
	r.Post("/continue", handlers.Continue) // finish interactions, mint token
	r.Post("/introspect", handlers.Introspect)
	r.Get("/.well-known/jwks.json", handlers.JWKS)
}
