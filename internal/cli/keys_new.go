package cli

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
)

type jwk map[string]string

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func generateKey(dir, keyType string) (path string, thumb string, err error) {
	fmt.Printf("Generating %s key...\n", keyType)

	switch keyType {
	case "jwk":
		priv, x, y, err := genP256()
		if err != nil {
			return "", "", err
		}
		privJWK := jwk{
			"kty": "EC", "crv": "P-256", "x": b64u(x), "y": b64u(y), "d": b64u(priv),
		}
		pubJWK := jwk{"kty": "EC", "crv": "P-256", "x": b64u(x), "y": b64u(y)}
		tp, _ := jwkThumbprint(pubJWK)
		privPath := filepath.Join(dir, fmt.Sprintf("key-%s.jwk", tp))
		pubPath := filepath.Join(dir, fmt.Sprintf("key-%s.pub.jwk", tp))
		if err := writeJSONFile(privPath, privJWK, 0o600); err != nil {
			return "", "", err
		}
		if err := writeJSONFile(pubPath, pubJWK, 0o644); err != nil {
			return "", "", err
		}
		return privPath, tp, nil
	default:
		return "", "", fmt.Errorf("unknown key type: %s", keyType)
	}
}

func genP256() (d []byte, x []byte, y []byte, err error) {
	curve := elliptic.P256()
	priv, xBig, yBig, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	return priv, bigToBytes(xBig), bigToBytes(yBig), nil
}

func bigToBytes(b *big.Int) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b.Bytes()):], b.Bytes())
	return out
}

// RFC 7638 thumbprint for EC: {"crv","kty","x","y"}; for OKP: {"crv","kty","x"}
func jwkThumbprint(pub jwk) (string, error) {
	var canon []byte
	var err error
	if pub["kty"] == "EC" {
		canon, err = json.Marshal(map[string]string{
			"crv": pub["crv"], "kty": "EC", "x": pub["x"], "y": pub["y"],
		})
	} else {
		canon, err = json.Marshal(map[string]string{
			"crv": pub["crv"], "kty": "OKP", "x": pub["x"],
		})
	}
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return b64u(sum[:]), nil
}

func writeJSONFile(path string, v any, perm uint32) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, b, perm)
}

func writeFile(path string, b []byte, perm uint32) error {
	return osWriteFile(path, b, perm)
}
