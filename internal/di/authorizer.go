package di

import (
	"os"

	"github.com/TwigBush/gnap-go/internal/authz"
)

func ProvideAuthorizer() authz.Authorizer {
	switch os.Getenv("TWIGBUSH_AUTHZ") {
	case "fga":
		cfg := authz.OpenFGAConfig{
			APIURL:   getenv("FGA_API_URL", "http://localhost:8080"),
			StoreID:  os.Getenv("FGA_STORE_ID"),
			APIToken: os.Getenv("FGA_API_TOKEN"),
			ModelID:  os.Getenv("FGA_MODEL_ID"),
		}
		a, err := authz.NewOpenFGA(cfg)
		if err != nil {
			panic(err)
		}
		return a
	case "mock":
		fallthrough
	default:
		return &authz.Mock{AlwaysAllow: true}
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
