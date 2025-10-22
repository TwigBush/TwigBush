package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newBufCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := cmdKeysRegister()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	return cmd, &buf
}

// Sanity: flags exist and have expected defaults
func TestCmdKeysRegister_FlagsPresent(t *testing.T) {
	c, _ := newBufCmd()

	for _, name := range []string{"as", "tenant", "rs-id", "admin-token", "key"} {
		if f := c.Flags().Lookup(name); f == nil {
			t.Fatalf("expected flag %q to be registered", name)
		}
	}

	tenant := c.Flags().Lookup("tenant")
	if tenant.DefValue != "default" {
		t.Fatalf("tenant default = %q, want %q", tenant.DefValue, "default")
	}
}

// No flags and no config: should require --as
// if .twigbush/config.yaml exists, this test will fail.
func TestCmdKeysRegister_RequiresAS(t *testing.T) {
	c, _ := newBufCmd()
	c.SetArgs([]string{}) // no flags

	err := c.Execute()
	if err == nil || err.Error() != "--as is required" {
		t.Fatalf("got error %v, want %q", err, "--as is required")
	}
}

// Has --as but missing --rs-id: should require rs-id
func TestCmdKeysRegister_RequiresRSID(t *testing.T) {
	c, _ := newBufCmd()
	c.SetArgs([]string{"--as", "http://localhost:8089"})

	err := c.Execute()
	if err == nil || err.Error() != "--rs-id is required" {
		t.Fatalf("got error %v, want %q", err, "--rs-id is required")
	}
}

// Has --as and --rs-id but no key and no config default: should require key
func TestCmdKeysRegister_RequiresKeyWithoutConfig(t *testing.T) {
	c, _ := newBufCmd()
	c.SetArgs([]string{
		"--as", "http://localhost:8089",
		"--rs-id", "checkout",
	})

	err := c.Execute()
	want := "--key is required or set default_key in config"
	if err == nil || err.Error() != want {
		t.Fatalf("got error %v, want %q", err, want)
	}
}

// With config file providing as_base_url and default_key, command should NOT fail
// on the required-flag validations. We cannot assert the final success without
// stubbing registerKeyWithAS, so we only assert it gets past the early checks.
func TestCmdKeysRegister_UsesConfigFallbacks(t *testing.T) {
	// Create a temp config file
	dir := t.TempDir()
	cfg := `
as_base_url: "http://cfg-as:8085"
default_key: "` + filepath.ToSlash(filepath.Join(dir, "mykey.jwk")) + `"
`
	cfgFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// Touch the key file so that a later implementation that checks for existence does not fail early
	if err := os.WriteFile(filepath.Join(dir, "mykey.jwk"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write key: %v", err)
	}

	// Point the global cfgPath at our temp file, then restore
	origCfg := cfgPath
	cfgPath = cfgFile
	defer func() { cfgPath = origCfg }()

	c, _ := newBufCmd()
	// Only provide rs-id. as and key should fall back to config values.
	c.SetArgs([]string{"--rs-id", "checkout"})

	err := c.Execute()
	if err == nil {
		// Good. We got past validation and the call returned nil.
		return
	}

	// If there is an error, ensure it is NOT one of the early validation errors.
	earlyFailures := []string{
		"--as is required",
		"--rs-id is required",
		"--key is required or set default_key in config",
	}
	for _, s := range earlyFailures {
		if strings.Contains(err.Error(), s) {
			t.Fatalf("expected to pass validation using config fallbacks, got validation error: %v", err)
		}
	}
	// Any other error indicates we made it past validation (for example, real network call),
	// which is acceptable for this test.
}
