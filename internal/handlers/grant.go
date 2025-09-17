package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/httpx"
	"github.com/TwigBush/gnap-go/internal/sign"
)

type GrantHandler struct {
	Store       gnap.Store
	WaitSeconds int // how long the client should wait before polling /continue
}

func NewGrantHandler(store gnap.Store) *GrantHandler {
	return &GrantHandler{Store: store, WaitSeconds: 5}
}

func (h *GrantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: enforce proof (httpsig/jwsd/dpop/mtls) when implemented
	if err := sign.VerifyRequestProof(r); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid proof")
		return
	}

	var req gnap.GrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Client.Key.Proof == "" || req.Client.Key.JWK.Kty == "" || len(req.Access) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "missing client.key or access")
		return
	}

	state, err := h.Store.CreateGrant(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	base := httpx.BaseURL(r)

	resp := gnap.GrantResponse{
		Continue: gnap.Continue{
			AccessToken: gnap.RandHex(16),         // 32 hex chars
			URI:         base + "/continue/" + state.ID,
			Wait:        h.WaitSeconds,
		},
		Interact: gnap.InteractOut{
			Expires: state.ExpiresAt,
			UserCode: gnap.UserCode{
				Code: gnap.RandUserCode(),
				URI:  base + "/device",
			},
		},
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

