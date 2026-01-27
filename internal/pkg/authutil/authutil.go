package authutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

var DefaultDecisionSecret = []byte("ops-hub-decision-secret")

// Sign creates a base64 encoded HMAC-SHA256 signature for the provided message.
func Sign(message string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Verify checks whether the signature matches the message using the provided secret.
func Verify(message, signature string, secret []byte) bool {
	expected := Sign(message, secret)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}
