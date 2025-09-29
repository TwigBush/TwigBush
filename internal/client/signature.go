package client

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TwigBush/gnap-go/internal/types"
)

// SignatureGenerator handles HTTP signature generation for GNAP requests
type SignatureGenerator struct {
	privateKey types.JWK
}

// NewSignatureGenerator creates a new signature generator with the provided private key
func NewSignatureGenerator(privateKey types.JWK) *SignatureGenerator {
	return &SignatureGenerator{
		privateKey: privateKey,
	}
}

// SignRequest adds HTTP signature headers to the request
func (s *SignatureGenerator) SignRequest(req *http.Request, body []byte) error {
	// TODO: Fixme, see HTTP Signatures (RFC 9421)

	// Create signature input
	now := time.Now().Unix()
	signatureInput := fmt.Sprintf(`sig1=("@method" "@target-uri" "content-type");created=%d`, now)

	// Create signature base string
	signatureBase := s.createSignatureBase(req, body)

	// TODO: Fixme, see HTTP Signatures (RFC 9421)
	signature, err := s.signData(signatureBase)
	if err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	req.Header.Set("Signature", fmt.Sprintf("sig1=:%s:", signature))
	req.Header.Set("Signature-Input", signatureInput)

	return nil
}

func (s *SignatureGenerator) createSignatureBase(req *http.Request, body []byte) string {
	// TODO: Fixme, see HTTP Signatures (RFC 9421)
	components := []string{
		fmt.Sprintf("@method: %s", req.Method),
		fmt.Sprintf("@target-uri: %s", req.URL.String()),
		fmt.Sprintf("content-type: %s", req.Header.Get("Content-Type")),
	}

	if len(body) > 0 {
		h := sha256.New()
		h.Write(body)
		digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
		components = append(components, fmt.Sprintf("content-digest: sha-256=:%s:", digest))
		req.Header.Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", digest))
	}

	return strings.Join(components, "\n")
}

// signData performs the actual signing operation
func (s *SignatureGenerator) signData(data string) (string, error) {
	// TODO: Fixme, see HTTP Signatures (RFC 9421)
	h := sha256.New()
	h.Write([]byte(data))
	hash := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(hash), nil
}
