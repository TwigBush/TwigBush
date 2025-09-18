package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/TwigBush/gnap-go/internal/client"
	"github.com/TwigBush/gnap-go/internal/gnap"
)

func main() {
	// Generate EC P-256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate EC key pair: %v", err)
	}

	// Extract public key coordinates and encode to base64url
	publicKey := privateKey.PublicKey
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Ensure coordinates are 32 bytes (P-256 uses 256-bit/32-byte coordinates)
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	xBase64 := base64.RawURLEncoding.EncodeToString(xPadded)
	yBase64 := base64.RawURLEncoding.EncodeToString(yPadded)

	config := client.Configuration{
		ClientID:      "example-client",
		ClientName:    "GNAP Go Client",
		ClientVersion: "1.0.0",
		ClientURI:     "https://example.com/client",
		KeyPair: client.KeyPair{
			PrivateKey: gnap.JWK{
				Kty: "EC",
				Crv: "P-256",
				X:   xBase64,
				Y:   yBase64,
			},
			PublicKey: gnap.JWK{
				Kty: "EC",
				Crv: "P-256",
				X:   xBase64,
				Y:   yBase64,
			},
		},
		ProofMethod: client.ProofMethod{
			HTTPSig: client.HTTPSig,
		},
		AsURL: getEnvOrDefault("GNAP_AS_URL", "http://localhost:8085"),
	}

	// Create GNAP client
	gnapClient := client.NewGnapClient(config)

	// Define resources to request access to
	resources := []gnap.AccessItem{
		{
			Type: "photo-api",
			Actions: []string{"read", "write"},
			Locations: []string{"https://server.example.net/"},
		},
		{
			Type: "financial",
			Actions: []string{"read"},
			Locations: []string{"https://backend.example.net/"},
		},
	}

	ctx := context.Background()

	fmt.Println("Starting GNAP authorization flow...")
	token, err := gnapClient.Authorize(ctx, resources, true)
	if err != nil {
		log.Fatalf("Authorization failed: %v", err)
	}

	fmt.Printf("Authorization successful! Token ID: %s\n", token.TokenID)
	fmt.Printf("Token format: %s\n", token.Format)
	fmt.Printf("Expires in: %d seconds\n", token.ExpiresIn)

	fmt.Println("\nMaking authenticated request...")
	response, err := gnapClient.MakeRequestWithContext(
		ctx,
		"https://server.example.net/photos",
		token,
		&client.RequestOptions{
			Method: "GET",
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
	)
	if err != nil {
		log.Printf("Request failed: %v", err)
	} else {
		fmt.Printf("Response: %+v\n", response)
	}

	// Example POST request with body
	fmt.Println("\nMaking authenticated POST request...")
	postData := map[string]interface{}{
		"name":        "My Photo",
		"description": "A test photo upload",
	}

	postResponse, err := gnapClient.MakeRequestWithContext(
		ctx,
		"https://server.example.net/photos",
		token,
		&client.RequestOptions{
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			Body: postData,
		},
	)
	if err != nil {
		log.Printf("POST request failed: %v", err)
	} else {
		fmt.Printf("POST Response: %+v\n", postResponse)
	}
}

// generateECKeyPair generates a P-256 EC key pair and returns it as JWK
func generateECKeyPair() (*ecdsa.PrivateKey, gnap.JWK, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, gnap.JWK{}, err
	}

	// Extract public key coordinates
	xBytes := privateKey.PublicKey.X.Bytes()
	yBytes := privateKey.PublicKey.Y.Bytes()

	// Ensure coordinates are 32 bytes (P-256 uses 256-bit/32-byte coordinates)
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	jwk := gnap.JWK{
		Kty: "EC",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(xPadded),
		Y:   base64.RawURLEncoding.EncodeToString(yPadded),
	}

	return privateKey, jwk, nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
