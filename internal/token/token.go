package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateToken() (secret string, hash string) {
	// we generate 32 crypt secure bytes.
	// rand read return an error but in fact panics if there are problem with the source.
	// panic here is good!
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)

	// random bytes encoded as hex
	secret = hex.EncodeToString(randomBytes)

	digest := sha256.Sum256([]byte(secret))

	// encode the hash as hex
	hash = hex.EncodeToString(digest[:])

	return
}
