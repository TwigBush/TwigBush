package handlers

import (
	"log"
	"net/http"
)

func Introspect(w http.ResponseWriter, r *http.Request) {
	//TODO: verify request proof (detached JWS or DPoP)
	// extract the token from the request
	log.Print("Introspect called")
	w.WriteHeader(http.StatusNotImplemented)
}
