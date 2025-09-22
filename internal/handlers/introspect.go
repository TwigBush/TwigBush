package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TwigBush/gnap-go/internal/httpx"
)


func Introspect(w http.ResponseWriter, r *http.Request) {
	//TODO: verify request proof 
	// extract the token from the request
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}
	// check if JWT
	claims, err := isJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// access "exp" on claims
	exp := claims.(map[string]interface{})["exp"]
	// get current time
	now := (float64)(time.Now().Unix())
	if exp.(float64) < now {
		http.Error(w, "Token expired", http.StatusUnauthorized)
		return
	}
	// return claims as JSON
	httpx.WriteJSON(w, http.StatusOK, claims)
}

func isJWT(token string) (interface{}, error) {

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// grab payload
	payload := parts[1]
	// decode
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}
	// parse claims
	// TODO: we should have a claims struct instead of interface{}
	// claims := map[string]any{
	//  "iss":    cfg.Issuer,
	//	"sub":    subjectOrAnon(grant.Subject),
	//	"aud":    cfg.Audience,
	//	"exp":    exp,
	//	"iat":    iat,
	//	"jti":    jti,
	//	"access": grant.ApprovedAccess,
	//	"key":    grant.Client.Key,
	//}
	var claims interface{}
	if err := json.Unmarshal(b, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}
	return claims, nil
}