package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Run() error {
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, middleware.Timeout(15*time.Second))

	registerRoutes(router)

	srv := &http.Server{Addr: ":8089", Handler: router}
	log.Print("listening on port ::: " + srv.Addr)
	return srv.ListenAndServe()
}
