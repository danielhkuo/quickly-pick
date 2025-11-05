// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package auth

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name    string
		byteLen int
		wantLen int // hex encoded length = byteLen * 2
	}{
		{"8 bytes", 8, 16},
		{"16 bytes", 16, 32},
		{"24 bytes", 24, 48},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenerateID(tt.byteLen)
			if err != nil {
				t.Fatalf("GenerateID() error = %v", err)
			}
			if len(id) != tt.wantLen {
				t.Errorf("GenerateID() length = %d, want %d", len(id), tt.wantLen)
			}
			// Verify it's valid hex
			for _, c := range id {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("GenerateID() contains invalid hex char: %c", c)
				}
			}
		})
	}

	// Test randomness - two IDs should be different
	id1, _ := GenerateID(16)
	id2, _ := GenerateID(16)
	if id1 == id2 {
		t.Error("GenerateID() produced duplicate IDs (extremely unlikely)")
	}
}

func TestGenerateAdminKey(t *testing.T) {
	tests := []struct {
		name   string
		pollID string
		salt   string
	}{
		{"standard", "poll123", "secret-salt"},
		{"empty poll id", "", "salt"},
		{"empty salt", "poll456", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenerateAdminKey(tt.pollID, tt.salt)

			// Should not be empty
			if key == "" {
				t.Error("GenerateAdminKey() returned empty string")
			}

			// Should be deterministic
			key2 := GenerateAdminKey(tt.pollID, tt.salt)
			if key != key2 {
				t.Error("GenerateAdminKey() is not deterministic")
			}

			// Different inputs should produce different keys
			if tt.pollID != "" && tt.salt != "" {
				differentKey := GenerateAdminKey(tt.pollID+"x", tt.salt)
				if key == differentKey {
					t.Error("GenerateAdminKey() produced same key for different poll IDs")
				}
			}

			// Should be URL-safe (no padding)
			if strings.Contains(key, "=") {
				t.Error("GenerateAdminKey() contains padding characters")
			}
		})
	}
}

func TestValidateAdminKey(t *testing.T) {
	pollID := "test-poll-123"
	salt := "test-salt"
	validKey := GenerateAdminKey(pollID, salt)

	tests := []struct {
		name     string
		pollID   string
		adminKey string
		salt     string
		wantErr  bool
	}{
		{"valid key", pollID, validKey, salt, false},
		{"wrong key", pollID, "wrong-key", salt, true},
		{"wrong poll id", "different-poll", validKey, salt, true},
		{"wrong salt", pollID, validKey, "different-salt", true},
		{"empty key", pollID, "", salt, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdminKey(tt.pollID, tt.adminKey, tt.salt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAdminKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != ErrInvalidAdminKey {
				t.Errorf("ValidateAdminKey() error = %v, want %v", err, ErrInvalidAdminKey)
			}
		})
	}
}

func TestGenerateVoterToken(t *testing.T) {
	// Test basic generation
	token, err := GenerateVoterToken()
	if err != nil {
		t.Fatalf("GenerateVoterToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateVoterToken() returned empty string")
	}

	// Should be URL-safe (no padding)
	if strings.Contains(token, "=") {
		t.Error("GenerateVoterToken() contains padding characters")
	}

	// Should be reasonably long (24 bytes encoded)
	if len(token) < 30 {
		t.Errorf("GenerateVoterToken() too short: %d chars", len(token))
	}

	// Test randomness - should not produce duplicates
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateVoterToken()
		if err != nil {
			t.Fatalf("GenerateVoterToken() error on iteration %d: %v", i, err)
		}
		if tokens[token] {
			t.Errorf("GenerateVoterToken() produced duplicate token: %s", token)
		}
		tokens[token] = true
	}
}

