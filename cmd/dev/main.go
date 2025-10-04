package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/server"
	"github.com/TwigBush/gnap-go/internal/types"
)

func main() {
	asAddr := flag.String("as", ":8085", "AS addr")
	uiAddr := flag.String("ui", ":8088", "Playground addr")
	storeKind := flag.String("store", "fs", "fs")
	flag.Parse()

	store := mustStore(*storeKind)

	as := server.BuildASRouter(server.Deps{Store: store}, server.Options{})
	ui := server.BuildPlaygroundRouter(server.Deps{Store: store}, server.Options{
		DevNoStore: true,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var g errgroup.Group
	g.Go(func() error { return run(ctx, *asAddr, as, "AS") })
	g.Go(func() error { return run(ctx, *uiAddr, ui, "Playground") })

	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func run(ctx context.Context, addr string, h http.Handler, name string) error {
	srv := &http.Server{Addr: addr, Handler: h}
	errc := make(chan error, 1)
	go func() {
		log.Printf("%s on %s", name, addr)
		errc <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx2)
	case err := <-errc:
		return err
	}
}

func mustStore(kind string) types.Store {
	cfg := types.Config{GrantTTLSeconds: 120}
	switch kind {
	case "fs":
		s, err := gnap.NewFileStore(defaultDataDir(), cfg)
		if err != nil {
			panic(err)
		}
		return s
	default:
		panic("unknown store")
	}
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
