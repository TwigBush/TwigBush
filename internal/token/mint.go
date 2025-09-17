package token

import "time"

type MintResult struct {
	AccessToken string
	ExpiresAt   time.Time
}

func MintShortLivedKeyBound() (*MintResult, error) {
	// TODO
	return &MintResult{AccessToken: "opaque", ExpiresAt: time.Now().Add(60 * time.Second)}, nil
}
