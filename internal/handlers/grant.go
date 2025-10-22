package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TwigBush/gnap-go/internal/httpx"
	"github.com/TwigBush/gnap-go/internal/sign"
	"github.com/TwigBush/gnap-go/internal/types"
)

type GrantHandler struct {
	Store       types.GrantStore
	WaitSeconds int // how long the client should wait before polling /continue
}

func NewGrantHandler(store types.GrantStore) *GrantHandler {
	return &GrantHandler{Store: store, WaitSeconds: 5}
}

func (h *GrantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Grant called")
	// TODO: enforce proof  when implemented
	if err := sign.VerifyRequestProof(r); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid proof")
		return
	}

	var req types.GrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Client.Key.Proof == "" || req.Client.Key.JWK.Kty == "" || len(req.AccessToken) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "missing client.key or access")
		return
	}

	state, err := h.Store.CreateGrant(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	base := httpx.BaseURL(r)

	code := ""
	if state.UserCode != nil {
		code = *state.UserCode
	}

	resp := types.GrantResponse{
		Continue: types.Continue{
			AccessToken: state.ContinuationToken,
			URI:         base + "/continue/" + state.ID,
			Wait:        h.WaitSeconds,
		},

		Interact: types.InteractOut{
			Expires: state.ExpiresAt,
			UserCode: types.UserCode{
				Code: code,
				URI:  base + "/device",
			},
		},
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}
