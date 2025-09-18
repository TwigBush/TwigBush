package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/token"
)

// AuthFlow handles GNAP authentication flows
type AuthFlow struct {
	config Configuration
	signer *SignatureGenerator
	client *http.Client
}

// NewAuthFlow creates a new AuthFlow instance
func NewAuthFlow(config Configuration) *AuthFlow {
	return &AuthFlow{
		config: config,
		signer: NewSignatureGenerator(config.KeyPair.PrivateKey),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// RequestGrant initiates a GNAP grant request
func (a *AuthFlow) RequestGrant(resources []gnap.AccessItem) (*gnap.GrantResponse, error) {
	// Prepare grant request
	grantRequest := gnap.GrantRequest{
		Client: gnap.Client{
			Key: gnap.ClientKey{
				Proof: string(a.config.ProofMethod.HTTPSig),
				JWK:   a.config.KeyPair.PublicKey,
			},
		},
		Access: resources,
		Interact: &gnap.Interact{
			Start: []string{"user_code"},
		},
	}

	url := fmt.Sprintf("%s/grants", a.config.AsURL)

	// Prepare request body
	body, err := json.Marshal(grantRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal grant request: %w", err)
	}
	fmt.Printf("%s", body)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add signature
	if err := a.signer.SignRequest(req, body); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Execute request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grant request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("grant request failed with status %d: %v", resp.StatusCode, errResp)
	}

	var grantResponse gnap.GrantResponse
	if err := json.NewDecoder(resp.Body).Decode(&grantResponse); err != nil {
		return nil, fmt.Errorf("failed to decode grant response: %w", err)
	}

	return &grantResponse, nil
}

// PollForToken polls for token approval
func (a *AuthFlow) PollForToken(ctx context.Context, grant *gnap.GrantResponse) (*token.Token, error) {
	if grant.Continue.URI == "" {
		return nil, errors.New("no continuation handle provided")
	}

	maxAttempts := 100
	attempts := 0
	waitSeconds := grant.Continue.Wait
	if waitSeconds == 0 {
		waitSeconds = 5
	}

	for attempts < maxAttempts {
		attempts++

		// wait before polling
		select {
		case <-time.After(time.Duration(waitSeconds) * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Create continuation request
		req, err := http.NewRequest("POST", grant.Continue.URI, bytes.NewBuffer([]byte("{}")))
		if err != nil {
			return nil, fmt.Errorf("failed to create continuation request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("GNAP %s", grant.Continue.AccessToken))
		req.Header.Set("Content-Type", "application/json")

		// Add signature
		if err := a.signer.SignRequest(req, []byte("{}")); err != nil {
			return nil, fmt.Errorf("failed to sign continuation request: %w", err)
		}

		// Execute request
		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("continuation request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Handle different response codes
		switch resp.StatusCode {
		case http.StatusOK:
			var continueResponse struct {
				AccessToken *token.Token       `json:"access_token,omitempty"`
				Continue    *gnap.Continue     `json:"continue,omitempty"`
				Error       *gnap.ErrorResponse `json:"error,omitempty"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&continueResponse); err != nil {
				return nil, fmt.Errorf("failed to decode continuation response: %w", err)
			}

			if continueResponse.AccessToken != nil {
				return continueResponse.AccessToken, nil
			}

			if continueResponse.Error != nil {
				return nil, fmt.Errorf("grant failed: %s", continueResponse.Error.Description)
			}

			// Update continuation if provided
			if continueResponse.Continue != nil {
				grant.Continue = *continueResponse.Continue
				waitSeconds = continueResponse.Continue.Wait
				if waitSeconds == 0 {
					waitSeconds = 5
				}
			}

		case http.StatusForbidden:
			return nil, errors.New("grant denied by user")

		case http.StatusBadRequest:
			var errResp struct {
				Error       string `json:"error"`
				Description string `json:"error_description"`
			}
			json.NewDecoder(resp.Body).Decode(&errResp)

			switch errResp.Error {
			case "expired_grant":
				return nil, errors.New("grant expired")
			case "too_fast":
				// Increase wait time
				waitSeconds = max(waitSeconds, 10)
				continue
			default:
				if errResp.Description != "" {
					return nil, fmt.Errorf("bad request: %s", errResp.Description)
				}
				return nil, fmt.Errorf("bad request: %s", string(bodyBytes))
			}

		default:
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}

	return nil, errors.New("polling timeout - user did not complete authorization")
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}