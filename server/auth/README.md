# auth

Security/identity primitives for the **Quickly Pick** backend.

This package is **not user-login auth**. It provides lightweight credentials and identifiers used by handlers and the database layer:

- Random IDs for polls/options/ballots/snapshots
- Deterministic **admin keys** (proof of poll ownership)
- Random **voter tokens** (credential to submit/update a ballot)
- Deterministic **share slugs** (public poll URL identifier)
- Privacy-preserving **IP hashing** (dedup/anti-abuse without storing raw IP)

## Overview

### IDs
`GenerateID(byteLen)` returns a **random hex string** (`byteLen * 2` hex chars). Use these as primary keys for poll/option/ballot/snapshot rows.

### Admin key (deterministic)
`GenerateAdminKey(pollID, salt)` returns:

- `HMAC-SHA256(key=salt, message=pollID)`  
- encoded as **base64 URL-safe**, **no padding**

This is deterministic: the same poll ID + salt always yields the same admin key, and it can be verified server-side with `ValidateAdminKey`.

> **Important:** If you change `ADMIN_KEY_SALT`, previously issued admin keys will no longer validate.

### Voter token (random)
`GenerateVoterToken()` returns:

- 24 random bytes (192 bits entropy)
- encoded as **base64 URL-safe**, **no padding**

This is intended to be stored (e.g., in `username_claim` and used to locate/UPSERT the voter’s ballot). Treat it like a password.

### Share slug (deterministic)
`GenerateShareSlug(pollID, salt)` returns:

- `HMAC-SHA256(key=salt, message=pollID)`
- takes the **first 8 bytes**
- converts to **base62** (`0-9a-zA-Z`) via an internal encoder

This yields a short, URL-friendly slug (typically up to ~11 chars, depending on leading zeros).

> **Important:** If you change `POLL_SLUG_SALT`, existing slugs will change and old public URLs will break.

### IP hashing (privacy-preserving)
`HashIP(ip, salt)` returns:

- `HMAC-SHA256(key=salt, message=ip)`
- truncated to the **first 8 bytes**
- hex-encoded (16 hex chars)

This supports dedup/anti-abuse while avoiding storing raw IPs.

## API

### `GenerateID(byteLen int) (string, error)`
Creates a random hex ID.

```go
pollID, err := auth.GenerateID(16)     // 32 hex chars
optionID, err := auth.GenerateID(12)   // 24 hex chars
ballotID, err := auth.GenerateID(16)
````

Suggested sizes (not enforced):

* Poll IDs: 16 bytes
* Option/Ballot/Snapshot IDs: 12–16 bytes

---

### `GenerateAdminKey(pollID, salt string) string`

Deterministic admin key for poll ownership.

```go
adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
```

Use when:

* returning from `POST /polls`
* verifying `X-Admin-Key` for admin endpoints (`/options`, `/publish`, `/close`, etc.)

---

### `ValidateAdminKey(pollID, adminKey, salt string) error`

Validates an admin key for a given poll.

Returns:

* `nil` on success
* `auth.ErrInvalidAdminKey` on failure

```go
if err := auth.ValidateAdminKey(pollID, providedKey, cfg.AdminKeySalt); err != nil {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

---

### `GenerateVoterToken() (string, error)`

Generates a random voter credential (base64url, no padding).

```go
voterToken, err := auth.GenerateVoterToken()
```

Use when:

* `POST /polls/:slug/claim-username` returns a token
* `POST /polls/:slug/ballots` uses `X-Voter-Token` to authenticate ballot submit/update

> Note: this package currently does **not** implement a `ValidateVoterToken` function. The unused `ErrInvalidToken` constant is reserved for potential format validation in the future.

---

### `GenerateShareSlug(pollID, salt string) string`

Generates a deterministic public slug for a poll (base62).

```go
slug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
// store in polls.share_slug
```

Use when:

* publishing a poll (e.g., `POST /polls/:id/publish`)

---

### `HashIP(ip, salt string) string`

Hashes an IP address for storage/analytics without retaining raw IPs.

```go
ipHash := auth.HashIP(clientIP, ipHashSalt)
```

Recommended:

* Use a **dedicated** IP hashing salt (separate from admin/slug salts) and keep it secret.

## Configuration (service-level)

Typical env vars used by the service:

* `ADMIN_KEY_SALT` → passed as `cfg.AdminKeySalt`
* `POLL_SLUG_SALT` → passed as `cfg.PollSlugSalt`

(If you also hash IPs, you should provide a separate secret salt for that purpose.)

## Security notes

* **Never log** admin keys or voter tokens.
* Use **HTTPS** in production so headers like `X-Admin-Key` / `X-Voter-Token` aren’t exposed.
* Salt rotation is disruptive for deterministic outputs:

  * rotating `ADMIN_KEY_SALT` breaks admin-key validation for existing polls
  * rotating `POLL_SLUG_SALT` breaks public URLs for existing polls
* Slugs are short and deterministic; in the extremely unlikely event of a collision, handle it at the DB layer (e.g., unique constraint + regenerate using a different scheme/version if needed).

## Tests

```bash
go test -v ./auth
go test -bench=. ./auth
```

## Dependencies

* `crypto/hmac`, `crypto/sha256` for deterministic keys/slugs/IP hashing
* `crypto/rand` for secure randomness
* `encoding/base64`, `encoding/hex` for encodings