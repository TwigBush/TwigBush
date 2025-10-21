package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

func registerKeyWithAS(privPath, asURL, tenant, rsID, adminToken string) error {
	pubPath := strings.TrimSuffix(privPath, ".jwk") + ".pub.jwk"
	if _, err := os.Stat(pubPath); err != nil {
		// fallback if user pointed at .pub.jwk directly
		if strings.HasSuffix(privPath, ".pub.jwk") {
			pubPath = privPath
		} else {
			return fmt.Errorf("public key not found: %s", pubPath)
		}
	}

	pubJSON, err := os.ReadFile(pubPath)
	if err != nil {
		return fmt.Errorf("read pub jwk: %w", err)
	}

	// Parse the JWK to extract kid
	parsedKey, err := jwk.ParseKey(pubJSON)
	if err != nil {
		return fmt.Errorf("parse public JWK: %w", err)
	}

	kid, ok := parsedKey.KeyID()
	if !ok || kid == "" {
		return fmt.Errorf("public key missing 'kid' field")
	}

	var pub any
	if err := json.Unmarshal(pubJSON, &pub); err != nil {
		return fmt.Errorf("invalid public JWK: %w", err)
	}

	body := map[string]any{
		"jwk":        pub,
		"alg":        "ES384", // todo: make this configurable
		"display_rs": rsID,
		"kid":        kid,
	}
	b, _ := json.Marshal(body)

	url := strings.TrimRight(asURL, "/") + "/admin/tenants/" + tenant + "/rs/keys"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+adminToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AS returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	fmt.Printf("✓ Key registered successfully for tenant '%s'\n", tenant)
	fmt.Printf("Response: %s\n", string(respBody))
	return nil
}
