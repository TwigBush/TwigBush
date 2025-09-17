package gnap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
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
	uc := RandUserCode()
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
		UserCode:          &uc,
	}

	s.mu.Lock()
	s.grants[state.ID] = state
	s.mu.Unlock()

	return state, nil
}

func (s *MemoryStore) GetGrant(ctx context.Context, id string) (*GrantState, bool) {
	s.mu.RLock()
	g, ok := s.grants[id]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}

	// mutate if expired (like Java)
	now := time.Now().UTC()
	if now.After(g.ExpiresAt) && g.Status != GrantStatusExpired {
		s.mu.Lock()
		g.Status = GrantStatusExpired
		g.UpdatedAt = now
		s.grants[id] = g
		s.mu.Unlock()
	}
	return g, true
}
func (s *MemoryStore) FindGrantByUserCodePending(ctx context.Context, code string) (*GrantState, bool) {
	if code == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.grants {
		log.Printf("checking grant %s", g.ID)
		if g == nil || g.Status != GrantStatusPending {
			continue
		}
		if g.UserCode != nil && *g.UserCode == code {
			// Note: expiry check like GetGrant could be added if desired
			return g, true
		}
	}
	return nil, false
}

func accessToGranted(a AccessItem) GrantedAccess {
	ga := GrantedAccess{
		Type: a.Type,
	}
	// resource_id: prefer explicit, else fallback to "<type>:*"
	if a.ResourceID != "" {
		ga.ResourceID = a.ResourceID
	} else if a.Type != "" {
		ga.ResourceID = a.Type + ":*"
	}
	// use Actions as "rights" by analogy (you can refine later)
	if len(a.Actions) > 0 {
		ga.Rights = append(ga.Rights, a.Actions...)
	}
	// scopes/resource_server not present in AccessItem; leave empty or derive later
	return ga
}

func (s *MemoryStore) ApproveGrant(ctx context.Context, id string, approved []AccessItem, subject string) (*GrantState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grants[id]
	if !ok {
		return nil, errors.New("grant not found")
	}

	// expire-on-read check like GetGrant
	now := time.Now().UTC()
	if now.After(g.ExpiresAt) {
		g.Status = GrantStatusExpired
		g.UpdatedAt = now
		s.grants[id] = g
		return nil, errors.New("grant expired")
	}

	// map requested/approved -> GrantedAccess (Java shape)
	out := make([]GrantedAccess, 0, len(approved))
	for _, a := range approved {
		out = append(out, accessToGranted(a))
	}

	g.Status = GrantStatusApproved
	g.ApprovedAccessGranted = out
	if subject != "" {
		g.Subject = &subject
	}
	g.UpdatedAt = now
	s.grants[id] = g
	return g, nil
}

func (s *MemoryStore) DenyGrant(ctx context.Context, id string) (*GrantState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grants[id]
	if !ok {
		return nil, errors.New("grant not found")
	}

	now := time.Now().UTC()
	if now.After(g.ExpiresAt) {
		g.Status = GrantStatusExpired
		g.UpdatedAt = now
		s.grants[id] = g
		return nil, errors.New("grant expired")
	}

	g.Status = GrantStatusDenied
	g.UpdatedAt = now
	s.grants[id] = g
	return g, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
