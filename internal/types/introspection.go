package types

type IntrospectionResult struct {
	Active bool     `json:"active"`
	Sub    string   `json:"sub,omitempty"` // subject if known
	Iss    string   `json:"iss,omitempty"` // issuer AS
	Aud    []string `json:"aud,omitempty"` // audiences for which token is valid
	Exp    int64    `json:"exp,omitempty"` // unix seconds
	Iat    int64    `json:"iat,omitempty"` // unix seconds
	Key    any      `json:"key,omitempty"` // bound key info if not bearer
	Flags  []string `json:"flags,omitempty"`
	// You can add Access []AccessItem if your AS returns it
}
