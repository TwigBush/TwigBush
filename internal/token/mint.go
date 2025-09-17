package token

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/google/uuid"
)

type IssueConfig struct {
	Issuer          string
	TokenTTLSeconds int64
	Audience        string // e.g. "mcp-resource-servers"
}

func IssueToken(grant *gnap.GrantState, cfg IssueConfig) (*Token, error) {
	if len(grant.ApprovedAccess) == 0 {
		return nil, ErrNotApproved
	}

	now := time.Now().UTC()
	expiresIn := int(cfg.TokenTTLSeconds)
	iat := now.Unix()
	exp := now.Add(time.Duration(cfg.TokenTTLSeconds) * time.Second).Unix()
	jti := uuid.NewString()

	claims := map[string]any{
		"iss":    cfg.Issuer,
		"sub":    subjectOrAnon(grant.Subject),
		"aud":    cfg.Audience,
		"exp":    exp,
		"iat":    iat,
		"jti":    jti,
		"access": grant.ApprovedAccess,
		"key":    grant.Client.Key,
	}

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)

	// base64url (no padding)
	enc := func(b []byte) string {
		return base64.RawURLEncoding.EncodeToString(b)
	}
	jwt := enc(hb) + "." + enc(pb) + ".dev-signature"

	t := &Token{
		Value:     jwt,
		Format:    "jwt",
		Key:       grant.Client.Key,
		Access:    grant.ApprovedAccess,
		ExpiresIn: expiresIn,
		TokenID:   jti,
	}
	return t, nil
}

var ErrNotApproved = gnap.Err("grant not approved")

func subjectOrAnon(s *string) string {
	if s == nil || *s == "" {
		return "anonymous"
	}
	return *s
}
