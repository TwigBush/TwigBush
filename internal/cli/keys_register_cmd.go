package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func cmdKeysRegister() *cobra.Command {
	var asURL, rsID, adminToken, keyPath, tenant string

	c := &cobra.Command{
		Use:   "register",
		Short: "Register the RS public key with the AS",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := loadConfig(cfgPath)
			if cfg != nil && asURL == "" {
				asURL = cfg.ASBaseURL
			}
			if cfg != nil && keyPath == "" {
				keyPath = cfg.DefaultKey
			}

			if asURL == "" {
				return fmt.Errorf("--as is required")
			}
			if rsID == "" {
				return fmt.Errorf("--rs-id is required")
			}
			if keyPath == "" {
				return fmt.Errorf("--key is required or set default_key in config")
			}
			if tenant == "" {
				tenant = "default"
			}

			return registerKeyWithAS(keyPath, asURL, tenant, rsID, adminToken)
		},
	}

	c.Flags().StringVar(&asURL, "as", "", "AS base URL, for example http://localhost:8089")
	c.Flags().StringVar(&tenant, "tenant", "default", "Tenant ID (default: default)")
	c.Flags().StringVar(&rsID, "rs-id", "", "Resource server identifier, for example checkout")
	c.Flags().StringVar(&adminToken, "admin-token", "", "admin bearer token for AS")
	c.Flags().StringVar(&keyPath, "key", "", "path to private key .jwk whose .pub.jwk will be uploaded")
	return c
}
