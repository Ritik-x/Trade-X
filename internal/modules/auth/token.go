package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const refreshTokenBytes = 32

// GenerateRefreshToken creates a cryptographically secure random token.
func GenerateRefreshToken() (string, error) {

	randomBytes := make([]byte, refreshTokenBytes)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	return hex.EncodeToString(randomBytes), nil
}

// HashRefreshToken returns the SHA-256 hash of a refresh token.
func HashRefreshToken(token string) string {

	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
