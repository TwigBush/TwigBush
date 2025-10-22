package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

// capture of write calls made through the osWriteFile seam
type writeCall struct {
	path string
	data []byte
	perm uint32
}

func TestGenerateKey_WritesExpectedFilesAndFields(t *testing.T) {
	// Save and restore seam
	oldWrite := osWriteFile
	t.Cleanup(func() { osWriteFile = oldWrite })

	var calls []writeCall
	osWriteFile = func(path string, b []byte, perm uint32) error {
		// capture a copy of the bytes
		cp := make([]byte, len(b))
		copy(cp, b)
		calls = append(calls, writeCall{path: path, data: cp, perm: perm})
		return nil
	}

	dir := t.TempDir()
	privPath, thumb, err := generateKey(dir, "jwk")
	if err != nil {
		t.Fatalf("generateKey error: %v", err)
	}
	if thumb == "" {
		t.Fatalf("thumbprint is empty")
	}
	// Expect two writes: private then public
	if len(calls) != 2 {
		t.Fatalf("expected 2 writes, got %d", len(calls))
	}

	wantPriv := filepath.Join(dir, "key-"+thumb+".jwk")
	wantPub := filepath.Join(dir, "key-"+thumb+".pub.jwk")

	if calls[0].path != wantPriv {
		t.Fatalf("private path = %s, want %s", calls[0].path, wantPriv)
	}
	if calls[1].path != wantPub {
		t.Fatalf("public path = %s, want %s", calls[1].path, wantPub)
	}
	if privPath != wantPriv {
		t.Fatalf("returned privPath = %s, want %s", privPath, wantPriv)
	}

	// Check file permissions passed to the seam
	if calls[0].perm != 0o600 {
		t.Fatalf("private perm = %o, want 0600", calls[0].perm)
	}
	if calls[1].perm != 0o644 {
		t.Fatalf("public perm = %o, want 0644", calls[1].perm)
	}

	// Inspect the JSON written for private key
	var privObj map[string]any
	if err := json.Unmarshal(calls[0].data, &privObj); err != nil {
		t.Fatalf("private JSON unmarshal: %v", err)
	}

	// Private key fields
	if got := privObj["kid"]; got != thumb {
		t.Fatalf("private kid = %v, want %s", got, thumb)
	}
	if got := privObj["alg"]; got != "ES384" {
		t.Fatalf("private alg = %v, want ES384", got)
	}
	if got := privObj["kty"]; got != "EC" {
		t.Fatalf("private kty = %v, want EC", got)
	}
	if got := privObj["crv"]; got != "P-384" {
		t.Fatalf("private crv = %v, want P-384", got)
	}
	if _, ok := privObj["d"]; !ok {
		t.Fatalf("private key missing 'd' field")
	}

	// Inspect the JSON written for public key
	var pubObj map[string]any
	if err := json.Unmarshal(calls[1].data, &pubObj); err != nil {
		t.Fatalf("public JSON unmarshal: %v", err)
	}

	if got := pubObj["kid"]; got != thumb {
		t.Fatalf("public kid = %v, want %s", got, thumb)
	}
	if _, ok := pubObj["d"]; ok {
		t.Fatalf("public key should not contain 'd'")
	}

	// Recompute thumbprint from the public JWK bytes and compare
	pubKey, err := jwk.ParseKey(calls[1].data)
	if err != nil {
		t.Fatalf("parse public jwk: %v", err)
	}
	tp2, err := jwkThumbprint(pubKey)
	if err != nil {
		t.Fatalf("jwkThumbprint: %v", err)
	}
	if tp2 != thumb {
		t.Fatalf("thumbprint mismatch: got %s, want %s", tp2, thumb)
	}
}

func TestGenerateKey_PropagatesWriteError(t *testing.T) {
	oldWrite := osWriteFile
	t.Cleanup(func() { osWriteFile = oldWrite })

	// Fail on the first write
	osWriteFile = func(path string, b []byte, perm uint32) error {
		return fmt.Errorf("boom")
	}

	_, _, err := generateKey(t.TempDir(), "jwk")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected write error to propagate, got %v", err)
	}
}

func TestJWKThumbprint_NoPaddingAndNonEmpty(t *testing.T) {
	// Create a tiny public JWK and compute thumbprint
	j := []byte(`{"kty":"EC","crv":"P-256","x":"AQ","y":"AQ"}`)
	key, err := jwk.ParseKey(j)
	if err != nil {
		t.Fatalf("parse jwk: %v", err)
	}
	tp, err := jwkThumbprint(key)
	if err != nil {
		t.Fatalf("jwkThumbprint: %v", err)
	}
	if tp == "" {
		t.Fatalf("thumbprint empty")
	}
	if strings.Contains(tp, "=") {
		t.Fatalf("thumbprint should be base64url without padding, got %q", tp)
	}
}
