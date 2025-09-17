package gnap

import (
	"context"
	"encoding/json"
	"time"
)

type GrantStatus string

const (
	GrantStatusPending GrantStatus = "pending"
)

type JWK struct {
	Kty string `json:"kty"`           // e.g. "EC"
	Crv string `json:"crv,omitempty"` // e.g. "P-256"
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	// Add "kid","alg" later if needed
}

type ClientKey struct {
	Proof string `json:"proof"` // e.g. "httpsig","jwsd","mtls","dpop"
	JWK   JWK    `json:"jwk"`
}

type Client struct {
	Key ClientKey `json:"key"`
}

type AccessConstraint map[string]string

type AccessItem struct {
	Type        string           `json:"type"`                  // e.g. "payment"
	ResourceID  string           `json:"resource_id,omitempty"` // e.g. "sku:GPU-HOURS-100"
	Actions     []string         `json:"actions,omitempty"`     // e.g. ["purchase"]
	Constraints AccessConstraint `json:"constraints,omitempty"` // amount,currency,merchant_id
	Locations   []string         `json:"locations,omitempty"`   // audiences / RS urls
}

type Interact struct {
	Start []string `json:"start,omitempty"` // e.g. ["user_code"], ["redirect","user_code"]
}

type GrantRequest struct {
	Client      Client       `json:"client"`
	Access      []AccessItem `json:"access"`
	Interact    *Interact    `json:"interact,omitempty"`
	TokenFormat string       `json:"token_format,omitempty"` // e.g. "jwt", "opaque"
}

type GrantState struct {
	ID                string       `json:"id"`
	Status            GrantStatus  `json:"status"`
	Client            Client       `json:"client"`
	RequestedAccess   []AccessItem `json:"requested_access"`
	ContinuationToken string       `json:"continuation_token"`
	TokenFormat       string       `json:"token_format"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	ExpiresAt         time.Time    `json:"expires_at"`
	InteractionNonce  *string      `json:"interaction_nonce,omitempty"`
	// Optional echo of locations to help RS routing
	Locations json.RawMessage `json:"locations,omitempty"`
}

type Config struct {
	GrantTTLSeconds int64
}

type Store interface {
	CreateGrant(ctx context.Context, req GrantRequest) (*GrantState, error)
}
