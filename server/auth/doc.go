// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package auth provides authentication and token generation utilities.

# Admin Keys

Admin keys use HMAC-SHA256 to create deterministic, verifiable keys:

	adminKey := auth.GenerateAdminKey(pollID, salt)
	err := auth.ValidateAdminKey(pollID, adminKey, salt)

The key is URL-safe base64 encoded without padding. Since it's deterministic,
the same poll ID and salt always produce the same key. This allows validation
without storing the key in the database.

# Voter Tokens

Voter tokens are random 24-byte (192-bit) secrets:

	token, err := auth.GenerateVoterToken()

Tokens are URL-safe base64 encoded and used to authenticate ballot submissions.
Each voter gets a unique token when claiming a username.

# Share Slugs

Share slugs create URL-friendly identifiers for published polls:

	slug := auth.GenerateShareSlug(pollID, salt)

Slugs are base62 encoded (alphanumeric only) for easy sharing. Like admin keys,
they're deterministic from the poll ID and salt.

# ID Generation

Random hex IDs for database records:

	id, err := auth.GenerateID(16)  // 32 hex characters

# IP Hashing

For privacy-preserving fraud detection:

	hash := auth.HashIP(ipAddress, salt)

Returns first 8 bytes (16 hex chars) of HMAC-SHA256.
*/
package auth
