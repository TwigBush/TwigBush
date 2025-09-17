package gnap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu     sync.RWMutex
	grants map[string]*GrantState
	cfg    Config
}

func NewMemoryStore(cfg Config) *MemoryStore {
	return &MemoryStore{
		grants: make(map[string]*GrantState),
		cfg:    cfg,
	}
}

func (s *MemoryStore) CreateGrant(ctx context.Context, req GrantRequest) (*GrantState, error) {
	now := time.Now().UTC()
	exp := now.Add(time.Duration(s.cfg.GrantTTLSeconds) * time.Second)

	cont := randHex(16) // 32 hex chars

	// collect locations (optional convenience for clients)
	var locs []string
	for _, a := range req.Access {
		if len(a.Locations) > 0 {
			locs = append(locs, a.Locations...)
		}
	}
	var locRaw json.RawMessage
	if len(locs) > 0 {
		locRaw, _ = json.Marshal(locs)
	}

	state := &GrantState{
		ID:                uuid.NewString(),
		Status:            GrantStatusPending,
		Client:            req.Client,
		RequestedAccess:   req.Access,
		ContinuationToken: cont,
		TokenFormat:       req.TokenFormat,
		CreatedAt:         now,
		UpdatedAt:         now,
		ExpiresAt:         exp,
		Locations:         locRaw,
	}

	s.mu.Lock()
	s.grants[state.ID] = state
	s.mu.Unlock()

	return state, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
