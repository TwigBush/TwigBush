package handlers

import (
	"github.com/TwigBush/gnap-go/internal/jwks"
	"log"
	"net/http"
)

func JWKSOriginal(w http.ResponseWriter, r *http.Request) {
	// verify request proof (detached JWS or DPoP)
	// decode GNAP grant request
	// call policy.Check(...)
	// issue short TTL key-bound token or start interact flow
	log.Print("JWKS called")
	w.WriteHeader(http.StatusNotImplemented)
}

func JWKS(w http.ResponseWriter, r *http.Request) {
	jwks.Serve(w, r)
}