func TestGenerateShareSlug(t *testing.T) {
	tests := []struct {
		name   string
		pollID string
		salt   string
	}{
		{"standard", "poll-abc-123", "slug-salt"},
		{"different poll", "poll-xyz-456", "slug-salt"},
		{"different salt", "poll-abc-123", "other-salt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug := GenerateShareSlug(tt.pollID, tt.salt)

			// Should not be empty
			if slug == "" {
				t.Error("GenerateShareSlug() returned empty string")
			}

			// Should be deterministic
			slug2 := GenerateShareSlug(tt.pollID, tt.salt)
			if slug != slug2 {
				t.Error("GenerateShareSlug() is not deterministic")
			}

			// Should be reasonably short
			if len(slug) > 15 {
				t.Errorf("GenerateShareSlug() too long: %d chars", len(slug))
			}

			// Should be URL-safe (alphanumeric only)
			for _, c := range slug {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
					t.Errorf("GenerateShareSlug() contains non-alphanumeric char: %c", c)
				}
			}
		})
	}

	// Different inputs should produce different slugs
	slug1 := GenerateShareSlug("poll1", "salt")
	slug2 := GenerateShareSlug("poll2", "salt")
	if slug1 == slug2 {
		t.Error("GenerateShareSlug() produced same slug for different poll IDs")
	}

	slug3 := GenerateShareSlug("poll1", "salt1")
	slug4 := GenerateShareSlug("poll1", "salt2")
	if slug3 == slug4 {
		t.Error("GenerateShareSlug() produced same slug for different salts")
	}
}

func TestBase62Encode(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"zero bytes", []byte{0, 0, 0, 0}},
		{"small value", []byte{0, 0, 0, 1}},
		{"large value", []byte{255, 255, 255, 255, 255, 255, 255, 255}},
		{"mixed value", []byte{42, 123, 200, 17}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base62Encode(tt.input)

			// Should not be empty (except for all zeros -> "0")
			if result == "" {
				t.Error("base62Encode() returned empty string")
			}

			// Should only contain base62 characters
			for _, c := range result {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
					t.Errorf("base62Encode() contains invalid char: %c", c)
				}
			}

			// Should be deterministic
			result2 := base62Encode(tt.input)
			if result != result2 {
				t.Error("base62Encode() is not deterministic")
			}
		})
	}

	// Different inputs should produce different outputs
	out1 := base62Encode([]byte{1, 2, 3, 4})
	out2 := base62Encode([]byte{5, 6, 7, 8})
	if out1 == out2 {
		t.Error("base62Encode() produced same output for different inputs")
	}
}

func TestHashIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		salt string
	}{
		{"IPv4", "192.168.1.1", "ip-salt"},
		{"IPv6", "2001:0db8:85a3::8a2e:0370:7334", "ip-salt"},
		{"localhost", "127.0.0.1", "ip-salt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashIP(tt.ip, tt.salt)

			// Should not be empty
			if hash == "" {
				t.Error("HashIP() returned empty string")
			}

			// Should be 16 hex characters (8 bytes * 2)
			if len(hash) != 16 {
				t.Errorf("HashIP() length = %d, want 16", len(hash))
			}

			// Should be valid hex
			for _, c := range hash {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("HashIP() contains invalid hex char: %c", c)
				}
			}

			// Should be deterministic
			hash2 := HashIP(tt.ip, tt.salt)
			if hash != hash2 {
				t.Error("HashIP() is not deterministic")
			}
		})
	}

	// Different IPs should produce different hashes
	hash1 := HashIP("192.168.1.1", "salt")
	hash2 := HashIP("192.168.1.2", "salt")
	if hash1 == hash2 {
		t.Error("HashIP() produced same hash for different IPs")
	}

	// Different salts should produce different hashes
	hash3 := HashIP("192.168.1.1", "salt1")
	hash4 := HashIP("192.168.1.1", "salt2")
	if hash3 == hash4 {
		t.Error("HashIP() produced same hash for different salts")
	}
}

// Benchmark tests
func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateID(16)
	}
}

func BenchmarkGenerateAdminKey(b *testing.B) {
	pollID := "test-poll-123"
	salt := "test-salt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateAdminKey(pollID, salt)
	}
}

func BenchmarkGenerateVoterToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateVoterToken()
	}
}

func BenchmarkGenerateShareSlug(b *testing.B) {
	pollID := "test-poll-123"
	salt := "slug-salt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateShareSlug(pollID, salt)
	}
}
