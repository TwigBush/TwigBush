package gnap

import (
	"context"
	"encoding/json"
	"time"
)

type GrantStatus string

const (
	GrantStatusPending  GrantStatus = "pending"
	GrantStatusApproved GrantStatus = "approved"
	GrantStatusDenied   GrantStatus = "denied"
	GrantStatusExpired  GrantStatus = "expired"
)

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

type ClientKey struct {
	Proof string `json:"proof"`
	JWK   JWK    `json:"jwk"`
}
type Client struct {
	Key ClientKey `json:"key"`
}

type GrantedAccess struct {
	ResourceID     string   `json:"resource_id,omitempty"`
	Type           string   `json:"type"`
	Rights         []string `json:"rights,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	ResourceServer string   `json:"resource_server,omitempty"`
}

type AccessConstraint map[string]string
type AccessItem struct {
	Type        string           `json:"type"`
	ResourceID  string           `json:"resource_id,omitempty"`
	Actions     []string         `json:"actions,omitempty"`
	Constraints AccessConstraint `json:"constraints,omitempty"`
	Locations   []string         `json:"locations,omitempty"`
}

type Interact struct {
	Start []string `json:"start,omitempty"`
}

type GrantRequest struct {
	Client      Client       `json:"client"`
	Access      []AccessItem `json:"access"`
	Interact    *Interact    `json:"interact,omitempty"`
	TokenFormat string       `json:"token_format,omitempty"` // "jwt"|"opaque"
}

type GrantState struct {
	ID                    string          `json:"id"`
	Status                GrantStatus     `json:"status"`
	Client                Client          `json:"client"`
	RequestedAccess       []AccessItem    `json:"requested_access"`
	ApprovedAccess        []AccessItem    `json:"approved_access,omitempty"` // set when approved
	Subject               *string         `json:"subject,omitempty"`         // e.g. sub id
	ContinuationToken     string          `json:"continuation_token"`
	TokenFormat           string          `json:"token_format"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	ExpiresAt             time.Time       `json:"expires_at"`
	Locations             json.RawMessage `json:"locations,omitempty"`
	UserCode              *string         `json:"user_code,omitempty"`
	ApprovedAccessGranted []GrantedAccess `json:"approved_access_granted,omitempty"`
	CodeVerified          bool            `json:"code_verified"`
}

type Config struct {
	GrantTTLSeconds int64
	TokenTTLSeconds int64
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

type Store interface {
	CreateGrant(ctx context.Context, req GrantRequest) (*GrantState, error)
	GetGrant(ctx context.Context, id string) (*GrantState, bool)
	FindGrantByUserCodePending(ctx context.Context, code string) (*GrantState, bool)

	ApproveGrant(ctx context.Context, id string, approved []AccessItem, subject string) (*GrantState, error)
	DenyGrant(ctx context.Context, id string) (*GrantState, error)

	MarkCodeVerified(ctx context.Context, id string) error
}
