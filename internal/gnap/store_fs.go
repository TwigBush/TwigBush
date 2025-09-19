package gnap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type FileStore struct {
	root string
	cfg  Config
	mu   sync.RWMutex // process-local concurrency
}

func NewFileStore(root string, cfg Config) (*FileStore, error) {
	// Make parent ~/.twigbush with 0700 when possible
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, "x")), 0o700); err != nil {
		return nil, fmt.Errorf("create parent dir: %w", err)
	}
	// Create data dir ~/.twigbush/data with 0700
	if err := os.MkdirAll(filepath.Join(root, "grants"), 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &FileStore{root: root, cfg: cfg}, nil
}

// ---------- helpers ----------

func (s *FileStore) grantPath(id string) string {
	return filepath.Join(s.root, "grants", id+".json")
}

func (s *FileStore) writeGrant(g *GrantState) error {
	path := s.grantPath(g.ID)
	tmp := path + ".tmp"

	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	// 0600 since grants can include sensitive data
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *FileStore) readGrant(id string) (*GrantState, error) {
	path := s.grantPath(id)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("grant not found")
		}
		return nil, err
	}
	var g GrantState
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *FileStore) listGrantFiles() ([]string, error) {
	dir := filepath.Join(s.root, "grants")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".json" {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

func randHexDuplicate(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- interface implementation ----------

func (s *FileStore) CreateGrant(ctx context.Context, req GrantRequest) (*GrantState, error) {
	now := time.Now().UTC()
	exp := now.Add(time.Duration(s.cfg.GrantTTLSeconds) * time.Second)

	cont := randHex(16) // 32 hex chars

	// Collect locations as JSON, same as memory store
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

	g := &GrantState{
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
	defer s.mu.Unlock()
	if err := s.writeGrant(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *FileStore) MarkCodeVerified(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, err := s.readGrant(id)
	if err != nil {
		return err
	}
	if g.Status != GrantStatusPending {
		return fmt.Errorf("grant not pending")
	}
	g.CodeVerified = true
	g.UpdatedAt = time.Now().UTC()
	return s.writeGrant(g)
}

func (s *FileStore) GetGrant(ctx context.Context, id string) (*GrantState, bool) {
	s.mu.Lock() // we may update status to expired
	defer s.mu.Unlock()

	g, err := s.readGrant(id)
	if err != nil {
		return nil, false
	}

	now := time.Now().UTC()
	if now.After(g.ExpiresAt) && g.Status != GrantStatusExpired {
		g.Status = GrantStatusExpired
		g.UpdatedAt = now
		_ = s.writeGrant(g)
	}
	return g, true
}

func (s *FileStore) FindGrantByUserCodePending(ctx context.Context, code string) (*GrantState, bool) {
	if code == "" {
		return nil, false
	}

	s.mu.RLock()
	files, err := s.listGrantFiles()
	s.mu.RUnlock()
	if err != nil {
		return nil, false
	}

	// Linear scan is fine for local mode. If needed later, add a small index file or map.
	for _, p := range files {
		s.mu.RLock()
		b, err := os.ReadFile(p)
		s.mu.RUnlock()
		if err != nil {
			continue
		}
		var g GrantState
		if err := json.Unmarshal(b, &g); err != nil {
			continue
		}
		if g.Status != GrantStatusPending {
			continue
		}
		if g.UserCode != nil && *g.UserCode == code {
			return &g, true
		}
	}
	return nil, false
}

func (s *FileStore) ApproveGrant(ctx context.Context, id string, approved []AccessItem, subject string) (*GrantState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, err := s.readGrant(id)
	if err != nil {
		return nil, err
	}
	if g.Status != GrantStatusPending {
		return nil, fmt.Errorf("grant not pending")
	}
	if !g.CodeVerified {
		return nil, fmt.Errorf("code not verified")
	}
	if len(approved) == 0 {
		approved = g.RequestedAccess
	}

	g.Status = GrantStatusApproved
	g.ApprovedAccess = approved
	g.Subject = &subject
	g.UpdatedAt = time.Now().UTC()

	if err := s.writeGrant(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *FileStore) DenyGrant(ctx context.Context, id string) (*GrantState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, err := s.readGrant(id)
	if err != nil {
		return nil, errors.New("grant not found")
	}

	now := time.Now().UTC()
	if now.After(g.ExpiresAt) {
		g.Status = GrantStatusExpired
		g.UpdatedAt = now
		_ = s.writeGrant(g)
		return nil, errors.New("grant expired")
	}

	g.Status = GrantStatusDenied
	g.UpdatedAt = now
	if err := s.writeGrant(g); err != nil {
		return nil, err
	}
	return g, nil
}
