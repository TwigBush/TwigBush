package handlers

import (
	"log"
	"net/http"
)

func Continue(w http.ResponseWriter, r *http.Request) {
	// verify request proof (detached JWS or DPoP)
	// decode GNAP grant request
	// call policy.Check(...)
	// issue short TTL key-bound token or start interact flow
	log.Print("Continue called")
	w.WriteHeader(http.StatusNotImplemented)
}
