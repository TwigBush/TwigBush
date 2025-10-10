package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetFlags puts globals and persistent flags back to their defaults so tests do not
// bleed state into each other.
func resetFlags(t *testing.T) {
	t.Helper()

	home, _ := os.UserHomeDir()
	defaultCfg := filepath.Join(home, ".twigbush", "config.yaml")

	// Reset bound variables via flags (since StringVar/BoolVar bind the variables).
	_ = rootCmd.PersistentFlags().Set("output", "json")
	_ = rootCmd.PersistentFlags().Set("show-curl", "false")
	_ = rootCmd.PersistentFlags().Set("as-base-url", "http://localhost:8085")
	_ = rootCmd.PersistentFlags().Set("rs-base-url", "http://localhost:8088")
	_ = rootCmd.PersistentFlags().Set("config", defaultCfg)

	// Clear CLI args for the next Execute call.
	rootCmd.SetArgs([]string{})
	// Write help and other cobra output to a buffer by default in tests.
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
}

func TestRootDefaultsAndFlags(t *testing.T) {
	resetFlags(t)

	if got, want := rootCmd.Use, "twigbush"; got != want {
		t.Fatalf("Use = %q, want %q", got, want)
	}
	if got, want := rootCmd.Short, "TwigBush developer CLI for GNAP flows"; got != want {
		t.Fatalf("Short = %q, want %q", got, want)
	}
	if !rootCmd.SilenceUsage {
		t.Fatalf("SilenceUsage = false, want true")
	}
	if !rootCmd.SilenceErrors {
		t.Fatalf("SilenceErrors = false, want true")
	}

	home, _ := os.UserHomeDir()
	wantCfg := filepath.Join(home, ".twigbush", "config.yaml")

	if output != "json" {
		t.Fatalf("output default = %q, want %q", output, "json")
	}
	if showCurl {
		t.Fatalf("showCurl default = true, want false")
	}
	if asBaseURL != "http://localhost:8085" {
		t.Fatalf("asBaseURL default = %q, want %q", asBaseURL, "http://localhost:8085")
	}
	if rsBaseURL != "http://localhost:8088" {
		t.Fatalf("rsBaseURL default = %q, want %q", rsBaseURL, "http://localhost:8088")
	}
	if cfgPath != wantCfg {
		t.Fatalf("config default = %q, want %q", cfgPath, wantCfg)
	}
}

func TestHelpCommandRuns(t *testing.T) {
	resetFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"help"})

	if err := Execute(); err != nil {
		t.Fatalf("help Execute() error = %v", err)
	}
	out := buf.String()
	// Minimal assertion that usage/help was printed.
	if !strings.Contains(out, "twigbush") || !strings.Contains(out, "Show help") && !strings.Contains(out, "Usage:") {
		t.Fatalf("help output did not contain expected text; got:\n%s", out)
	}
}

func TestExecuteNoArgsPrintsHint(t *testing.T) {
	resetFlags(t)

	// Capture os.Stdout since rootCmd.Run uses fmt.Println (not cmd.Print*)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// No args triggers the Run func that prints the friendly hint.
	rootCmd.SetArgs([]string{})
	err := Execute()

	// Restore and read captured output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out, "Use -h for help") {
		t.Fatalf("expected hint to be printed, got:\n%s", out)
	}
}

func TestFlagOverridesAreApplied(t *testing.T) {
	resetFlags(t)

	// Override a couple of flags and ensure globals are updated.
	rootCmd.SetArgs([]string{
		"--output", "yaml",
		"--show-curl",
		"--as-base-url", "http://example-as:8085",
		"--rs-base-url", "http://example-rs:8088",
	})

	// Capture stdout to drain any hint output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := Execute()
	w.Close()
	os.Stdout = old
	_, _ = io.Copy(io.Discard, r)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output != "yaml" {
		t.Fatalf("output = %q, want %q", output, "yaml")
	}
	if !showCurl {
		t.Fatalf("showCurl = false, want true")
	}
	if asBaseURL != "http://example-as:8085" {
		t.Fatalf("asBaseURL = %q, want %q", asBaseURL, "http://example-as:8085")
	}
	if rsBaseURL != "http://example-rs:8088" {
		t.Fatalf("rsBaseURL = %q, want %q", rsBaseURL, "http://example-rs:8088")
	}
}
