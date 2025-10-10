package cli

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/spf13/cobra"
)

func cmdSignCurl() *cobra.Command {
	var (
		keyPath  string
		method   string
		rawURL   string
		httpsig  bool
		bodyPath string
		tenant   string
		algFlag  string // optional override: ed25519 | ecdsa-p256-sha256 | ecdsa-p384-sha384
	)
	c := &cobra.Command{
		Use:   "curl",
		Short: "Wrap a curl with HTTP Message Signatures",
		Example: "twigbush sign curl --httpsig --key ~/.twigbush/keys/key-XYZ.jwk " +
			"--method POST --url http://localhost:8089/introspect --body ./body.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !httpsig {
				return fmt.Errorf("use --httpsig")
			}
			if keyPath == "" || rawURL == "" {
				return fmt.Errorf("--key and --url are required")
			}

			// Load JWK twice: once as raw private key for signing, once to read kid
			privJWK, err := os.ReadFile(keyPath)
			if err != nil {
				return fmt.Errorf("read key: %w", err)
			}

			var priv any
			if err := jwk.ParseRawKey(privJWK, &priv); err != nil {
				return fmt.Errorf("parse raw jwk: %w", err)
			}
			k, err := jwk.ParseKey(privJWK)
			if err != nil {
				return fmt.Errorf("parse jwk: %w", err)
			}
			kidStr, ok := k.KeyID()
			if !ok {
				return fmt.Errorf("key missing kid")
			}

			alg := strings.ToLower(strings.TrimSpace(algFlag))
			if alg == "" {
				switch pk := priv.(type) {
				case ed25519.PrivateKey:
					alg = "ed25519"
				case *ecdsa.PrivateKey:
					switch pk.Curve {
					case elliptic.P256():
						alg = "ecdsa-p256-sha256"
					case elliptic.P384():
						alg = "ecdsa-p384-sha384"
					default:
						return fmt.Errorf("unsupported ECDSA curve")
					}
				default:
					return fmt.Errorf("unsupported key type %T", pk)
				}
			}

			u, err := url.Parse(rawURL)
			if err != nil {
				return err
			}

			var body []byte
			if bodyPath != "" {
				body, err = os.ReadFile(bodyPath)
				if err != nil {
					return fmt.Errorf("read body: %w", err)
				}
			}

			// Optional Content-Digest (recommended when body is present) – GNAP verifier should check it. :contentReference[oaicite:2]{index=2}
			headers := map[string]string{}
			comps := []string{"@method", "@target-uri"}
			if len(body) > 0 {
				d := sha256.Sum256(body)
				headers["Content-Digest"] = "sha-256=:" + base64.StdEncoding.EncodeToString(d[:]) + ":"
				comps = append(comps, "content-digest")
			}
			if tenant != "" {
				headers["X-Tenant-ID"] = tenant
			}

			created := fmt.Sprintf("%d", time.Now().Unix())
			sigInput := buildSignatureInput("sig1", comps, map[string]string{
				"created": created,
				"keyid":   kidStr,
				"alg":     alg,
			})
			headers["Signature-Input"] = sigInput

			base, err := buildSignatureBase(u, strings.ToUpper(method), headers, comps, map[string]string{
				"created": created, "keyid": kidStr, "alg": alg,
			})
			if err != nil {
				return fmt.Errorf("base: %w", err)
			}

			sig, err := signBase(priv, alg, base)
			if err != nil {
				return err
			}
			headers["Signature"] = "sig1=:" + base64.StdEncoding.EncodeToString(sig) + ":"

			// Print curl command
			fmt.Println(curlForCLI(strings.ToUpper(method), rawURL, bodyPath, headers))
			return nil
		},
	}
	c.Flags().StringVar(&keyPath, "key", "", "path to private JWK")
	_ = c.MarkFlagRequired("key")
	c.Flags().StringVar(&method, "method", "POST", "HTTP method")
	c.Flags().StringVar(&rawURL, "url", "", "target URL")
	_ = c.MarkFlagRequired("url")
	c.Flags().BoolVar(&httpsig, "httpsig", true, "use HTTP Message Signatures")
	c.Flags().StringVar(&bodyPath, "body", "", "path to request body (optional)")
	c.Flags().StringVar(&tenant, "tenant", "default", "X-Tenant-ID header (optional)")
	c.Flags().StringVar(&algFlag, "alg", "", "override alg (ed25519|ecdsa-p256-sha256|ecdsa-p384-sha384)")
	return c
}

