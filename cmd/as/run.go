package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"time"
)

func Run() error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	registerRoutes(r) // lives in routes.go

	srv := &http.Server{
		Addr:    ":8085",
		Handler: r,
	}
	log.Print("listening on port ::: " + srv.Addr)
	return srv.ListenAndServe()
}
