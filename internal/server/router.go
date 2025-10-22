package server

import (
	"crypto"
	"encoding/json"
	"net/http"
	"os"

	"github.com/TwigBush/gnap-go/internal/gnap"
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
	GrantStore types.GrantStore
	RSKeyStore *gnap.RSKeyStore
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

	grant := handlers.NewGrantHandler(d.GrantStore)
	cont := handlers.NewContinueHandler(d.GrantStore)
	device := handlers.NewDeviceHandler(d.GrantStore)

	r.Get("/healthz", healthCheckHandler)
	r.Get("/version", handlers.VersionHandler)

	r.Route("/", func(rsr chi.Router) {
		rsr.Use(mw2.VerifyRSProof(
			mw2.WithRSKeyResolver(func(r *http.Request, params map[string]string) (crypto.PublicKey, error) {
				// Resolve by keyid, or by RS identity in mTLS, or by tenant
				// todo: finish
				kid := params["keyid"]
				return d.RSKeyStore.LookupRSPublicKeyById(kid)

			}),
			mw2.WithRSRequiredComponents([]string{"@method", "@target-uri"}), // add "content-digest" if you require it
			mw2.WithRSAllowedAlgs("ecdsa-p256-sha256", "ecdsa-p384-sha384", "ed25519"),
		))
		rsr.Post("/grants", grant.ServeHTTP)

		rsr.Post("/continue/{grantId}", cont.ServeHTTP)
		rsr.Post("/introspect", handlers.Introspect)
		//rsr.Post("/register", rs.HandleRegisterResourceSet)
		//rsr.Post("/token", rs.HandleTokenChaining)
	})

	r.Get("/.well-known/jwks.json", handlers.JWKS)

	r.Post("/device/verify/json", device.VerifyJSON)
	r.Post("/device/verify", device.VerifyForm)
	r.Get("/device", device.Page)
	r.Post("/device/consent", device.ConsentForm)

	if d.RSKeyStore != nil {
		rsKeys := handlers.NewRSKeysHandler(d.RSKeyStore)
		r.Route("/admin/tenants/{tenant}/rs/keys", func(r chi.Router) {
			r.Post("/", rsKeys.RegisterKey)
			r.Get("/", rsKeys.ListKeys)
			r.Get("/{thumb256}", rsKeys.GetKey)
			r.Delete("/{thumb256}", rsKeys.DeactivateKey)
		})

	}

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
	playground.MountDebug(r, d.GrantStore)
	return r
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"version": version.Version,
	})
}
