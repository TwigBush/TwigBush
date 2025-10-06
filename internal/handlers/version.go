package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/TwigBush/gnap-go/internal/version"
)

// VersionHandler returns version information as JSON
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", version.Version)

	if err := json.NewEncoder(w).Encode(version.Get()); err != nil {
		http.Error(w, "Failed to encode version", http.StatusInternalServerError)
		return
	}
}
