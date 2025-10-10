package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCmdKeys_MetadataAndChildren(t *testing.T) {
	t.Parallel()

	c := cmdKeys()
	if c.Use != "keys" {
		t.Fatalf("Use = %q, want %q", c.Use, "keys")
	}
	if c.Short != "Key management" {
		t.Fatalf("Short = %q, want %q", c.Short, "Key management")
	}
	if !c.HasAvailableSubCommands() {
		t.Fatalf("expected subcommands to be available")
	}

	// Expect exactly the two children wired in: new and register
	subs := c.Commands()
	if len(subs) < 2 {
		t.Fatalf("got %d subcommands, want at least 2", len(subs))
	}

	seen := map[string]bool{}
	for _, sc := range subs {
		seen[sc.Name()] = true
		// Each added subcommand should have the keys command as parent
		if sc.Parent() != c {
			t.Fatalf("subcommand %q has wrong parent", sc.Name())
		}
	}

	for _, want := range []string{"new", "register"} {
		if !seen[want] {
			t.Fatalf("missing %q subcommand; got names: %v", want, keys(seen))
		}
	}
}

func TestCmdKeys_FindSubcommands(t *testing.T) {
	t.Parallel()

	c := cmdKeys()
	for _, path := range [][]string{{"new"}, {"register"}} {
		found, _, err := c.Find(path)
		if err != nil {
			t.Fatalf("Find(%v) error: %v", path, err)
		}
		if found == nil || found.Name() != path[0] {
			t.Fatalf("Find(%v) did not resolve expected command", path)
		}
		if found.Parent() != c {
			t.Fatalf("resolved command %q has wrong parent", found.Name())
		}
	}
}

func TestCmdKeys_HelpFlag(t *testing.T) {
	t.Parallel()

	c := cmdKeys()
	c.SilenceErrors = true
	c.SilenceUsage = true

	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"-h"})

	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() help error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Key management") || !strings.Contains(out, "Usage") {
		t.Fatalf("help output missing expected text; got:\n%s", out)
	}
}

// helper to show seen subcommand names in failure messages
func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
