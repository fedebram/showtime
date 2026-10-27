package token

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

// Matches reports if secret hashes to hash.
func Matches(secret, hash string) bool {
	digest := sha256.Sum256([]byte(secret))
	// to prevent timing attack
	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(digest[:])), []byte(hash)) == 1
}
