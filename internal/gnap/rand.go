package gnap

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func RandHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RandUserCode() string {
	h := strings.ToUpper(RandHex(4)) // 8 hex chars
	return h[:4] + "-" + h[4:]
}
