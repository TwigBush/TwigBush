package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Run() error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, middleware.Timeout(15*time.Second))

	registerRoutes(r)

	srv := &http.Server{Addr: ":8089", Handler: r}
	log.Print("listening on port ::: " + srv.Addr)
	return srv.ListenAndServe()
}
