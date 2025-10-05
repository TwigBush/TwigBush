package handlers

import (
	"log"
	"net/http"
)

func Version(w http.ResponseWriter, r *http.Request) {

	// issue short TTL key-bound token or start interact flow
	log.Print("Version called")
	w.WriteHeader(http.StatusNotImplemented)
}
