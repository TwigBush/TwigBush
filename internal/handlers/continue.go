package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/httpx"
	"github.com/TwigBush/gnap-go/internal/token"
)

type ContinueHandler struct {
	Store gnap.Store
	// optional: configure default wait seconds
	WaitSeconds int
}

func NewContinueHandler(store gnap.Store) *ContinueHandler {
	return &ContinueHandler{Store: store, WaitSeconds: 5}
}

func (h *ContinueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	grantID := chi.URLParam(r, "grantId")

	authz := r.Header.Get("Authorization")
	contToken, ok := httpx.ExtractGNAPToken(authz)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "Missing continuation token")
		return
	}

	grant, found := h.Store.GetGrant(r.Context(), grantID)
	if !found {
		httpx.WriteError(w, http.StatusNotFound, "Grant not found")
		return
	}

	if contToken != grant.ContinuationToken {
		httpx.WriteError(w, http.StatusUnauthorized, "Invalid continuation token")
		return
	}

	switch grant.Status {
	case gnap.GrantStatusPending:
		// mirror your Java: return a "continue" object
		resp := map[string]any{
			"continue": map[string]any{
				"access_token": contToken,
				"uri":          baseURL(r) + "/continue/" + grantID,
				"wait":         h.WaitSeconds,
			},
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
		return

	case gnap.GrantStatusApproved:
		// issue final access token
		issuer := baseURL(r) // simple; swap for configured issuer if you have one
		tok, err := token.IssueToken(grant, token.IssueConfig{
			Issuer:          issuer,
			TokenTTLSeconds: grantTokenTTL(r.Context(), h.Store), // simple helper, or use constant
			Audience:        "mcp-resource-servers",
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

	case gnap.GrantStatusDenied:
		httpx.WriteError(w, http.StatusForbidden, "Grant denied by user")
		return

	case gnap.GrantStatusExpired:
		httpx.WriteError(w, http.StatusBadRequest, "Grant expired")
		return
	}

	httpx.WriteError(w, http.StatusBadRequest, "Unknown grant status")
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// If you wired Config into the store, you can read it from there.
// For now, just return a constant or 300s to mirror Java.
func grantTokenTTL(_ context.Context, _ gnap.Store) int64 { return 300 }
