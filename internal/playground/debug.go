package playground

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/TwigBush/gnap-go/internal/gnap"
)

func MountDebug(r chi.Router, store gnap.Store) {
	r.Post("/debug/approve/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		g, ok := store.GetGrant(r.Context(), id)
		if !ok || g == nil {
			http.NotFound(w, r)
			return
		}
		if _, err := store.ApproveGrant(r.Context(), id, g.RequestedAccess, "user:debug"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	r.Post("/debug/deny/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, ok := store.GetGrant(r.Context(), id); !ok {
			http.NotFound(w, r)
			return
		}
		if _, err := store.DenyGrant(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
