package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/token"
)

type GnapClient struct {
	auth   *AuthFlow
	signer *SignatureGenerator
	config Configuration
	client *http.Client
}

func NewGnapClient(config Configuration) *GnapClient {
	return &GnapClient{
		auth:   NewAuthFlow(config),
		signer: NewSignatureGenerator(config.KeyPair.PrivateKey),
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GnapClient) Authorize(ctx context.Context, resources []gnap.AccessItem, interactive bool) (*token.Token, error) {
	if !interactive {
		return nil, fmt.Errorf("no valid token and non-interactive mode")
	}

	// Request grant
	grant, err := c.auth.RequestGrant(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to request grant: %w", err)
	}

	// Display user code if available
	if grant.Interact.UserCode.Code != "" {
		fmt.Printf("To authorize, visit: %s\n", grant.Interact.UserCode.URI)
		fmt.Printf("Enter code: %s\n", grant.Interact.UserCode.Code)
	}

	// Poll for completion
	token, err := c.auth.PollForToken(ctx, grant)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// TODO: Store token for future use 
	return token, nil
}

// RequestOptions contains options for making authenticated requests
type RequestOptions struct {
	Method  string
	Headers map[string]string
	Body    interface{}
}

// MakeRequest makes an authenticated request with the provided token
func (c *GnapClient) MakeRequest(url string, token *token.Token, options *RequestOptions) (interface{}, error) {
	if options == nil {
		options = &RequestOptions{Method: "GET"}
	}
	if options.Method == "" {
		options.Method = "GET"
	}

	// Prepare request body
	var bodyReader io.Reader
	var bodyBytes []byte
	if options.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(options.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequest(options.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Value))
	if options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	// Add signature if key-bound token
	if token.Key.JWK.Kty != "" {
		if err := c.signer.SignRequest(req, bodyBytes); err != nil {
			return nil, fmt.Errorf("failed to sign request: %w", err)
		}
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If JSON parsing fails, return raw body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return string(body), nil
	}

	return result, nil
}

// MakeRequestWithContext makes an authenticated request with context
func (c *GnapClient) MakeRequestWithContext(ctx context.Context, url string, token *token.Token, options *RequestOptions) (interface{}, error) {
	if options == nil {
		options = &RequestOptions{Method: "GET"}
	}
	if options.Method == "" {
		options.Method = "GET"
	}

	// Prepare request body
	var bodyReader io.Reader
	var bodyBytes []byte
	if options.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(options.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, options.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Value))
	if options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	// Add signature if key-bound token
	if token.Key.JWK.Kty != "" {
		if err := c.signer.SignRequest(req, bodyBytes); err != nil {
			return nil, fmt.Errorf("failed to sign request: %w", err)
		}
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If JSON parsing fails, return raw body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return string(body), nil
	}

	return result, nil
}