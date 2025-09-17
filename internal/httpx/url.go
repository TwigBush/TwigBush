package httpx

import (
	"net"
	"net/http"
)

func BaseURL(r *http.Request) string {
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	// If Host is empty, fall back to server addr
	if host == "" {
		h, p, _ := net.SplitHostPort(r.URL.Host)
		if h == "" {
			h = "localhost"
		}
		if p == "" {
			p = "80"
		}
		host = net.JoinHostPort(h, p)
	}
	return scheme + "://" + host
}
