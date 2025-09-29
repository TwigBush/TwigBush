package handlers

import (
	"context"
	"net/http"

	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/go-chi/chi/v5"

	"github.com/TwigBush/gnap-go/internal/httpx"
	"github.com/TwigBush/gnap-go/internal/token"
)

type ContinueHandler struct {
	Store       types.Store
	WaitSeconds int // how long the client should wait before polling /continue
}

func NewContinueHandler(store types.Store) *ContinueHandler {
	return &ContinueHandler{Store: store, WaitSeconds: 5}
}

func (h *ContinueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// GNAP continuation MUST be a POST with Authorization: GNAP <token>
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Responses should not be cached
	w.Header().Set("Cache-Control", "no-store")

	grantID := chi.URLParam(r, "grantId")

	authz := r.Header.Get("Authorization")
	contToken, ok := httpx.ExtractGNAPToken(authz)
	if !ok || contToken == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing continuation token")
		return
	}

	grant, found := h.Store.GetGrant(r.Context(), grantID)
	if !found || grant == nil {
		httpx.WriteError(w, http.StatusNotFound, "grant not found")
		return
	}

	// Simple token equality check (opaque token)
	if contToken != grant.ContinuationToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid continuation token")
		return
	}

	switch grant.Status {
	case types.GrantStatusPending:
		// Still pending: instruct client to poll again
		resp := map[string]any{
			"continue": map[string]any{
				"access_token": contToken,
				"uri":          baseURL(r) + "/continue/" + grantID,
				"wait":         h.WaitSeconds,
			},
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
		return

	case types.GrantStatusApproved:
		// Issue the final access token. Bind to client key if your IssueToken supports it.
		issuer := baseURL(r)
		tok, err := token.IssueToken(grant, token.IssueConfig{
			Issuer:          issuer,
			Audience:        "mcp-resource-servers",
			TokenTTLSeconds: grantTokenTTL(r.Context(), h.Store),
			//BindJWK:         grant.Client.Key.JWK, // enable cnf/jkt in token layer
		})
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		resp := map[string]any{
			"access_token": tok,
			"instance_id":  grant.ID,
		}
		if grant.Subject != nil {
			resp["subject"] = map[string]any{"sub_ids": []string{*grant.Subject}}
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
		return

	case types.GrantStatusDenied:
		httpx.WriteError(w, http.StatusForbidden, "grant denied by user")
		return

	case types.GrantStatusExpired:
		httpx.WriteError(w, http.StatusBadRequest, "grant expired")
		return

	default:
		httpx.WriteError(w, http.StatusBadRequest, "unknown grant status")
		return
	}
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// If you wired Config into the store, you can read it from there.
// For now, return a sane default.
func grantTokenTTL(_ context.Context, _ types.Store) int64 { return 300 }
