package token

import "github.com/TwigBush/gnap-go/internal/gnap"

type Token struct {
	Value     string            `json:"value"`
	Format    string            `json:"format"` // "jwt"
	Key       gnap.ClientKey    `json:"key"`
	Access    []gnap.AccessItem `json:"access"`
	ExpiresIn int               `json:"expires_in"`
	TokenID   string            `json:"token_id"`
}
