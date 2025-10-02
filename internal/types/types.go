package types

import (
	"context"
	"encoding/json"
	"fmt"
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

//type AccessItem struct {
//	Type        string          `json:"type"`
//	ResourceID  string          `json:"resource_id,omitempty"`
//	Actions     []string        `json:"actions,omitempty"`
//	Constraints json.RawMessage `json:"constraints,omitempty"`
//	Locations   []string        `json:"locations,omitempty"`
//}

type Interact struct {
	Start []string `json:"start,omitempty"`
}

//type GrantRequest struct {
//	Client      Client       `json:"client"`
//	Access      []AccessItem `json:"access"`
//	Interact    *Interact    `json:"interact,omitempty"`
//	TokenFormat string       `json:"token_format,omitempty"` // "jwt"|"opaque"
//}

type AccessToken struct {
	Label  string       `json:"label,omitempty"`
	Access []AccessItem `json:"access"`
	Flags  []string     `json:"flags,omitempty"`
}

// AccessTokenRequest can be either a single AccessToken or an array of AccessTokens
type AccessTokenRequest []AccessToken

// Custom unmarshaling to handle both single object and array formats
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
	Type        string          `json:"type"`
	Actions     []string        `json:"actions,omitempty"`
	Locations   []string        `json:"locations,omitempty"`
	Datatypes   []string        `json:"datatypes,omitempty"`
	Identifier  string          `json:"identifier,omitempty"`
	Constraints json.RawMessage `json:"constraints,omitempty"`
}

type GrantRequest struct {
	AccessToken AccessTokenRequest `json:"access_token"`
	Access      []AccessItem       `json:"access"` // todo: addresss this hierarch before merge
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

type LineItem struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type Schedule struct {
	Kind string `json:"kind"`
}

type StylesByMeal struct {
	Breakfast []string `json:"breakfast"`
	Lunch     []string `json:"lunch"`
	Dinner    []string `json:"dinner"`
}

type FinanceConstraints struct {
	MaxAmount    float64  `json:"max_amount,omitempty"`
	Currency     string   `json:"currency,omitempty"`
	AccountTypes []string `json:"account_types,omitempty"`
}

type ShoppingConstraints struct {
	Profile        string       `json:"profile,omitempty"`
	AccountID      string       `json:"account_id,omitempty"`
	MerchantID     string       `json:"merchant_id,omitempty"`
	LineItems      []LineItem   `json:"line_items,omitempty"`
	Currency       string       `json:"currency,omitempty"`
	AmountCents    int          `json:"amount_cents,omitempty"`
	Servings       int          `json:"servings,omitempty"`
	PlanID         string       `json:"plan_id,omitempty"`
	RecipeSpecHash string       `json:"recipe_spec_hash,omitempty"`
	BasketHash     string       `json:"basket_hash,omitempty"`
	QuoteHash      string       `json:"quote_hash,omitempty"`
	QuoteExpiresAt string       `json:"quote_expires_at,omitempty"`
	BudgetCents    int          `json:"budget_cents,omitempty"`
	MealsPerDay    int          `json:"meals_per_day,omitempty"`
	StylesByMeal   StylesByMeal `json:"styles_by_meal,omitempty"`
	Domain         string       `json:"domain,omitempty"`
	Schedule       Schedule     `json:"schedule,omitempty"`
}

// Helper methods to unmarshal constraints
func (a *AccessItem) GetShoppingConstraints() (*ShoppingConstraints, error) {
	if a.Constraints == nil {
		return nil, nil
	}
	var constraints ShoppingConstraints
	err := json.Unmarshal(a.Constraints, &constraints)
	return &constraints, err
}

func (a *AccessItem) GetFinanceConstraints() (*FinanceConstraints, error) {
	if a.Constraints == nil {
		return nil, nil
	}
	var constraints FinanceConstraints
	err := json.Unmarshal(a.Constraints, &constraints)
	return &constraints, err
}

type Constraint interface {
	Type() string
	Validate() error
}

func (s ShoppingConstraints) Type() string { return "shopping" }

func (s ShoppingConstraints) Validate() error {
	if s.BudgetCents < 0 {
		return fmt.Errorf("budget_cents cannot be negative")
	}
	if s.AmountCents < 0 {
		return fmt.Errorf("amount_cents cannot be negative")
	}
	if s.AmountCents > s.BudgetCents && s.BudgetCents > 0 {
		return fmt.Errorf("amount_cents (%d) cannot exceed budget_cents (%d)", s.AmountCents, s.BudgetCents)
	}
	if s.Servings < 0 {
		return fmt.Errorf("servings cannot be negative")
	}
	if s.MealsPerDay < 0 {
		return fmt.Errorf("meals_per_day cannot be negative")
	}
	for _, item := range s.LineItems {
		if item.Qty < 0 {
			return fmt.Errorf("line item %s quantity cannot be negative", item.SKU)
		}
		if item.SKU == "" {
			return fmt.Errorf("line item SKU cannot be empty")
		}
	}
	return nil
}

func (s ShoppingConstraints) IsQuoteExpired() bool {
	if s.QuoteExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, s.QuoteExpiresAt)
	if err != nil {
		return true // treat parse errors as expired
	}
	return time.Now().After(expiresAt)
}

func (s ShoppingConstraints) TotalItemsCount() int {
	total := 0
	for _, item := range s.LineItems {
		total += item.Qty
	}
	return total
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

	ApproveGrant(ctx context.Context, id string, approved AccessTokenRequest, subject string) (*GrantState, error)
	DenyGrant(ctx context.Context, id string) (*GrantState, error)

	MarkCodeVerified(ctx context.Context, id string) error
}

type KeyPair struct {
	PrivateKey JWK
	PublicKey  JWK
}

type Proof string

const (
	HTTPSig    Proof = "httpsig"
	MTLs       Proof = "mtls"
	DPoP       Proof = "dpop"
	JSECP256k1 Proof = "jsecp256k1"
)

type ProofMethod struct {
	HTTPSig Proof
	MTLs    Proof
	DPoP    Proof
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
