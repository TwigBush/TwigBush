package main

import (
	"github.com/go-chi/chi/v5"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/handlers"
	"github.com/TwigBush/gnap-go/internal/playground"
)

func registerRoutes(r chi.Router) {
	store := gnap.NewMemoryStore(gnap.Config{
		GrantTTLSeconds: 120,
		TokenTTLSeconds: 300,
	})

	// Core endpoints
	grant := handlers.NewGrantHandler(store)
	cont := handlers.NewContinueHandler(store)
	device := handlers.NewDeviceHandler(store)

	r.Post("/grant", grant.ServeHTTP)
	r.Post("/continue/{grantId}", cont.ServeHTTP)
	r.Get("/device", device.Page)
	r.Post("/device/verify", device.VerifyForm)
	r.Post("/device/consent", device.ConsentForm)

	// Demo-only pieces
	sse := playground.NewSSEHub()
	r.Get("/events", sse.ServeHTTP)
	playground.MountUI(r)
	playground.MountDebug(r, store) // <-- add this so Approve/Deny buttons work
}
