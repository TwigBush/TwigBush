package gnap

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

type RSKeyRecord struct {
	Tenant    string          `json:"tenant"`
	Thumb256  string          `json:"thumb256"` // RFC 7638 thumbprint (SHA-256)
	KID       string          `json:"kid,omitempty"`
	Alg       string          `json:"alg,omitempty"`
	PubJWK    json.RawMessage `json:"pub_jwk"` // public key only
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
	RotatedAt *time.Time      `json:"rotated_at,omitempty"`
	DisplayRS string          `json:"display_rs,omitempty"` // optional metadata
}

type RSKeyStore struct {
	mu      sync.RWMutex
	dataDir string
	cache   map[string]map[string]RSKeyRecord // tenant -> thumb256 -> record // todo make this fs or database
}

func NewRSKeyStore(dataDir string) (*RSKeyStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	s := &RSKeyStore{
		dataDir: dataDir,
		cache:   make(map[string]map[string]RSKeyRecord),
	}
	if err := s.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("load from disk: %w", err)
	}
	return s, nil
}

func computeThumb256(pub jwk.Key) (string, error) {
	tp, err := pub.Thumbprint(crypto.SHA256) // RFC 7638
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tp), nil
}

func (s *RSKeyStore) UpsertRSKey(ctx context.Context, tenant string, pub jwk.Key, kid, alg, displayRS string, acceptTOFU bool) (RSKeyRecord, error) {
	thumb, err := computeThumb256(pub)
	if err != nil {
		return RSKeyRecord{}, err
	}

	// Marshal canonical public JWK (no private members)
	pubJSON, err := json.Marshal(pub)
	if err != nil {
		return RSKeyRecord{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cache[tenant]; !ok {
		s.cache[tenant] = make(map[string]RSKeyRecord)
	}

	var rec RSKeyRecord
	// If key already present, keep existing CreatedAt
	if existing, ok := s.cache[tenant][thumb]; ok {
		rec = existing
		rec.KID = kid
		rec.Alg = alg
		rec.DisplayRS = displayRS
		rec.PubJWK = pubJSON
		rec.Active = true
	} else {
		// First-seen key: allow only if policy says TOFU or you are in a trusted admin path
		if !acceptTOFU {
			return RSKeyRecord{}, errors.New("unknown RS key and TOFU disabled")
		}
		rec = RSKeyRecord{
			Tenant:    tenant,
			Thumb256:  thumb,
			KID:       kid,
			Alg:       alg,
			PubJWK:    pubJSON,
			Active:    true,
			CreatedAt: time.Now().UTC(),
			DisplayRS: displayRS,
		}
	}

	s.cache[tenant][thumb] = rec

	// Persist to disk
	if err := s.saveToDisk(tenant, thumb, rec); err != nil {
		return RSKeyRecord{}, fmt.Errorf("save to disk: %w", err)
	}

	return rec, nil
}

func (s *RSKeyStore) GetRSKey(ctx context.Context, tenant, thumb256 string) (RSKeyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if tenantKeys, ok := s.cache[tenant]; ok {
		rec, ok := tenantKeys[thumb256]
		return rec, ok
	}
	return RSKeyRecord{}, false
}

func (s *RSKeyStore) ListRSKeys(ctx context.Context, tenant string) []RSKeyRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := []RSKeyRecord{}
	if tenantKeys, ok := s.cache[tenant]; ok {
		for _, rec := range tenantKeys {
			keys = append(keys, rec)
		}
	}
	return keys
}

func (s *RSKeyStore) DeactivateRSKey(ctx context.Context, tenant, thumb256 string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tenantKeys, ok := s.cache[tenant]; ok {
		if rec, ok := tenantKeys[thumb256]; ok {
			rec.Active = false
			now := time.Now().UTC()
			rec.RotatedAt = &now
			s.cache[tenant][thumb256] = rec
			return s.saveToDisk(tenant, thumb256, rec)
		}
	}
	return errors.New("key not found")
}

func (s *RSKeyStore) saveToDisk(tenant, thumb256 string, rec RSKeyRecord) error {
	tenantDir := filepath.Join(s.dataDir, "rs_keys", tenant)
	if err := os.MkdirAll(tenantDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(tenantDir, thumb256+".json")
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *RSKeyStore) loadFromDisk() error {
	baseDir := filepath.Join(s.dataDir, "rs_keys")
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}

	tenants, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	for _, tenantEntry := range tenants {
		if !tenantEntry.IsDir() {
			continue
		}
		tenant := tenantEntry.Name()
		tenantDir := filepath.Join(baseDir, tenant)

		files, err := os.ReadDir(tenantDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
				continue
			}

			path := filepath.Join(tenantDir, file.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var rec RSKeyRecord
			if err := json.Unmarshal(data, &rec); err != nil {
				continue
			}

			if _, ok := s.cache[tenant]; !ok {
				s.cache[tenant] = make(map[string]RSKeyRecord)
			}
			s.cache[tenant][rec.Thumb256] = rec
		}
	}

	return nil
}

// LookupRSPublicKeyById retrieves a public key by its key ID (kid) from the store.
// It searches across all tenants and returns the first active key with matching kid.
// Returns the crypto.PublicKey interface suitable for signature verification.
func (s *RSKeyStore) LookupRSPublicKeyById(kid string) (crypto.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Search all tenants for the key
	for tenant, keys := range s.cache {
		for _, rec := range keys {
			if rec.KID == kid && rec.Active {
				// Parse the stored JWK
				var raw any
				if err := jwk.ParseRawKey(rec.PubJWK, &raw); err != nil {
					return nil, fmt.Errorf("parse JWK for kid %s in tenant %s: %w", kid, tenant, err)
				}

				switch k := raw.(type) {
				case ed25519.PublicKey:
					return k, nil
				case *ecdsa.PublicKey:
					// (optional) enforce curve
					if k.Curve != elliptic.P256() {
						return nil, fmt.Errorf("unsupported EC curve: %s in tenant %s: %s", k.Curve.Params().Name, kid, tenant)
					}
					return k, nil

				}
			}
		}
	}
	return nil, fmt.Errorf("public key not found for kid: %s", kid)
}

// LookupRSPublicKeyByTenant retrieves a public key by kid from a specific tenant.
// Use this when you know which tenant the request belongs to.
func (s *RSKeyStore) LookupRSPublicKeyByTenant(tenant, kid string) (crypto.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tenantKeys, ok := s.cache[tenant]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", tenant)
	}

	for _, rec := range tenantKeys {
		if rec.KID == kid && rec.Active {
			var raw any
			if err := jwk.ParseRawKey(rec.PubJWK, &raw); err != nil {
				return nil, fmt.Errorf("kid %s: parse raw JWK: %w", kid, err)
			}
			switch k := raw.(type) {
			case ed25519.PublicKey:
				return k, nil
			case *ecdsa.PublicKey:
				// optional: enforce P-256
				if k.Curve != elliptic.P256() {
					return nil, fmt.Errorf("kid %s: unsupported EC curve %s", kid, k.Curve.Params().Name)
				}
				return k, nil
			default:
				return nil, fmt.Errorf("kid %s: unsupported key type %T", kid, k)
			}
		}
	}

	return nil, fmt.Errorf("public key not found for kid %s in tenant %s", kid, tenant)
}

// LookupRSPublicKeyByThumbprint retrieves a public key by its RFC 7638 thumbprint.
// This is the canonical way to look up keys as thumbprints are collision-free.
func (s *RSKeyStore) LookupRSPublicKeyByThumbprint(tenant, thumb256 string) (crypto.PublicKey, error) {
	rec, ok := s.GetRSKey(context.Background(), tenant, thumb256)
	if !ok {
		return nil, fmt.Errorf("key not found for thumbprint: %s", thumb256)
	}
	if !rec.Active {
		return nil, fmt.Errorf("key is inactive")
	}

	var raw any
	if err := jwk.ParseRawKey(rec.PubJWK, &raw); err != nil {
		return nil, fmt.Errorf("parse raw JWK: %w", err)
	}

	switch k := raw.(type) {
	case ed25519.PublicKey:
		return k, nil
	case *ecdsa.PublicKey:
		// optional: enforce P-256 only
		if k.Curve != elliptic.P256() {
			return nil, fmt.Errorf("unsupported EC curve: %s", k.Curve.Params().Name)
		}
		return k, nil
	default:
		return nil, fmt.Errorf("unsupported key type %T", k)
	}
}
