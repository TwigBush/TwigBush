package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to write a file
func testWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestRegisterKeyWithAS_PublicKeyMissing(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "key.jwk")    // we do not need to write it, code does not read it
	pub := filepath.Join(dir, "key.pub.jwk") // not created

	err := registerKeyWithAS(priv, "http://example.invalid", "default", "checkout", "")
	if err == nil {
		t.Fatalf("expected error for missing public key")
	}
	// Error should mention the derived .pub.jwk path
	if !strings.Contains(err.Error(), pub) {
		t.Fatalf("error did not mention missing pub path %q: %v", pub, err)
	}
}

func TestRegisterKeyWithAS_Success_UsesDerivedPubAndAuthorization(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "key.jwk")
	pub := filepath.Join(dir, "key.pub.jwk")

	// Minimal valid JSON for a JWK object
	testWriteFile(t, priv, []byte(`{"dummy":"priv"}`)) // not read by the function
	testWriteFile(t, pub, []byte(`{"kty":"EC","crv":"P-256","x":"AA","y":"BB"}`))

	seen := struct {
		method string
		path   string
		auth   string
		ct     string
		body   map[string]any
	}{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.method = r.Method
		seen.path = r.URL.Path
		seen.auth = r.Header.Get("Authorization")
		seen.ct = r.Header.Get("Content-Type")

		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seen.body)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	asURL := srv.URL + "/" // exercise TrimRight on asURL
	tenant := "default"
	rsID := "checkout"
	token := "token123"

	err := registerKeyWithAS(priv, asURL, tenant, rsID, token)
	if err != nil {
		t.Fatalf("registerKeyWithAS error: %v", err)
	}

	// Request checks
	if seen.method != http.MethodPost {
		t.Fatalf("method = %s, want POST", seen.method)
	}
	wantPath := "/admin/tenants/" + tenant + "/rs/keys"
	if seen.path != wantPath {
		t.Fatalf("path = %s, want %s", seen.path, wantPath)
	}
	if seen.ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", seen.ct)
	}
	if seen.auth != "Bearer "+token {
		t.Fatalf("Authorization = %q, want %q", seen.auth, "Bearer "+token)
	}

	// Body checks
	if got := seen.body["display_rs"]; got != rsID {
		t.Fatalf("display_rs = %v, want %s", got, rsID)
	}
	if got := seen.body["alg"]; got != "ES384" {
		t.Fatalf("alg = %v, want ES384", got)
	}
	j, ok := seen.body["jwk"].(map[string]any)
	if !ok || len(j) == 0 {
		t.Fatalf("jwk object missing or empty: %#v", seen.body["jwk"])
	}
}

func TestRegisterKeyWithAS_Success_WhenPrivIsPubFile(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "my.pub.jwk")
	testWriteFile(t, pub, []byte(`{}`))

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	err := registerKeyWithAS(pub, srv.URL, "acme", "orders", "")
	if err != nil {
		t.Fatalf("registerKeyWithAS error: %v", err)
	}
	if gotPath != "/admin/tenants/acme/rs/keys" {
		t.Fatalf("path = %s, want /admin/tenants/acme/rs/keys", gotPath)
	}
}

func TestRegisterKeyWithAS_Non2xx(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "k.pub.jwk")
	testWriteFile(t, pub, []byte(`{}`))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusBadRequest)
	}))
	defer srv.Close()

	err := registerKeyWithAS(pub, srv.URL, "default", "checkout", "")
	if err == nil {
		t.Fatalf("expected error on non 2xx response")
	}
	// The function reports "AS returned <status>: <body>"
	// It reads the body earlier, so body may be empty. Check the status part.
	if !strings.Contains(err.Error(), "AS returned 400 Bad Request") {
		t.Fatalf("unexpected error: %v", err)
	}
}
