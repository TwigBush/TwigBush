package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	mw2 "github.com/TwigBush/gnap-go/internal/mw"
	"github.com/TwigBush/gnap-go/internal/types"
)

// ==== Request and response shapes ====

// RS → AS request body per RFC 9767
type asIntroReq struct {
	AccessToken    string             `json:"access_token"`     // required
	Proof          string             `json:"proof,omitempty"`  // recommended, registered method name
	ResourceServer json.RawMessage    `json:"resource_server"`  // required, string or object by ref
	Access         []types.AccessItem `json:"access,omitempty"` // optional, GNAP Section 8
	// Additional registry fields could appear; ignore unknowns
}

type asIntroResp struct {
	Active     bool               `json:"active"`
	Iss        string             `json:"iss,omitempty"`    // required when active
	Access     []types.AccessItem `json:"access,omitempty"` // required when active (can be empty array)
	Key        *gnap.BoundKey     `json:"key,omitempty"`    // only if token is bound
	Flags      []string           `json:"flags,omitempty"`
	Exp        int64              `json:"exp,omitempty"`
	Iat        int64              `json:"iat,omitempty"`
	Nbf        int64              `json:"nbf,omitempty"`
	Aud        []string           `json:"aud,omitempty"` // array form is allowed
	Sub        string             `json:"sub,omitempty"`
	InstanceID string             `json:"instance_id,omitempty"`
}

type RSRegistry interface {
	// Resolve RS identifier from the supplied resource_server value (string or object)
	Resolve(ctx context.Context, raw json.RawMessage) (string, error)

	// Get the RS verification key material to verify HTTP Message Signatures
	GetVerificationKey(ctx context.Context, rsID string, r *http.Request) (any, error)
}

// ==== Dependencies provided at construction ====

type IntrospectionHandler struct {
	Store      *gnap.TokenStoreContainer
	RSRegistry RSRegistry
	ASGrantURL string // iss to return, for example: https://as.example.com/tx
}

func NewIntrospectionHandler(store *gnap.TokenStoreContainer) *IntrospectionHandler {
	return &IntrospectionHandler{Store: store}
}

// ==== Public HTTP handler (AS endpoint) ====
func (h *IntrospectionHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	// 1) Verify RS authentication via HTTP Message Signatures
	rsIdent, ok := mw2.RSIdentityFromContext(r)
	if !ok || rsIdent.ID == "" {
		writeActiveFalse(w)
		return
	}
	rsID := rsIdent.ID

	// 2) Parse and validate request body
	var in asIntroReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.AccessToken == "" || len(in.ResourceServer) == 0 {
		writeActiveFalse(w)
		return
	}

	// Server reference in body must match the authenticated RS identity
	bodyRS, err := h.RSRegistry.Resolve(r.Context(), in.ResourceServer)
	if err != nil || bodyRS == "" || bodyRS != rsID {
		writeActiveFalse(w)
		return
	}

	// 3) Hash the token and lookup by hash only (opaque token pattern)
	sum := sha256.Sum256([]byte(in.AccessToken))
	hashB64 := base64.RawURLEncoding.EncodeToString(sum[:])

	tr, err := h.Store.GetByHash(r.Context(), hashB64)
	if err != nil || tr == nil {
		writeActiveFalse(w)
		return
	}

	now := time.Now().Unix()

	// 4) Evaluate "active" per RFC
	if tr.Iss == "" || tr.Iss != h.ASGrantURL {
		writeActiveFalse(w)
		return
	}
	if tr.Revoked {
		writeActiveFalse(w)
		return
	}
	if tr.Exp != 0 && tr.Exp <= now {
		writeActiveFalse(w)
		return
	}
	if tr.Nbf != 0 && now < tr.Nbf {
		writeActiveFalse(w)
		return
	}
	// Proof binding must match if token is bound
	if tr.BoundKey != nil {
		if in.Proof == "" || tr.BoundProof == "" || in.Proof != tr.BoundProof {
			writeActiveFalse(w)
			return
		}
	}
	// Audience must allow this RS
	if !audAllows(tr.Aud, rsID) {
		writeActiveFalse(w)
		return
	}
	// If caller supplied required access, token must be appropriate
	if len(in.Access) > 0 {
		ok, err := canSatisfyAccess(tr.Access, in.Access, rsID)
		if err != nil || !ok {
			writeActiveFalse(w)
			return
		}
	}

	// 5) Build filtered access for this RS (may be empty array)
	filtered := filterAccessForRS(tr.Access, rsID)

	// 6) Respond active=true with required fields
	resp := asIntroResp{
		Active:     true,
		Iss:        tr.Iss,
		Access:     filtered, // required, may be empty
		Flags:      nil,      // todo (joshfischer) do we want to include tracking flags
		Exp:        tr.Exp,
		Iat:        tr.Iat,
		Nbf:        tr.Nbf,
		Aud:        tr.Aud, // array form
		Sub:        tr.Sub,
		InstanceID: tr.InstanceID,
	}
	if tr.BoundKey != nil {
		resp.Key = &gnap.BoundKey{
			Proof: tr.BoundKey.Proof,
			JWK:   tr.BoundKey.JWK,
			Ref:   tr.BoundKey.Ref,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func audAllows(aud []string, rsID string) bool {
	if len(aud) == 0 {
		return true
	}
	for _, a := range aud {
		if a == rsID {
			return true
		}
	}
	return false
}

// canSatisfyAccess returns true if tokenAccess is appropriate for requestedAccess at this RS.
// You decide subset and constraint semantics. On parse error, return (false, err)
// If you cannot process the request's access content, you must not mark active.
func canSatisfyAccess(tokenAccess, requested []types.AccessItem, rsID string) (bool, error) {
	// Minimal conservative behavior: require that each requested type/id is present in token

	index := map[string]struct{}{}
	for _, a := range tokenAccess {
		key := a.Type + "|" + a.ID
		index[key] = struct{}{}
	}
	for _, r := range requested {
		key := r.Type + "|" + r.ID
		if _, ok := index[key]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func filterAccessForRS(all []types.AccessItem, rsID string) []types.AccessItem {
	if len(all) == 0 {
		return []types.AccessItem{}
	}
	out := make([]types.AccessItem, 0, len(all))
	for _, a := range all {
		// If you tag access with audiences, filter here. Otherwise return as is.
		out = append(out, a)
	}
	return out
}

func writeActiveFalse(w http.ResponseWriter) {
	// Spec requires 200 with only {"active": false} and no other fields
	_ = json.NewEncoder(w).Encode(map[string]bool{"active": false})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
