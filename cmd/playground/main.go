package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/server"
	"github.com/TwigBush/gnap-go/internal/types"
)

func main() {
	store := mustStore()
	h := server.BuildPlaygroundRouter(server.Deps{GrantStore: store}, server.Options{
		DevNoStore: true,
	})
	log.Fatal(http.ListenAndServe(":8088", h))
}

func mustStore() types.GrantStore {
	s, err := gnap.NewFileStore(defaultDataDir(), types.Config{GrantTTLSeconds: 120})
	if err != nil {
		panic(err)
	}
	return s
}

func defaultDataDir() string {
	// Respect explicit override first
	if v := os.Getenv("TWIGBUSH_DATA_DIR"); v != "" {
		return v
	}
	// Resolve HOME cross-platform
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// Fallback to current working dir if HOME is unknown
		return filepath.Join(".", ".twigbush", "data")
	}
	return filepath.Join(home, ".twigbush", "data")
}
