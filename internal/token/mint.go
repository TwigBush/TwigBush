package token

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/google/uuid"
)

type IssueConfig struct {
	Issuer          string
	TokenTTLSeconds int64
	Audience        string // e.g. "mcp-resource-servers"
}

func IssueToken(grant *types.GrantState, cfg IssueConfig) ([]*Token, error) {
	if len(grant.ApprovedAccess) == 0 {
		return nil, ErrNotApproved
	}

	now := time.Now().UTC()
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

	// interate through approved access to mint tokens per resource

	var tokens []*Token
	for _, g := range grant.ApprovedAccess {
		log.Printf("grant %v", g)

		t := &Token{
			// todo - address issuing a token
			Value:  "some-specific-token-value-" + string(jwt[0:10]),
			Access: g.Access,
		}
		tokens = append(tokens, t)
	}

	return tokens, nil
}

var ErrNotApproved = gnap.Err("grant not approved")

func subjectOrAnon(s *string) string {
	if s == nil || *s == "" {
		return "anonymous"
	}
	return *s
}
