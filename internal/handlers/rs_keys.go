package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type RSKeysHandler struct {
	store *gnap.RSKeyStore
}

func NewRSKeysHandler(store *gnap.RSKeyStore) *RSKeysHandler {
	return &RSKeysHandler{store: store}
}

// POST /admin/tenants/{tenant}/rs/keys
func (h *RSKeysHandler) RegisterKey(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	if tenant == "" {
		tenant = "default" // fallback tenant
	}

	var in struct {
		JWK       json.RawMessage `json:"jwk"`
		KID       string          `json:"kid"`
		Alg       string          `json:"alg"`
		DisplayRS string          `json:"display_rs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	pub, err := jwk.ParseKey(in.JWK)
	if err != nil {
		http.Error(w, "invalid JWK", http.StatusBadRequest)
		return
	}

	// todo: check if key is allowed to be saved, e.g. private key vs public key

	// Admin endpoint: TOFU disabled (acceptTOFU = false for explicit registration)
	rec, err := h.store.UpsertRSKey(r.Context(), tenant, pub, in.KID, in.Alg, in.DisplayRS, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"thumb256":   rec.Thumb256,
		"kid":        rec.KID,
		"display_rs": rec.DisplayRS,
	})
}

// GET /admin/tenants/{tenant}/rs/keys
func (h *RSKeysHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	if tenant == "" {
		tenant = "default"
	}

	keys := h.store.ListRSKeys(r.Context(), tenant)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"keys": keys,
	})
}

// GET /admin/tenants/{tenant}/rs/keys/{thumb256}
func (h *RSKeysHandler) GetKey(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	thumb256 := chi.URLParam(r, "thumb256")

	if tenant == "" {
		tenant = "default"
	}

	rec, ok := h.store.GetRSKey(r.Context(), tenant, thumb256)
	if !ok {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

// DELETE /admin/tenants/{tenant}/rs/keys/{thumb256}
func (h *RSKeysHandler) DeactivateKey(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	thumb256 := chi.URLParam(r, "thumb256")

	if tenant == "" {
		tenant = "default"
	}

	if err := h.store.DeactivateRSKey(r.Context(), tenant, thumb256); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
