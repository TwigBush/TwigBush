package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TwigBush/gnap-go/internal/httpx"
	"github.com/TwigBush/gnap-go/internal/types"
)

func Introspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.IntrospectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{
			Error:       "invalid_request",
			Description: "Invalid request format",
		})
		return
	}

	if req.AccessToken == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{
			Error:       "invalid_request",
			Description: "access_token is required",
		})
		return
	}

	if req.ResourceServer == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{
			Error:       "invalid_request",
			Description: "resource_server is required",
		})
		return
	}

	// TODO: Verify RS signature (RS signs request with its own key)

	resp := validateToken(req.AccessToken, req.Proof, req.ResourceServer, req.Access)
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func validateToken(accessToken string, proof string, resourceServer interface{}, requestedAccess []string) types.IntrospectionResponse {
	// default to inactive
	resp := types.IntrospectionResponse{
		Active: false,
	}

	// parse as JWT
	claims, err := parseJWT(accessToken)
	if err != nil {
		// token is not a JWT
		return resp
	}

	// Check expiration
	now := time.Now().Unix()
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < now {
			// Token is expired
			return resp
		}
		expInt := int64(exp)
		resp.Exp = &expInt
	}

	// Check not before
	if nbf, ok := claims["nbf"].(float64); ok {
		if int64(nbf) > now {
			// Token is not yet valid
			return resp
		}
		nbfInt := int64(nbf)
		resp.Nbf = &nbfInt
	}

	// TODO: Check if token was issued by this AS
	// TODO: Check if token has been revoked
	// TODO: Verify proof method matches
	// TODO: Verify appropriate for this RS
	// TODO: Verify access rights if requested

	// If all checks pass, token is active
	resp.Active = true

	// Populate response fields from claims
	if iss, ok := claims["iss"].(string); ok {
		resp.Iss = iss
	}

	if sub, ok := claims["sub"].(string); ok {
		resp.Sub = sub
	}

	if aud := claims["aud"]; aud != nil {
		resp.Aud = aud
	}

	if iat, ok := claims["iat"].(float64); ok {
		iatInt := int64(iat)
		resp.Iat = &iatInt
	}

	if access, ok := claims["access"]; ok {
		if accessArray, isArray := access.([]interface{}); isArray {
			resp.Access = accessArray
		}
	}

	// Add key information if token is bound
	if key, ok := claims["key"]; ok && key != nil {
		if keyMap, ok := key.(map[string]interface{}); ok {
			if proofStr, ok := keyMap["proof"].(string); ok {
				if jwkMap, ok := keyMap["jwk"].(map[string]interface{}); ok {
					jwk := types.JWK{}
					if kty, ok := jwkMap["kty"].(string); ok {
						jwk.Kty = kty
					}
					if crv, ok := jwkMap["crv"].(string); ok {
						jwk.Crv = crv
					}
					if x, ok := jwkMap["x"].(string); ok {
						jwk.X = x
					}
					if y, ok := jwkMap["y"].(string); ok {
						jwk.Y = y
					}
					resp.Key = &types.ClientKey{
						Proof: proofStr,
						JWK:   jwk,
					}
				}
			}
		}
	}

	return resp
}

func parseJWT(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload := parts[1]
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(b, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	return claims, nil
}