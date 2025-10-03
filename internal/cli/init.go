package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func cmdInit() *cobra.Command {
	var fga string
	var keyType string

	c := &cobra.Command{
		Use:   "init",
		Short: "Create ~/.twigbush/config.yaml and a default key",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfg := &Config{
				ASBaseURL:   asBaseURL,
				RSBaseURL:   rsBaseURL,
				FGAEndpoint: fga,
			}
			// Make keys dir
			homeDir, err := configDir()
			if err != nil {
				return err
			}
			keysDir := filepath.Join(homeDir, "keys")
			if err := ensureDir(keysDir); err != nil {
				return err
			}
			// Generate a default key
			keyPath, thumb, err := generateKey(keysDir, keyType)
			if err != nil {
				return err
			}
			cfg.DefaultKey = keyPath
			// Save config
			if err := saveConfig(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Printf("Wrote config: %s\nDefault key: %s (thumbprint %s)\n", cfgPath, keyPath, thumb)
			return nil
		},
	}
	c.Flags().StringVar(&fga, "fga-endpoint", "", "OpenFGA endpoint URL (optional)")
	c.Flags().StringVar(&keyType, "key-type", "dpop", "default key type: dpop|httpsig|ed25519|p256")
	return c
}
