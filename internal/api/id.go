package api

import (
	"crypto/rand"
	"encoding/hex"
)

// generateID produces a random 16-byte hex string suitable for record IDs.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
