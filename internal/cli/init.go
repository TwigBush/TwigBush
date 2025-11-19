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

			// Create AS data directory structure for registry
			// This is where AS stores registered RS public keys
			asDataDir := filepath.Join(homeDir, "data")
			if err := ensureDir(asDataDir); err != nil {
				return fmt.Errorf("failed to create AS data directory: %w", err)
			}

			rsKeyRegistryDir := filepath.Join(asDataDir, "rs_keys")
			if err := ensureDir(rsKeyRegistryDir); err != nil {
				return fmt.Errorf("failed to create RS key registry: %w", err)
			}

			grantsDir := filepath.Join(asDataDir, "grants")
			if err := ensureDir(grantsDir); err != nil {
				return fmt.Errorf("failed to create grants directory: %w", err)
			}

			tokens := filepath.Join(asDataDir, "tokens")
			if err := ensureDir(tokens); err != nil {
				return fmt.Errorf("failed to create tokens directory: %w", err)
			}

			// Create default tenant directory
			defaultTenantDir := filepath.Join(rsKeyRegistryDir, "default")
			if err := ensureDir(defaultTenantDir); err != nil {
				return fmt.Errorf("failed to create default tenant directory: %w", err)
			}
			// Generate a default key
			keyPath, thumb, err := generateKey(keysDir, "")
			if err != nil {
				return err
			}
			cfg.DefaultKey = keyPath
			// Save config
			if err := saveConfig(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Printf("Initialized TwigBush directories:\n")
			fmt.Printf("  Config: %s\n", cfgPath)
			fmt.Printf("  Client keys: %s\n", keysDir)
			fmt.Printf("  AS registry: %s\n", rsKeyRegistryDir)
			fmt.Printf("\nGenerated default key: %s\n", keyPath)
			fmt.Printf("Thumbprint: %s\n", thumb)
			return nil
		},
	}
	c.Flags().StringVar(&fga, "fga-endpoint", "", "OpenFGA endpoint URL (optional)")
	c.Flags().StringVar(&keyType, "key-type", "jwk", "default key type: jwk")
	return c
}
