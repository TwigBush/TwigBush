package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func cmdKeysNew() *cobra.Command {
	var keyType string

	c := &cobra.Command{
		Use:   "new",
		Short: "Generate a new key as JWK and print its thumbprint",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := configDir()
			if err != nil {
				return err
			}
			keysDir := filepath.Join(dir, "keys")
			if err := ensureDir(keysDir); err != nil {
				return err
			}
			path, tp, err := generateKey(keysDir, keyType)
			if err != nil {
				return err
			}
			fmt.Printf("Wrote %s\nThumbprint: %s\n", path, tp)

			// Optionally update default key in config if empty
			cfg, err := loadConfig(cfgPath)
			if err == nil && cfg.DefaultKey == "" {
				cfg.DefaultKey = path
				_ = saveConfig(cfgPath, cfg)
			}
			return nil
		},
	}
	c.Flags().StringVar(&keyType, "type", "jwk", "dpop|httpsig|ed25519|p256")
	return c
}

// small wrapper to allow custom perms without fs import in other file
func osWriteFile(path string, b []byte, perm uint32) error {
	return os.WriteFile(path, b, os.FileMode(perm))
}
