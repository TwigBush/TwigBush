package token

import "github.com/TwigBush/gnap-go/internal/types"

type Token struct {
	Value  string             `json:"value"`
	Access []types.AccessItem `json:"access"`
	Label  string             `json:"label"`
}
