package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOsWriteFile_DefaultWritesWithPerm(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello.txt")

	data := []byte("hi")
	if err := osWriteFile(p, data, 0o600); err != nil {
		t.Fatalf("osWriteFile error: %v", err)
	}

	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "hi" {
		t.Fatalf("content = %q, want %q", string(got), "hi")
	}

	// Windows does not reliably enforce POSIX perms
	if runtime.GOOS != "windows" {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if st.Mode().Perm() != 0o600 {
			t.Fatalf("perm = %v, want 0600", st.Mode().Perm())
		}
	}
}

func TestOsWriteFile_CanBeOverridden(t *testing.T) {
	orig := osWriteFile
	t.Cleanup(func() { osWriteFile = orig })

	called := false
	var gotPath string
	var gotPerm uint32
	var gotData []byte

	osWriteFile = func(path string, b []byte, perm uint32) error {
		called = true
		gotPath = path
		gotPerm = perm
		gotData = append([]byte(nil), b...) // copy
		return nil
	}

	if err := writeFile("x/y/z.txt", []byte("stubbed"), 0o777); err != nil {
		t.Fatalf("writeFile error: %v", err)
	}

	if !called {
		t.Fatalf("seam was not invoked")
	}
	if gotPath != "x/y/z.txt" {
		t.Fatalf("path = %q, want %q", gotPath, "x/y/z.txt")
	}
	if gotPerm != 0o777 {
		t.Fatalf("perm = %o, want %o", gotPerm, 0o777)
	}
	if string(gotData) != "stubbed" {
		t.Fatalf("data = %q, want %q", string(gotData), "stubbed")
	}
}
