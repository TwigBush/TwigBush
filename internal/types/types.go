package types

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

type Interact struct {
	Start []string `json:"start,omitempty"`
}

type AccessToken struct {
	Label  string       `json:"label,omitempty"`
	Access []AccessItem `json:"access"`
	Flags  []string     `json:"flags,omitempty"`
}

// AccessTokenRequest can be either a single AccessToken or an array of AccessTokens
type AccessTokenRequest []AccessToken

// UnmarshalJSON Custom unmarshaling to handle both single object and array formats
// todo - revist this for single vs multiple access token issuance
func (a *AccessTokenRequest) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as array first
	var tokens []AccessToken
	if err := json.Unmarshal(data, &tokens); err == nil {
		*a = tokens
		return nil
	}

	// If that fails, try as single object
	var token AccessToken
	if err := json.Unmarshal(data, &token); err != nil {
		return err
	}

	*a = []AccessToken{token}
	return nil
}

type AccessItem struct {
	ID          string          `json:"id,omitempty"`
	Type        string          `json:"type"`
	Actions     []string        `json:"actions,omitempty"`
	Locations   []string        `json:"locations,omitempty"`
	Datatypes   []string        `json:"datatypes,omitempty"`
	Identifier  string          `json:"identifier,omitempty"`
	Constraints json.RawMessage `json:"constraints,omitempty"`
}

type GrantRequest struct {
	AccessToken AccessTokenRequest `json:"access_token"`
	Client      Client             `json:"client"`
	Interact    *Interact          `json:"interact,omitempty"`
	TokenFormat string             `json:"token_format,omitempty"`
}

type GrantState struct {
	ID                    string             `json:"id"`
	Status                GrantStatus        `json:"status"`
	Client                Client             `json:"client"`
	RequestedAccess       AccessTokenRequest `json:"requested_access"`
	ApprovedAccess        AccessTokenRequest `json:"approved_access,omitempty"` // set when approved
	Subject               *string            `json:"subject,omitempty"`         // e.g. sub id
	ContinuationToken     string             `json:"continuation_token"`
	TokenFormat           string             `json:"token_format"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
	ExpiresAt             time.Time          `json:"expires_at"`
	Locations             json.RawMessage    `json:"locations,omitempty"`
	UserCode              *string            `json:"user_code,omitempty"`
	ApprovedAccessGranted []GrantedAccess    `json:"approved_access_granted,omitempty"`
	CodeVerified          bool               `json:"code_verified"`
}

type Config struct {
	GrantTTLSeconds int64
	TokenTTLSeconds int64
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

type GrantStore interface {
	CreateGrant(ctx context.Context, req GrantRequest) (*GrantState, error)
	GetGrant(ctx context.Context, id string) (*GrantState, bool)
	FindGrantByUserCodePending(ctx context.Context, code string) (*GrantState, bool)

	ApproveGrant(ctx context.Context, id string, approved AccessTokenRequest, subject string) (*GrantState, error)
	DenyGrant(ctx context.Context, id string) (*GrantState, error)

	MarkCodeVerified(ctx context.Context, id string) error
}

type KeyPair struct {
	PrivateKey JWK
	PublicKey  JWK
}

type Proof string

type ProofMethod struct {
	HTTPSig Proof
	MTLs    Proof
}

type Configuration struct {
	ClientID      string
	ClientName    string
	ClientVersion string
	ClientURI     string
	KeyPair       KeyPair
	ProofMethod   ProofMethod
	AsURL         string
}

type Continue struct {
	AccessToken string `json:"access_token"`
	URI         string `json:"uri"`
	Wait        int    `json:"wait"` // seconds to poll before calling /continue
}

type UserCode struct {
	Code string `json:"code"`
	URI  string `json:"uri"`
}

type InteractOut struct {
	Expires  time.Time `json:"expires"`
	UserCode UserCode  `json:"user_code"`
}

type GrantResponse struct {
	Continue Continue    `json:"continue"`
	Interact InteractOut `json:"interact"`
}
