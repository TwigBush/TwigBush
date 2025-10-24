package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ASBaseURL   string `yaml:"as_base_url"   mapstructure:"as_base_url"`
	RSBaseURL   string `yaml:"rs_base_url"   mapstructure:"rs_base_url"`
	FGAEndpoint string `yaml:"fga_endpoint"  mapstructure:"fga_endpoint"`
	DefaultKey  string `yaml:"default_key"   mapstructure:"default_key"` // path to JWK
}

func ensureDir(p string) error { return os.MkdirAll(p, 0o755) }

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".twigbush"), nil
}

func loadConfig(path string) (*Config, error) {
	if path == "" {
		dir, err := configDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "config.yaml")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("as_base_url", "http://localhost:8085")
	v.SetDefault("rs_base_url", "http://localhost:8088")
	v.SetDefault("fga_endpoint", "")
	v.SetDefault("default_key", "")

	// Env overrides: TWIGBUSH_AS_BASE_URL, TWIGBUSH_RS_BASE_URL, etc.
	v.SetEnvPrefix("TWIGBUSH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Read file if it exists, otherwise return defaults without error
	if err := v.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if !errors.As(err, &nf) {
			return nil, err
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveConfig(path string, c *Config) error {
	if path == "" {
		dir, err := configDir()
		if err != nil {
			return err
		}
		path = filepath.Join(dir, "config.yaml")
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("as_base_url", c.ASBaseURL)
	v.Set("rs_base_url", c.RSBaseURL)
	v.Set("fga_endpoint", c.FGAEndpoint)
	v.Set("default_key", c.DefaultKey)

	// Write or create
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := v.WriteConfigAs(path); err != nil {
			return err
		}
	} else {
		if err := v.WriteConfigAs(path); err != nil {
			return err
		}
	}

	// Restrict perms to owner
	_ = os.Chmod(path, 0o600)
	return nil
}
