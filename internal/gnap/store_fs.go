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

	"github.com/TwigBush/gnap-go/internal/types"
	"github.com/google/uuid"
)

type FileStore struct {
	root string
	cfg  types.Config
	mu   sync.RWMutex // process-local concurrency
}

func NewFileStore(root string, cfg types.Config) (*FileStore, error) {
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

func (fileStore *FileStore) grantPath(id string) string {
	return filepath.Join(fileStore.root, "grants", id+".json")
}

func (fileStore *FileStore) writeGrant(g *types.GrantState) error {
	path := fileStore.grantPath(g.ID)
	tempFilePath := path + ".tempFilePath"

	bytes, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	// 0600 since grants can include sensitive data
	if err := os.WriteFile(tempFilePath, bytes, 0o600); err != nil {
		return err
	}
	return os.Rename(tempFilePath, path)
}

func (fileStore *FileStore) readGrant(id string) (*types.GrantState, error) {
	path := fileStore.grantPath(id)
	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("grant not found")
		}
		return nil, err
	}
	var grantState types.GrantState
	if err := json.Unmarshal(bytes, &grantState); err != nil {
		return nil, err
	}
	return &grantState, nil
}

func (fileStore *FileStore) listGrantFiles() ([]string, error) {
	dir := filepath.Join(fileStore.root, "grants")
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

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- interface implementation ----------

func (fileStore *FileStore) CreateGrant(ctx context.Context, req types.GrantRequest) (*types.GrantState, error) {
	now := time.Now().UTC()
	expiration := now.Add(time.Duration(fileStore.cfg.GrantTTLSeconds) * time.Second)

	continueToken := randHex(16) // 32 hex chars

	// Collect locations as JSON, same as memory store
	var locations []string
	for _, accessToken := range req.AccessToken {
		for _, accessItem := range accessToken.Access {
			if len(accessItem.Locations) > 0 {
				locations = append(locations, accessItem.Locations...)
			}
		}
	}

	uc := RandUserCode()

	grantState := &types.GrantState{
		ID:                uuid.NewString(),
		Status:            types.GrantStatusPending,
		Client:            req.Client,
		RequestedAccess:   req.AccessToken,
		ContinuationToken: continueToken,
		TokenFormat:       req.TokenFormat,
		CreatedAt:         now,
		UpdatedAt:         now,
		ExpiresAt:         expiration,
		Locations:         locations,
		UserCode:          &uc,
	}

	fileStore.mu.Lock()
	defer fileStore.mu.Unlock()
	if err := fileStore.writeGrant(grantState); err != nil {
		return nil, err
	}
	return grantState, nil
}

func (fileStore *FileStore) MarkCodeVerified(ctx context.Context, id string) error {
	fileStore.mu.Lock()
	defer fileStore.mu.Unlock()

	grant, err := fileStore.readGrant(id)
	if err != nil {
		return err
	}
	if grant.Status != types.GrantStatusPending {
		return fmt.Errorf("grant not pending")
	}
	grant.CodeVerified = true
	grant.UpdatedAt = time.Now().UTC()
	return fileStore.writeGrant(grant)
}

func (fileStore *FileStore) GetGrant(ctx context.Context, id string) (*types.GrantState, bool) {
	fileStore.mu.Lock() // we may update status to expired
	defer fileStore.mu.Unlock()

	grant, err := fileStore.readGrant(id)
	if err != nil {
		return nil, false
	}

	now := time.Now().UTC()
	if now.After(grant.ExpiresAt) && grant.Status != types.GrantStatusExpired {
		grant.Status = types.GrantStatusExpired
		grant.UpdatedAt = now
		_ = fileStore.writeGrant(grant)
	}
	return grant, true
}

func (fileStore *FileStore) FindGrantByUserCodePending(ctx context.Context, code string) (*types.GrantState, bool) {
	if code == "" {
		return nil, false
	}

	fileStore.mu.RLock()
	files, err := fileStore.listGrantFiles()
	fileStore.mu.RUnlock()
	if err != nil {
		return nil, false
	}

	// Linear scan is fine for local mode. If needed later, add a small index file or map.
	for _, p := range files {
		fileStore.mu.RLock()
		b, err := os.ReadFile(p)
		fileStore.mu.RUnlock()
		if err != nil {
			continue
		}
		var grantState types.GrantState
		if err := json.Unmarshal(b, &grantState); err != nil {
			continue
		}
		if grantState.Status != types.GrantStatusPending {
			continue
		}
		if grantState.UserCode != nil && *grantState.UserCode == code {
			return &grantState, true
		}
	}
	return nil, false
}

func (fileStore *FileStore) ApproveGrant(ctx context.Context, id string, approved types.AccessTokenRequest, subject string) (*types.GrantState, error) {
	fileStore.mu.Lock()
	defer fileStore.mu.Unlock()

	grant, err := fileStore.readGrant(id)
	if err != nil {
		return nil, err
	}
	if grant.Status != types.GrantStatusPending {
		return nil, fmt.Errorf("grant not pending")
	}
	if !grant.CodeVerified {
		return nil, fmt.Errorf("code not verified")
	}
	if len(approved) == 0 {
		approved = grant.RequestedAccess
	}

	grant.Status = types.GrantStatusApproved
	grant.ApprovedAccess = approved
	grant.Subject = &subject
	grant.UpdatedAt = time.Now().UTC()

	if err := fileStore.writeGrant(grant); err != nil {
		return nil, err
	}
	return grant, nil
}

func (fileStore *FileStore) DenyGrant(ctx context.Context, id string) (*types.GrantState, error) {
	fileStore.mu.Lock()
	defer fileStore.mu.Unlock()

	grant, err := fileStore.readGrant(id)
	if err != nil {
		return nil, errors.New("grant not found")
	}

	now := time.Now().UTC()
	if now.After(grant.ExpiresAt) {
		grant.Status = types.GrantStatusExpired
		grant.UpdatedAt = now
		_ = fileStore.writeGrant(grant)
		return nil, errors.New("grant expired")
	}

	grant.Status = types.GrantStatusDenied
	grant.UpdatedAt = now
	if err := fileStore.writeGrant(grant); err != nil {
		return nil, err
	}
	return grant, nil
}
