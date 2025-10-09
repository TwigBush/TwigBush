package cli

import (
	"fmt"

	"path/filepath"

	"github.com/spf13/cobra"
)

func cmdKeysNew() *cobra.Command {
	var keyType, asURL, rsID, adminToken, tenant string
	var doRegister bool

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

			// Optional immediate registration
			if doRegister {
				if cfg != nil && asURL == "" {
					asURL = cfg.ASBaseURL
				}
				if asURL == "" {
					return fmt.Errorf("--as is required to register")
				}
				if rsID == "" {
					return fmt.Errorf("--rs-id is required to register")
				}
				if tenant == "" {
					tenant = "default"
				}
				if err := registerKeyWithAS(path, asURL, tenant, rsID, adminToken); err != nil {
					return err
				}
				fmt.Println("Registered public key with AS")
			}
			return nil
		},
	}
	c.Flags().StringVar(&keyType, "type", "jwk", "key type: jwk")
	c.Flags().BoolVar(&doRegister, "register", false, "register the new public key with the AS")
	c.Flags().StringVar(&asURL, "as", "", "AS base URL, for example http://localhost:8089")
	c.Flags().StringVar(&tenant, "tenant", "default", "Tenant ID (default: default)")
	c.Flags().StringVar(&rsID, "rs-id", "", "Resource server identifier, for example checkout")
	c.Flags().StringVar(&adminToken, "admin-token", "", "admin bearer token for AS")
	return c
}
