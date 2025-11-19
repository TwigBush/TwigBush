package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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

	tokenValue := base64.RawURLEncoding.EncodeToString(raw)

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

// Validation errors.
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
	ErrNotYetValid  = errors.New("token not yet valid")
	ErrRevokedToken = errors.New("revoked token")
)

// ValidateOpaqueConfig controls issuer and audience checks.
type ValidateOpaqueConfig struct {
	ExpectedIssuer   string
	ExpectedAudience []string
}

// ValidateOpaqueToken hashes the incoming token, loads the record from the store,
// and checks expiry, nbf, revoked flag, and iss/aud.
func ValidateOpaqueToken(
	ctx context.Context,
	store *gnap.TokenStoreContainer,
	tokenValue string,
	cfg ValidateOpaqueConfig,
) (*gnap.TokenRecord, error) {
	if tokenValue == "" {
		return nil, ErrInvalidToken
	}

	// Compute hash in the same way as IssueOpaqueToken
	sum := sha256.Sum256([]byte(tokenValue))
	hashB64 := base64.RawURLEncoding.EncodeToString(sum[:])

	rec, err := store.GetByHash(ctx, hashB64)
	if err != nil {
		// For security, treat any load error as "invalid" to callers.
		return nil, ErrInvalidToken
	}

	now := time.Now().Unix()
	if rec.Revoked {
		return nil, ErrRevokedToken
	}
	if now < rec.Nbf {
		return nil, ErrNotYetValid
	}
	if now > rec.Exp {
		return nil, ErrExpiredToken
	}

	if cfg.ExpectedIssuer != "" && rec.Iss != cfg.ExpectedIssuer {
		return nil, ErrInvalidToken
	}
	if len(cfg.ExpectedAudience) != 0 && !hasOverlap(rec.Aud, cfg.ExpectedAudience) {
		return nil, ErrInvalidToken
	}

	return rec, nil
}

func hasOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}
