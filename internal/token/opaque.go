package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/types"
)

// IssueOpaqueConfig contains configuration for issuing opaque tokens
type IssueOpaqueConfig struct {
	Issuer          string
	Audience        []string
	TokenTTLSeconds int
	BoundProof      string          // "httpsig", "dpop", etc.
	ClientJWK       json.RawMessage // the client's bound key
	Subject         string
	InstanceID      string
}

// IssueOpaqueToken generates a cryptographically random opaque token,
// stores only its hash with the token metadata, and returns the token value.
func IssueOpaqueToken(ctx context.Context, store *gnap.TokenStoreContainer, access []types.AccessItem, cfg IssueOpaqueConfig) (string, error) {
	// Generate 32 random bytes
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	// Encode as base64url - this is what the client receives
	tokenValue := base64.RawURLEncoding.EncodeToString(raw)

	// Hash the token value - this is what we store
	sum := sha256.Sum256([]byte(tokenValue))
	hashB64 := base64.RawURLEncoding.EncodeToString(sum[:])

	now := time.Now().Unix()
	exp := now + int64(cfg.TokenTTLSeconds)

	// Build the token record
	record := &gnap.TokenRecord{
		HashB64:    hashB64,
		Iss:        cfg.Issuer,
		Access:     access,
		Aud:        cfg.Audience,
		Sub:        cfg.Subject,
		InstanceID: cfg.InstanceID,
		Iat:        now,
		Exp:        exp,
		Nbf:        now,
		Revoked:    false,
	}

	// Add key binding if provided
	if cfg.BoundProof != "" && len(cfg.ClientJWK) > 0 {
		record.BoundProof = cfg.BoundProof
		record.BoundKey = &gnap.BoundKey{
			Proof: cfg.BoundProof,
			JWK:   cfg.ClientJWK,
		}
	}

	// Store by hash
	if err := store.Put(ctx, hashB64, record); err != nil {
		return "", err
	}

	// Return the actual token value to the client
	return tokenValue, nil
}
