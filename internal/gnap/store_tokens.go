package gnap

import (
	"context"
	"encoding/json"

	"os"
	"path/filepath"
	"sync"

	"github.com/TwigBush/gnap-go/internal/types"
)

// ==== Token storage and validation contracts ====

type TokenRecord struct {
	HashB64    string
	Value      string
	Iss        string
	Access     []types.AccessItem
	Aud        []string
	Sub        string
	InstanceID string
	Exp        int64
	Iat        int64
	Nbf        int64
	Revoked    bool

	// Binding
	BoundProof string    // e.g., "httpsig", "dpop", "mtls"
	BoundKey   *BoundKey // if bound, one of JWK or Ref populated
}

// AS → RS response when active
type BoundKey struct {
	Proof string          `json:"proof"`
	JWK   json.RawMessage `json:"jwk,omitempty"` // by value
	Ref   string          `json:"ref,omitempty"` // or by reference
}

type TokenStoreContainer struct {
	mu      sync.RWMutex
	dataDir string
	cache   map[string]TokenRecord
}

type TokenStore interface {
	GetByValue(ctx context.Context, token string) (*TokenRecord, error)
	GetByHash(ctx context.Context, hashB64 string) (*TokenRecord, error)
	Put(ctx context.Context, hashB64 string, record *TokenRecord) error
}

func NewTokenStore(dataDir string) (*TokenStoreContainer, error) {
	tokensDir := filepath.Join(dataDir, "tokens")
	if err := os.MkdirAll(tokensDir, 0700); err != nil {
		return nil, err
	}

	store := &TokenStoreContainer{
		dataDir: tokensDir,
		cache:   make(map[string]TokenRecord),
	}

	// Load existing tokens into cache
	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}

	return store, nil
}

// Put stores a token record by its hash
func (s *TokenStoreContainer) Put(ctx context.Context, hashB64 string, record *TokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure HashB64 is set
	record.HashB64 = hashB64

	// Marshal to JSON
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	// Write to disk using hash as filename
	path := filepath.Join(s.dataDir, hashB64+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	// Update cache
	s.cache[hashB64] = *record

	return nil
}

func (s *TokenStoreContainer) GetByHash(ctx context.Context, hashB64 string) (*TokenRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check cache first
	if record, ok := s.cache[hashB64]; ok {
		return &record, nil
	}

	// Read from disk
	path := filepath.Join(s.dataDir, hashB64+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Token not found (not an error)
		}
		return nil, err
	}

	var record TokenRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	// Update cache
	s.mu.RUnlock()
	s.mu.Lock()
	s.cache[hashB64] = record
	s.mu.Unlock()
	s.mu.RLock()

	return &record, nil
}

func (s *TokenStoreContainer) Revoke(ctx context.Context, hashB64 string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing record
	record, err := s.GetByHash(ctx, hashB64)
	if err != nil || record == nil {
		return err
	}

	// Mark as revoked
	record.Revoked = true

	// Save back to disk
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.dataDir, hashB64+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	// Update cache
	s.cache[hashB64] = *record

	return nil
}

func (s *TokenStoreContainer) loadFromDisk() error {
	if _, err := os.Stat(s.dataDir); os.IsNotExist(err) {
		return nil
	}

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.dataDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip problematic files
		}

		var rec TokenRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue // Skip invalid JSON
		}

		// Extract hash from filename (remove .json extension)
		hashB64 := file.Name()[:len(file.Name())-5]
		s.cache[hashB64] = rec
	}

	return nil
}

func (s *TokenStoreContainer) CleanupExpired(ctx context.Context, now int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for hashB64, record := range s.cache {
		if record.Exp > 0 && record.Exp <= now {
			// Remove from disk
			path := filepath.Join(s.dataDir, hashB64+".json")
			_ = os.Remove(path) // Ignore errors

			// Remove from cache
			delete(s.cache, hashB64)
		}
	}

	return nil
}
