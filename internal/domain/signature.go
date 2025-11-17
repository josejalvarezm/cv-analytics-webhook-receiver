package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// HMACValidator implements SignatureValidator using HMAC-SHA256
type HMACValidator struct {
	secret string
}

// NewHMACValidator creates a new HMAC signature validator
func NewHMACValidator(secret string) *HMACValidator {
	return &HMACValidator{secret: secret}
}

// Validate checks if the payload signature is valid
func (v *HMACValidator) Validate(payload []byte, signature string) error {
	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
