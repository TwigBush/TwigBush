package gnap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/google/uuid"
)

type MemoryStore struct {
	mu     sync.RWMutex
	grants map[string]*types.GrantState
	cfg    types.Config
}

func NewMemoryStore(cfg types.Config) *MemoryStore {
	return &MemoryStore{
		grants: make(map[string]*types.GrantState),
		cfg:    cfg,
	}
}

func (s *MemoryStore) CreateGrant(ctx context.Context, req types.GrantRequest) (*types.GrantState, error) {
	now := time.Now().UTC()
	exp := now.Add(time.Duration(s.cfg.GrantTTLSeconds) * time.Second)

	cont := randHex(16) // 32 hex chars

	// Collect locations (optional convenience for clients)
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

	uc := RandUserCode()

	state := &types.GrantState{
		ID:                uuid.NewString(),
		Status:            types.GrantStatusPending,
		Client:            req.Client,
		RequestedAccess:   req.Access,
		ContinuationToken: cont,
		TokenFormat:       req.TokenFormat,
		CreatedAt:         now,
		UpdatedAt:         now,
		ExpiresAt:         exp,
		Locations:         locRaw,
		UserCode:          &uc, // device/user_code flow
	}

	s.mu.Lock()
	s.grants[state.ID] = state
	s.mu.Unlock()

	return state, nil
}

func (s *MemoryStore) MarkCodeVerified(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grants[id]
	if !ok {
		return fmt.Errorf("grant not found")
	}
	if g.Status != types.GrantStatusPending {
		return fmt.Errorf("grant not pending")
	}

	g.CodeVerified = true
	g.UpdatedAt = time.Now().UTC()
	s.grants[id] = g
	return nil
}

func (s *MemoryStore) GetGrant(ctx context.Context, id string) (*types.GrantState, bool) {
	s.mu.RLock()
	g, ok := s.grants[id]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}

	// Mutate to expired-on-read if TTL elapsed (matches your Java behavior)
	now := time.Now().UTC()
	if now.After(g.ExpiresAt) && g.Status != types.GrantStatusExpired {
		s.mu.Lock()
		g.Status = types.GrantStatusExpired
		g.UpdatedAt = now
		s.grants[id] = g
		s.mu.Unlock()
	}

	return g, true
}

func (s *MemoryStore) FindGrantByUserCodePending(ctx context.Context, code string) (*types.GrantState, bool) {
	if code == "" {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.grants {
		if g == nil || g.Status != types.GrantStatusPending {
			continue
		}
		if g.UserCode != nil && *g.UserCode == code {
			// (Optional) You could also check expiry here similar to GetGrant
			return g, true
		}
	}
	return nil, false
}

// AccessItem -> GrantedAccess projector (kept for when/if you store GrantedAccess explicitly)
func accessToGranted(a types.AccessItem) types.GrantedAccess {
	ga := types.GrantedAccess{Type: a.Type}
	// resource_id: prefer explicit, else fallback to "<type>:*"
	if a.ResourceID != "" {
		ga.ResourceID = a.ResourceID
	} else if a.Type != "" {
		ga.ResourceID = a.Type + ":*"
	}
	// Map actions -> rights (can refine later)
	if len(a.Actions) > 0 {
		ga.Rights = append(ga.Rights, a.Actions...)
	}
	// scopes/resource_server not present in AccessItem; leave empty or derive later
	return ga
}

func (s *MemoryStore) ApproveGrant(ctx context.Context, id string, approved []types.AccessItem, subject string) (*types.GrantState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grants[id]
	if !ok {
		return nil, fmt.Errorf("grant not found")
	}
	if g.Status != types.GrantStatusPending {
		return nil, fmt.Errorf("grant not pending")
	}
	// Enforce device code ordering for production-like behavior
	if !g.CodeVerified {
		return nil, fmt.Errorf("code not verified")
	}

	// If caller didn't specify, approve exactly what was requested
	if len(approved) == 0 {
		approved = g.RequestedAccess
	}

	g.Status = types.GrantStatusApproved
	g.ApprovedAccess = approved
	g.Subject = &subject
	g.UpdatedAt = time.Now().UTC()
	s.grants[id] = g
	return g, nil
}

func (s *MemoryStore) DenyGrant(ctx context.Context, id string) (*types.GrantState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grants[id]
	if !ok {
		return nil, errors.New("grant not found")
	}

	now := time.Now().UTC()
	if now.After(g.ExpiresAt) {
		g.Status = types.GrantStatusExpired
		g.UpdatedAt = now
		s.grants[id] = g
		return nil, errors.New("grant expired")
	}

	g.Status = types.GrantStatusDenied
	g.UpdatedAt = now
	s.grants[id] = g
	return g, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
