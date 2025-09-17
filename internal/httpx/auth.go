package httpx

import "strings"

func ExtractGNAPToken(authz string) (string, bool) {
	const prefix = "GNAP "
	if authz == "" || !strings.HasPrefix(authz, prefix) {
		return "", false
	}
	return strings.TrimSpace(authz[len(prefix):]), true
}
