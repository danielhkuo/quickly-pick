// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidAdminKey = errors.New("invalid admin key")
	ErrInvalidToken    = errors.New("invalid token format")
)

// GenerateID creates a random hex ID of the specified byte length
func GenerateID(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateAdminKey creates an HMAC-based admin key for a poll
// This is deterministic and verifiable
func GenerateAdminKey(pollID, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(pollID))
	sum := h.Sum(nil)
	// Use URL-safe base64 and trim padding for cleaner keys
	return strings.TrimRight(base64.URLEncoding.EncodeToString(sum), "=")
}

// ValidateAdminKey checks if the provided admin key is valid for the poll
func ValidateAdminKey(pollID, adminKey, salt string) error {
	expected := GenerateAdminKey(pollID, salt)
	if !hmac.Equal([]byte(adminKey), []byte(expected)) {
		return ErrInvalidAdminKey
	}
	return nil
}

// GenerateVoterToken creates a random secure token for a voter
// This is used to identify voters and allow ballot updates
func GenerateVoterToken() (string, error) {
	b := make([]byte, 24) // 24 bytes = 192 bits of entropy
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate voter token: %w", err)
	}
	// URL-safe base64 without padding
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="), nil
}

// GenerateShareSlug creates a short, deterministic URL slug for a poll
// Uses HMAC for determinism and base62 encoding for URL-friendliness
func GenerateShareSlug(pollID, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(pollID))
	sum := h.Sum(nil)

	// Take first 8 bytes for a shorter slug
	shortHash := sum[:8]

	// Convert to base62 (alphanumeric only, no special chars)
	return base62Encode(shortHash)
}

// base62Encode converts bytes to base62 (0-9, a-z, A-Z)
// This creates URL-friendly slugs without special characters
func base62Encode(data []byte) string {
	const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Convert bytes to a big integer
	var num uint64
	for i := 0; i < len(data) && i < 8; i++ {
		num = num<<8 | uint64(data[i])
	}

	if num == 0 {
		return "0"
	}

	// Convert to base62
	result := make([]byte, 0, 11) // max length for uint64
	for num > 0 {
		result = append(result, base62Chars[num%62])
		num /= 62
	}

	// Reverse the string
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// HashIP creates a one-way hash of an IP address for privacy
// Includes salt to prevent rainbow table attacks
func HashIP(ip, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(ip))
	sum := h.Sum(nil)
	// Return first 16 hex chars (64 bits) - enough for deduplication
	return hex.EncodeToString(sum[:8])
}
