package mw

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

type RSKeyResolver func(r *http.Request, params map[string]string) (crypto.PublicKey, error)

type rsCfg struct {
	resolve        RSKeyResolver
	requireTLS     bool
	requiredComps  []string // components you insist must be covered
	allowedAlgs    map[string]struct{}
	maxSkewSeconds int64 // for `created` param
}

type RSOption func(*rsCfg)

func WithRSKeyResolver(fn RSKeyResolver) RSOption  { return func(c *rsCfg) { c.resolve = fn } }
func WithRSRequiredComponents(v []string) RSOption { return func(c *rsCfg) { c.requiredComps = v } }
func WithRSAllowedAlgs(algs ...string) RSOption {
	return func(c *rsCfg) { c.allowedAlgs = toSet(algs) }
}
func WithRSRequireTLS(v bool) RSOption      { return func(c *rsCfg) { c.requireTLS = v } }
func WithRSMaxSkewSeconds(s int64) RSOption { return func(c *rsCfg) { c.maxSkewSeconds = s } }
func toSet(xs []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, x := range xs {
		m[strings.ToLower(x)] = struct{}{}
	}
	return m
}

// VerifyRSProof validates an RS caller using HTTP Message Signatures (RFC 9421).
// It expects headers: Signature-Input and Signature with label `sig1`.
func VerifyRSProof(opts ...RSOption) func(http.Handler) http.Handler {
	cfg := &rsCfg{
		requireTLS:     true,
		requiredComps:  []string{"@method", "@target-uri"},
		allowedAlgs:    toSet([]string{"ed25519", "ecdsa-p256-sha256"}),
		maxSkewSeconds: 300,
	}
	for _, o := range opts {
		o(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.requireTLS && r.TLS == nil {
				http.Error(w, "TLS required", http.StatusUpgradeRequired)
				return
			}
			sigInput := r.Header.Get("Signature-Input")
			sig := r.Header.Get("Signature")
			if sigInput == "" || sig == "" {
				http.Error(w, "missing HTTP Signature headers", http.StatusUnauthorized)
				return
			}

			entry, err := parseSignatureInputForLabel(sigInput, "sig1")
			if err != nil {
				http.Error(w, "invalid Signature-Input", http.StatusUnauthorized)
				return
			}
			if err := ensureRequired(entry.components, cfg.requiredComps); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err := checkCreated(entry.params, cfg.maxSkewSeconds); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			alg := strings.ToLower(entry.params["alg"])
			if _, ok := cfg.allowedAlgs[alg]; !ok {
				http.Error(w, "unsupported alg", http.StatusUnauthorized)
				return
			}

			pub, err := cfg.resolve(r, entry.params)
			if err != nil {
				http.Error(w, "rs key not found", http.StatusUnauthorized)
				return
			}

			// Buffer body for content-digest or component coverage if needed
			var body []byte
			if r.Body != nil {
				defer r.Body.Close()
				body, _ = io.ReadAll(io.LimitReader(r.Body, 1<<20))
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			base, err := buildSignatureBase(r, entry)
			if err != nil {
				http.Error(w, "cannot build signature base", http.StatusUnauthorized)
				return
			}
			rawSig, err := extractSignatureForLabel(sig, "sig1")
			if err != nil {
				http.Error(w, "invalid Signature header", http.StatusUnauthorized)
				return
			}

			if err := verifySignature(alg, pub, base, rawSig); err != nil {
				http.Error(w, "invalid http signature", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// --- RFC 9421 minimal parsing and verification ---

type sigInputEntry struct {
	components []string
	params     map[string]string
}

func parseSignatureInputForLabel(h, label string) (*sigInputEntry, error) {
	// Expect: Signature-Input: sig1=("@method" "@target-uri");created=1697044520;keyid="rs-kid";alg="ecdsa-p256-sha256"
	parts := splitTopLevel(h, ',')
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.HasPrefix(p, label+"=") {
			continue
		}
		rest := strings.TrimPrefix(p, label+"=")
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(rest, "(") {
			return nil, errors.New("missing components")
		}
		idx := strings.Index(rest, ")")
		if idx < 0 {
			return nil, errors.New("unterminated components")
		}
		compStr := rest[1:idx]
		rest = strings.TrimSpace(rest[idx+1:])
		var comps []string
		for _, c := range strings.Fields(compStr) {
			c = strings.TrimSpace(c)
			c = strings.Trim(c, "\"")
			if c != "" {
				comps = append(comps, c)
			}
		}
		params := map[string]string{}
		for len(rest) > 0 {
			if rest[0] != ';' {
				break
			}
			rest = rest[1:]
			kv, next := nextParam(rest)
			rest = next
			if kv.k == "" {
				continue
			}
			params[strings.ToLower(kv.k)] = kv.v
		}
		return &sigInputEntry{components: comps, params: params}, nil
	}
	return nil, errors.New("label not found")
}

type kvp struct{ k, v string }

func nextParam(s string) (kvp, string) {
	s = strings.TrimSpace(s)
	i := strings.IndexAny(s, "=\";")
	if i < 0 {
		return kvp{}, ""
	}
	key := strings.TrimSpace(s[:i])
	rest := strings.TrimSpace(s[i:])
	if strings.HasPrefix(rest, "=\"") {
		rest = rest[2:]
		j := strings.Index(rest, "\"")
		if j < 0 {
			return kvp{}, ""
		}
		return kvp{key, rest[:j]}, strings.TrimSpace(rest[j+1:])
	}
	if strings.HasPrefix(rest, "=") {
		rest = rest[1:]
		j := strings.Index(rest, ";")
		if j < 0 {
			return kvp{key, strings.TrimSpace(rest)}, ""
		}
		return kvp{key, strings.TrimSpace(rest[:j])}, strings.TrimSpace(rest[j:])
	}
	return kvp{}, ""
}

func ensureRequired(have, required []string) error {
	set := map[string]struct{}{}
	for _, c := range have {
		set[strings.ToLower(c)] = struct{}{}
	}
	for _, need := range required {
		if _, ok := set[strings.ToLower(need)]; !ok {
			return fmt.Errorf("missing required component %q", need)
		}
	}
	return nil
}

func checkCreated(params map[string]string, maxSkew int64) error {
	created := params["created"]
	if created == "" {
		return nil
	}
	sec, err := parseInt(created)
	if err != nil {
		return errors.New("bad created param")
	}
	now := time.Now().Unix()
	if sec > now+maxSkew || sec < now-maxSkew {
		return errors.New("signature outside time window")
	}
	return nil
}

func parseInt(s string) (int64, error) {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errors.New("not int")
		}
		n = n*10 + int64(ch-'0')
	}
	return n, nil
}

func extractSignatureForLabel(h, label string) ([]byte, error) {
	// Expect: Signature: sig1=:BASE64:
	parts := splitTopLevel(h, ',')
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.HasPrefix(strings.ToLower(p), strings.ToLower(label)+"=") {
			continue
		}
		v := strings.TrimSpace(p[len(label)+1:])
		v = strings.TrimSpace(v)
		if !strings.HasPrefix(v, ":") || !strings.HasSuffix(v, ":") {
			return nil, errors.New("sig not sf-binary")
		}
		b64 := v[1 : len(v)-1]
		return base64.StdEncoding.DecodeString(b64)
	}
	return nil, errors.New("label not found")
}

func buildSignatureBase(r *http.Request, e *sigInputEntry) ([]byte, error) {
	var b bytes.Buffer
	for _, c := range e.components {
		lc := strings.ToLower(c)
		switch {
		case strings.HasPrefix(lc, "@"):
			switch lc {
			case "@method":
				fmt.Fprintf(&b, "\"@method\": %s\n", strings.ToLower(r.Method))
			case "@target-uri":
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				host := r.Host
				if host == "" {
					host = r.URL.Host
				}
				fmt.Fprintf(&b, "\"@target-uri\": %s://%s%s\n", scheme, host, r.URL.RequestURI())
			case "@authority":
				host := r.Host
				if host == "" {
					host = r.URL.Host
				}
				fmt.Fprintf(&b, "\"@authority\": %s\n", strings.ToLower(host))
			default:
				return nil, fmt.Errorf("unsupported derived component %q", c)
			}
		default:
			// header field; HTTP field names are case-insensitive
			hname := textproto.CanonicalMIMEHeaderKey(c)
			vals := r.Header.Values(hname)
			if len(vals) == 0 {
				return nil, fmt.Errorf("missing covered header %q", c)
			}
			// RFC 9421 covers the field-value; we join with comma+space if multiple
			fmt.Fprintf(&b, "\"%s\": %s\n", strings.ToLower(c), strings.Join(vals, ", "))
		}
	}
	// Signature params line
	var comps []string
	for _, c := range e.components {
		comps = append(comps, fmt.Sprintf("\"%s\"", c))
	}
	var params []string
	if v := e.params["created"]; v != "" {
		params = append(params, fmt.Sprintf("created=%s", v))
	}
	if v := e.params["keyid"]; v != "" {
		params = append(params, fmt.Sprintf("keyid=%q", v))
	}
	if v := e.params["alg"]; v != "" {
		params = append(params, fmt.Sprintf("alg=%q", strings.ToLower(v)))
	}
	if v := e.params["nonce"]; v != "" {
		params = append(params, fmt.Sprintf("nonce=%q", v))
	}
	fmt.Fprintf(&b, "\"@signature-params\": (%s);%s\n", strings.Join(comps, " "), strings.Join(params, ";"))
	return b.Bytes(), nil
}

func verifySignature(alg string, pub crypto.PublicKey, base, sig []byte) error {
	switch alg {
	case "ed25519":
		pk, ok := pub.(ed25519.PublicKey)
		if !ok {
			return errors.New("key type mismatch")
		}
		if !ed25519.Verify(pk, base, sig) {
			return errors.New("bad signature")
		}
		return nil
	case "ecdsa-p256-sha256":
		pk, ok := pub.(*ecdsa.PublicKey)
		if !ok || pk.Curve.Params().Name != "P-256" {
			return errors.New("key type mismatch")
		}
		h := sha256.Sum256(base)
		if ok := ecdsa.VerifyASN1(pk, h[:], sig); !ok {
			return errors.New("bad signature")
		}
		return nil
	default:
		return errors.New("unsupported alg")
	}
}

// splitTopLevel splits on sep, ignoring commas inside quotes or parentheses.
func splitTopLevel(s string, sep rune) []string {
	var out []string
	var buf strings.Builder
	depth := 0
	inQuotes := false
	for _, r := range s {
		switch r {
		case '"':
			inQuotes = !inQuotes
			buf.WriteRune(r)
		case '(':
			depth++
			buf.WriteRune(r)
		case ')':
			depth--
			buf.WriteRune(r)
		default:
			if r == sep && depth == 0 && !inQuotes {
				out = append(out, buf.String())
				buf.Reset()
				continue
			}
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
	}
	return out
}
