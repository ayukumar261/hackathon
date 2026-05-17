package agentphone

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature checks an HMAC-SHA256 signature over body with secret.
// Accepts headers in raw hex ("abcd...") or prefixed ("sha256=abcd...").
// If timestamp is non-empty, also accepts signatures computed over
// "{timestamp}.{body}" (Stripe/AgentPhone style).
func VerifySignature(body []byte, header, timestamp, secret string) bool {
	if secret == "" || header == "" {
		return false
	}
	got := strings.TrimSpace(header)
	if i := strings.Index(got, "="); i >= 0 && strings.EqualFold(got[:i], "sha256") {
		got = got[i+1:]
	}
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return false
	}
	if macEq(gotBytes, body, secret) {
		return true
	}
	if timestamp != "" {
		signed := append([]byte(timestamp+"."), body...)
		if macEq(gotBytes, signed, secret) {
			return true
		}
	}
	return false
}

func macEq(want, payload []byte, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hmac.Equal(want, mac.Sum(nil))
}
