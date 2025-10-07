package cli

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func generateKey(dir, keyType string) (path string, thumb string, err error) {
	fmt.Printf("Generating ES384 key...\n")

	// Generate P-384 ECDSA key
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Import into JWK
	privKey, err := jwk.Import(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to import private key: %w", err)
	}

	// Set algorithm to ES384 (P-384 with SHA-384)
	if err := privKey.Set(jwk.AlgorithmKey, jwa.ES384()); err != nil {
		return "", "", fmt.Errorf("failed to set algorithm: %w", err)
	}

	// Get public key
	pubKey, err := jwk.PublicKeyOf(privKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Calculate thumbprint
	tp, err := jwkThumbprint(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate thumbprint: %w", err)
	}

	// Set key ID based on thumbprint
	if err := privKey.Set(jwk.KeyIDKey, tp); err != nil {
		return "", "", fmt.Errorf("failed to set private key ID: %w", err)
	}
	if err := pubKey.Set(jwk.KeyIDKey, tp); err != nil {
		return "", "", fmt.Errorf("failed to set public key ID: %w", err)
	}

	// Define file paths
	privPath := filepath.Join(dir, fmt.Sprintf("key-%s.jwk", tp))
	pubPath := filepath.Join(dir, fmt.Sprintf("key-%s.pub.jwk", tp))

	// Write private key
	privJSON, err := json.MarshalIndent(privKey, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}
	if err := writeFile(privPath, privJSON, 0o600); err != nil {
		return "", "", err
	}

	// Write public key
	pubJSON, err := json.MarshalIndent(pubKey, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	if err := writeFile(pubPath, pubJSON, 0o644); err != nil {
		return "", "", err
	}

	return privPath, tp, nil
}

// jwkThumbprint computes RFC 7638 thumbprint using jwx
func jwkThumbprint(key jwk.Key) (string, error) {
	// jwx provides built-in thumbprint calculation
	tp, err := key.Thumbprint(crypto.SHA384)
	if err != nil {
		return "", fmt.Errorf("failed to compute thumbprint: %w", err)
	}
	return b64u(tp), nil
}

func writeFile(path string, b []byte, perm uint32) error {
	return osWriteFile(path, b, perm)
}
