package server

import (
	"encoding/json"
	"net/http"
)

// grantDiscoveryResp defines the GNAP AS discovery document.
type grantDiscoveryResp struct {
	GrantReqEndpoint                  string   `json:"grant_request_endpoint"`
	InteractionStartModesSupported    []string `json:"interaction_start_modes_supported,omitempty"`
	InteractionFinishMethodsSupported []string `json:"interaction_finish_methods_supported,omitempty"`
	KeyProofsSupported                []string `json:"key_proofs_supported,omitempty"`
	SubIDFormatsSupported             []string `json:"sub_id_formats_supported,omitempty"`
	AssertionFormatsSupported         []string `json:"assertion_formats_supported,omitempty"`
	KeyRotationSupported              *bool    `json:"key_rotation_supported,omitempty"`
}

// GrantDiscoveryHandler returns an OPTIONS handler configured with optional fields.
func GrantDiscoveryHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodOptions {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fullURL := buildAbsoluteURL(req)
		fullURL = ensureHTTPS(fullURL) // RFC 9635: MUST be https

		var keyRotation *bool
		if opts.KeyRotationSupported {
			keyRotation = &opts.KeyRotationSupported
		}

		response := &grantDiscoveryResp{
			GrantReqEndpoint:                  fullURL,
			InteractionStartModesSupported:    opts.InteractionStartModes,
			InteractionFinishMethodsSupported: opts.InteractionFinishMethods,
			KeyProofsSupported:                opts.KeyProofs,
			SubIDFormatsSupported:             opts.SubIDFormats,
			AssertionFormatsSupported:         opts.AssertionFormats,
			KeyRotationSupported:              keyRotation,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

// buildAbsoluteURL constructs a full URL using X-Forwarded-Proto or TLS detection.
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

	return scheme + "://" + host + r.URL.RequestURI()
}

// ensureHTTPS forces https scheme in URLs.
func ensureHTTPS(url string) string {
	if len(url) >= 7 && url[:7] == "https://" {
		return "https://" + url[7:]
	}
	return url
}
