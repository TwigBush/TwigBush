package handlers

import (
	"github.com/TwigBush/gnap-go/internal/jwks"
	"log"
	"net/http"
)

func JWKSOriginal(w http.ResponseWriter, r *http.Request) {

	// issue short TTL key-bound token or start interact flow
	log.Print("JWKS called")
	w.WriteHeader(http.StatusNotImplemented)
}

func JWKS(w http.ResponseWriter, r *http.Request) {
	jwks.Serve(w, r)
}
