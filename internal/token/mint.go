package token

import (
	"context"
	"encoding/json"
	"log"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/types"
)

type IssueConfig struct {
	Issuer          string
	Audience        []string
	TokenTTLSeconds int
	BoundProof      string          // "httpsig",  etc.
	ClientJWK       json.RawMessage // the client's bound key
}

func IssueToken(ctx context.Context, store *gnap.TokenStoreContainer, grant *types.GrantState, cfg IssueConfig) ([]*Token, error) {
	if len(grant.ApprovedAccess) == 0 {
		return nil, ErrNotApproved
	}

	var tokens []*Token

	// Issue one token per approved access grant
	for _, g := range grant.ApprovedAccess {
		log.Printf("Issuing token for grant: %v", g)

		// Generate opaque token using the new pattern
		tokenValue, err := IssueOpaqueToken(ctx, store, g.Access, IssueOpaqueConfig{
			Issuer:          cfg.Issuer,
			Audience:        cfg.Audience,
			TokenTTLSeconds: cfg.TokenTTLSeconds,
			BoundProof:      cfg.BoundProof,
			ClientJWK:       cfg.ClientJWK,
			Subject:         subjectOrAnon(grant.Subject),
			InstanceID:      grant.ID,
		})
		if err != nil {
			return nil, err
		}

		t := &Token{
			Value:  tokenValue,
			Access: g.Access,
			Label:  g.Label,
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
