package server

import (
	"encoding/json"
	"net/http"
)

type grantDiscoveryResp struct {
	GrantReqEndpoint                  string   `json:"grant_request_endpoint"`
	InteractionStartModesSupported    []string `json:"interaction_start_modes_supported,omitempty"`
	InteractionFinishMethodsSupported []string `json:"interaction_finish_methods_supported,omitempty"`
	KeyProofsSupported                []string `json:"key_proofs_supported,omitempty"`
	SubIDFormatsSupported             []string `json:"sub_id_formats_supported,omitempty"`
	AssertionFormatsSupported         []string `json:"assertion_formats_supported,omitempty"`
	KeyRotationSupported              *bool    `json:"key_rotation_supported,omitempty"`
}

func GrantDiscoveryHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodOptions {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fullURL := buildAbsoluteURL(req)

	response := &grantDiscoveryResp{
		GrantReqEndpoint: fullURL,
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func buildAbsoluteURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	finalURL := scheme + "://" + host + r.URL.RequestURI()

	return finalURL
}
