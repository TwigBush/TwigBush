package token

import "github.com/TwigBush/gnap-go/internal/types"

type Token struct {
	Value     string             `json:"value"`
	Format    string             `json:"format"` // "jwt"
	Key       types.ClientKey    `json:"key"`
	Access    []types.AccessItem `json:"access"`
	ExpiresIn int                `json:"expires_in"`
	TokenID   string             `json:"token_id"`
}