func buildSignatureInput(label string, comps []string, params map[string]string) string {
	var b strings.Builder
	b.WriteString(label)
	b.WriteString("=(")
	for i, c := range comps {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%q", c)
	}
	b.WriteString(")")
	if v := params["created"]; v != "" {
		fmt.Fprintf(&b, ";created=%s", v)
	}
	if v := params["keyid"]; v != "" {
		fmt.Fprintf(&b, ";keyid=%q", v)
	}
	if v := params["alg"]; v != "" {
		fmt.Fprintf(&b, ";alg=%q", strings.ToLower(v))
	}
	return b.String()
}

// Mirrors your verifier’s base construction (derived components + covered headers)
func buildSignatureBase(u *url.URL, method string, hdr map[string]string, comps []string, params map[string]string) ([]byte, error) {
	var b strings.Builder
	for _, c := range comps {
		lc := strings.ToLower(c)
		switch lc {
		case "@method":
			fmt.Fprintf(&b, "\"@method\": %s\n", strings.ToLower(method))
		case "@target-uri":
			scheme := u.Scheme
			if scheme == "" {
				scheme = "http"
			}
			host := u.Host
			fmt.Fprintf(&b, "\"@target-uri\": %s://%s%s\n", scheme, host, u.RequestURI())
		default:
			v, ok := hdr[textprotoCanonical(lc)]
			if !ok {
				return nil, fmt.Errorf("missing covered header %q", lc)
			}
			fmt.Fprintf(&b, "\"%s\": %s\n", lc, v)
		}
	}
	// @signature-params line
	fmt.Fprintf(&b, "\"@signature-params\": (")
	for i, c := range comps {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%q", c)
	}
	b.WriteString(")")
	if v := params["created"]; v != "" {
		fmt.Fprintf(&b, ";created=%s", v)
	}
	if v := params["keyid"]; v != "" {
		fmt.Fprintf(&b, ";keyid=%q", v)
	}
	if v := params["alg"]; v != "" {
		fmt.Fprintf(&b, ";alg=%q", strings.ToLower(v))
	}
	b.WriteByte('\n')
	return []byte(b.String()), nil
}

func signBase(priv any, alg string, base []byte) ([]byte, error) {
	switch strings.ToLower(alg) {
	case "ed25519":
		pk, ok := priv.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key not ed25519")
		}
		return ed25519.Sign(pk, base), nil
	case "ecdsa-p256-sha256":
		pk, ok := priv.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key not ecdsa p256")
		}
		sum := sha256.Sum256(base)
		return ecdsa.SignASN1(nil, pk, sum[:])
	case "ecdsa-p384-sha384":
		pk, ok := priv.(*ecdsa.PrivateKey)
		if !ok || pk.Curve != elliptic.P384() {
			return nil, fmt.Errorf("key not ecdsa p384")
		}
		sum := sha512.Sum384(base)
		return ecdsa.SignASN1(nil, pk, sum[:])
	default:
		return nil, fmt.Errorf("unsupported alg %q", alg)
	}
}

func curlForCLI(method, rawURL, bodyPath string, headers map[string]string) string {
	var b strings.Builder
	b.WriteString("curl -sS -X ")
	b.WriteString(method)
	b.WriteString(" ")
	b.WriteString(fmt.Sprintf("%q", rawURL))
	for k, v := range headers {
		b.WriteString(" -H ")
		b.WriteString(fmt.Sprintf("%q", fmt.Sprintf("%s: %s", k, v)))
	}
	if bodyPath != "" {
		b.WriteString(" --data-binary @")
		b.WriteString(fmt.Sprintf("%q", bodyPath))
		b.WriteString(" -H \"Content-Type: application/json\"")
	}
	return b.String()
}

// Minimal canonicalization for case-insensitive header map
func textprotoCanonical(s string) string {
	return http.CanonicalHeaderKey(s) // import "net/http"
}
