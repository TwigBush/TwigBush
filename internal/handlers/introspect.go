package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/TwigBush/gnap-go/internal/types"
)

type introspectReq struct {
	Token  string `json:"token,omitempty"`
	Action string `json:"action,omitempty"` // which action to authorize now
	Type   string `json:"type,omitempty"`   // GNAP access item type
	ID     string `json:"id,omitempty"`     // GNAP access item identifier
}

type IntrospectResp struct {
	Active bool     `json:"active"`
	Reason string   `json:"reason,omitempty"`
	Sub    string   `json:"sub,omitempty"`
	Iss    string   `json:"iss,omitempty"`
	Aud    []string `json:"aud,omitempty"`
	Exp    int64    `json:"exp,omitempty"`
	Iat    int64    `json:"iat,omitempty"`
	Key    any      `json:"key,omitempty"`
	Flags  []string `json:"flags,omitempty"`
	// optionally: Access []types.AccessItem if you return it
}

func Introspect(w http.ResponseWriter, r *http.Request) {
	// todo verify dpop
	// 1) Extract token
	var in introspectReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	tok := in.Token
	if tok == "" {
		if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "GNAP ") || strings.HasPrefix(hdr, "Bearer ") {
			parts := strings.SplitN(hdr, " ", 2)
			if len(parts) == 2 {
				tok = parts[1]
			}
		}
	}
	if tok == "" {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}

	// 2) Ask AS to introspect for this RS
	intro, err := fetchIntrospection(r.Context(), tok)
	if err != nil || !intro.Active || isExpired(intro) || !audOK(intro) {
		writeJSON(w, http.StatusOK, IntrospectResp{Active: false, Reason: "invalid_or_expired"})
		return
	}

	// 3) Build a policy check from GNAP access
	// subject: prefer end-user subject; fallback to client key id

	subject := intro.Sub
	if subject == "" {
		subject = subjectFromKey(intro.Key) // e.g., "client:<kid>"
	}
	//relation := in.Action
	//object := in.Type + ":" + in.ID

	//dec, err := authorizer.Check(r.Context(), authz.Request{
	//	Subject:  subject,
	//	Relation: relation,
	//	Object:   object,
	//	// Context: include constraints if you model CEL conditions in FGA
	//})
	//if err != nil {
	//	log.Printf("authz_error: %v", err)
	//	http.Error(w, "authz_error", http.StatusInternalServerError)
	//	return
	//}

	// 4) Respond per GNAP RS introspection

	resp := IntrospectResp{
		Active: intro.Active,
		Sub:    intro.Sub,
		Iss:    intro.Iss,
		Aud:    intro.Aud,
		Exp:    intro.Exp,
		Iat:    intro.Iat,
		Key:    intro.Key,
		Flags:  intro.Flags,
	}
	writeJSON(w, http.StatusOK, resp)
}

// helpers (stub)
// Replace this stub with a real call to your AS RS-introspection endpoint.
func fetchIntrospection(ctx context.Context, token string) (*types.IntrospectionResult, error) {
	// TODO: HTTP POST to AS introspection with proof, parse JSON.
	// Minimal mock for now
	return &types.IntrospectionResult{
		Active: true,
		Sub:    "user:alice",
		Iss:    "http://localhost:8089/grants",
		Aud:    []string{"rs:checkout"}, // set this RS audience
		Exp:    time.Now().Add(10 * time.Minute).Unix(),
	}, nil
}

func isExpired(i *types.IntrospectionResult) bool {
	return i == nil || (i.Exp != 0 && i.Exp <= time.Now().Unix())
}

// Verify token audience is appropriate for this RS.
// Replace "rs:checkout" with your RS audience id or check against config.
func audOK(i *types.IntrospectionResult) bool {
	if len(i.Aud) == 0 {
		return true // if you do not enforce aud yet
	}
	for _, a := range i.Aud {
		if a == "rs:checkout" {
			return true
		}
	}
	return false
}

func subjectFromKey(k any) string { return "client:unknown" }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
