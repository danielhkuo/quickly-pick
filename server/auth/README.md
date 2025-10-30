# Auth Package

Token generation and validation for the quickly-pick polling service.

## Overview

This package provides all authentication and identification primitives needed for the MVP:

- **Admin Keys**: HMAC-based, deterministic keys that prove poll ownership
- **Voter Tokens**: Random secure tokens for ballot submission
- **Share Slugs**: Short, deterministic, URL-friendly poll identifiers
- **ID Generation**: Random identifiers for polls, ballots, etc.
- **IP Hashing**: Privacy-preserving IP tracking for anti-abuse

## Functions

### `GenerateID(byteLen int) (string, error)`
Creates a random hex ID of specified byte length.

**Usage:**
```go
pollID, err := auth.GenerateID(16)  // Returns 32 hex characters
optionID, err := auth.GenerateID(12) // Returns 24 hex characters
```

**Recommended sizes:**
- Poll IDs: 16 bytes (32 hex chars)
- Option/Ballot IDs: 12-16 bytes

---

### `GenerateAdminKey(pollID, salt string) string`
Creates an HMAC-based admin key for a poll. This is deterministic and verifiable.

**Properties:**
- Deterministic: same pollID + salt always produces the same key
- Secure: HMAC-SHA256 prevents forgery
- URL-safe: base64-encoded without padding

**Usage:**
```go
adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
// Return this to the creator when they create a poll
```

**When to use:**
- Return when `POST /polls` succeeds
- Store in client (cookie, localStorage, etc.)

---

### `ValidateAdminKey(pollID, adminKey, salt string) error`
Verifies an admin key is valid for a poll.

**Returns:**
- `nil` if valid
- `auth.ErrInvalidAdminKey` if invalid

**Usage:**
```go
err := auth.ValidateAdminKey(pollID, providedKey, cfg.AdminKeySalt)
if err != nil {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
// Proceed with admin operation
```

**When to use:**
- `POST /polls/:id/options` (add options)
- `POST /polls/:id/publish` (publish poll)
- `POST /polls/:id/close` (close poll)

---

### `GenerateVoterToken() (string, error)`
Creates a random secure token for a voter.

**Properties:**
- 192 bits of entropy (24 bytes)
- Cryptographically random
- URL-safe base64 encoding
- Unique per generation

**Usage:**
```go
voterToken, err := auth.GenerateVoterToken()
if err != nil {
    // Handle error
}
// Store in username_claim table and return to voter
```

**When to use:**
- `POST /polls/:slug/claim-username` - generate and return token

---

### `GenerateShareSlug(pollID, salt string) string`
Creates a short, deterministic URL slug for a poll.

**Properties:**
- Deterministic: same pollID + salt = same slug
- Short: ~11 characters
- URL-safe: alphanumeric only (base62)
- HMAC-based for security

**Usage:**
```go
shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
// Store in poll.share_slug column
shareURL := "https://yourapp.com/polls/" + shareSlug
```

**When to use:**
- `POST /polls/:id/publish` - generate slug when publishing

---

### `HashIP(ip, salt string) string`
One-way hash of IP address for privacy-preserving tracking.

**Properties:**
- HMAC-based (not reversible)
- 64 bits output (16 hex chars)
- Salted to prevent rainbow tables
- Deterministic for same IP

**Usage:**
```go
clientIP := r.RemoteAddr // or from X-Forwarded-For
ipHash := auth.HashIP(clientIP, "ip-hash-salt")
// Store in ballot.ip_hash for duplicate detection
```

**When to use:**
- `POST /polls/:slug/ballots` - track IP for anti-abuse

---

## Integration with Handlers

### Creating a Poll
```go
func CreatePoll(w http.ResponseWriter, r *http.Request) {
    // 1. Generate poll ID
    pollID, _ := auth.GenerateID(16)
    
    // 2. Generate admin key
    adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
    
    // 3. Insert into database
    // ...
    
    // 4. Return to creator
    json.NewEncoder(w).Encode(map[string]string{
        "poll_id":   pollID,
        "admin_key": adminKey,
    })
}
```

### Publishing a Poll
```go
func PublishPoll(w http.ResponseWriter, r *http.Request) {
    pollID := getPollIDFromPath(r)
    adminKey := r.Header.Get("X-Admin-Key")
    
    // 1. Validate admin key
    if err := auth.ValidateAdminKey(pollID, adminKey, cfg.AdminKeySalt); err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // 2. Generate share slug
    shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
    
    // 3. Update database: set status='open', share_slug=shareSlug
    // ...
    
    json.NewEncoder(w).Encode(map[string]string{
        "share_slug": shareSlug,
        "share_url":  "https://yourapp.com/polls/" + shareSlug,
    })
}
```

### Claiming a Username
```go
func ClaimUsername(w http.ResponseWriter, r *http.Request) {
    shareSlug := getSlugFromPath(r)
    username := getRequestedUsername(r)
    
    // 1. Find poll by share_slug
    // ...
    
    // 2. Generate voter token
    voterToken, _ := auth.GenerateVoterToken()
    
    // 3. Insert into username_claim (handles UNIQUE constraint)
    // ...
    
    json.NewEncoder(w).Encode(map[string]string{
        "voter_token": voterToken,
    })
}
```

### Submitting a Ballot
```go
func SubmitBallot(w http.ResponseWriter, r *http.Request) {
    shareSlug := getSlugFromPath(r)
    voterToken := r.Header.Get("X-Voter-Token")
    
    // 1. Verify poll is open
    // ...
    
    // 2. Hash IP for tracking
    clientIP := r.RemoteAddr
    ipHash := auth.HashIP(clientIP, "ip-hash-salt")
    
    // 3. UPSERT ballot with voter_token (allows edits)
    ballotID, _ := auth.GenerateID(16)
    // INSERT ... ON CONFLICT (poll_id, voter_token) DO UPDATE ...
    
    // 4. Store scores
    // ...
}
```

## Security Considerations

1. **Admin Key Salt**: Keep `AdminKeySalt` secret. If leaked, anyone can forge admin keys.
2. **Slug Salt**: Keep `PollSlugSalt` secret to prevent slug enumeration attacks.
3. **Voter Tokens**: Never log voter tokens. They're sensitive credentials.
4. **HTTPS Only**: All tokens/keys must be transmitted over HTTPS.
5. **IP Hashing**: Use a separate salt for IP hashing, never reuse other salts.

## Testing

Run tests with:
```bash
go test -v ./auth
```

Run benchmarks:
```bash
go test -bench=. ./auth
```

## Dependencies

- `crypto/hmac` - HMAC-SHA256 for deterministic keys
- `crypto/rand` - Cryptographically secure random generation
- `crypto/sha256` - Hash function
- `encoding/base64` - URL-safe encoding
- `encoding/hex` - Hex encoding for IDs
