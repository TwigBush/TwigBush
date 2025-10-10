package cli

import "os"

var osWriteFile = func(path string, b []byte, perm uint32) error {
	return os.WriteFile(path, b, os.FileMode(perm))
}
