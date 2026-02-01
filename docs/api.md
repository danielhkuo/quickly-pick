# API Reference

Complete REST API documentation for the Quickly Pick polling service.

## Base URL

- Development: `http://localhost:3318`
- Production: Configure via `VITE_API_BASE_URL`

## Authentication

Quickly Pick uses header-based authentication with two token types:

### Admin Authentication (X-Admin-Key)

Admin operations require the `X-Admin-Key` header. This key is returned when creating a poll and is derived from `HMAC-SHA256(poll_id, salt)`.

```
X-Admin-Key: <admin_key>
```

### Voter Authentication (X-Voter-Token)

Voting operations require the `X-Voter-Token` header. This token is returned when claiming a username for a poll.

```
X-Voter-Token: <voter_token>
```

### Device Tracking (X-Device-UUID)

Optional header for native app device tracking. Enables poll history across sessions.

```
X-Device-UUID: <device_uuid>
```

## Error Responses

All errors return JSON with this structure:

```json
{
  "error": "Bad Request",
  "message": "title is required"
}
```

Common HTTP status codes:
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (missing or invalid token)
- `403` - Forbidden (action not allowed, e.g., viewing sealed results)
- `404` - Not Found (poll/option doesn't exist)
- `409` - Conflict (invalid state transition)
- `500` - Internal Server Error

---

## Endpoints

### Health Check

#### GET /health

Returns server health status.

**Response:** `200 OK`
```
OK
```

**Example:**
```bash
curl http://localhost:3318/health
```

---

### Poll Management (Admin)

#### POST /polls

Create a new poll.

**Request Body:**
```json
{
  "title": "Where should we eat?",
  "description": "Friday lunch spot",
  "creator_name": "Alice"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Poll title |
| `description` | string | No | Optional description |
| `creator_name` | string | Yes | Name of the poll creator |

**Response:** `201 Created`
```json
{
  "poll_id": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
  "admin_key": "Hk9X2mPqR5tYwZ3nL8vBcFgJdKsA7eNuQoMpCxIyTzU"
}
```

**Example:**
```bash
curl -X POST http://localhost:3318/polls \
  -H "Content-Type: application/json" \
  -d '{"title": "Where should we eat?", "creator_name": "Alice"}'
```

---

#### GET /polls/{id}/admin

Get poll details for admin access.

**Headers:**
- `X-Admin-Key` (required)

**Response:** `200 OK`
```json
{
  "poll": {
    "id": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
    "title": "Where should we eat?",
    "description": "Friday lunch spot",
    "creator_name": "Alice",
    "method": "bmj",
    "status": "draft",
    "share_slug": null,
    "closes_at": null,
    "closed_at": null,
    "created_at": "2025-01-15T10:30:00Z"
  },
  "options": []
}
```

**Example:**
```bash
curl http://localhost:3318/polls/a1b2c3d4e5f67890a1b2c3d4e5f67890/admin \
  -H "X-Admin-Key: Hk9X2mPqR5tYwZ3nL8vBcFgJdKsA7eNuQoMpCxIyTzU"
```

---

#### POST /polls/{id}/options

Add an option to a draft poll.

**Headers:**
- `X-Admin-Key` (required)

**Request Body:**
```json
{
  "label": "Sushi Palace"
}
```

**Response:** `201 Created`
```json
{
  "option_id": "opt123456789abc"
}
```

**Errors:**
- `409 Conflict` - Poll is not in draft status

**Example:**
```bash
curl -X POST http://localhost:3318/polls/a1b2c3d4/options \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: Hk9X2mPqR5tYwZ3nL8vBcFgJdKsA7eNuQoMpCxIyTzU" \
  -d '{"label": "Sushi Palace"}'
```

---

#### POST /polls/{id}/publish

Publish a draft poll to open it for voting.

**Headers:**
- `X-Admin-Key` (required)

**Requirements:**
- Poll must be in `draft` status
- Poll must have at least 2 options

**Response:** `200 OK`
```json
{
  "share_slug": "k7Yz3mNx",
  "share_url": "https://quickly-pick.com/polls/k7Yz3mNx"
}
```

**Errors:**
- `400 Bad Request` - Poll must have at least 2 options
- `409 Conflict` - Poll is not in draft status

**Example:**
```bash
curl -X POST http://localhost:3318/polls/a1b2c3d4/publish \
  -H "X-Admin-Key: Hk9X2mPqR5tYwZ3nL8vBcFgJdKsA7eNuQoMpCxIyTzU"
```

---

#### POST /polls/{id}/close

Close a poll and compute final results.

**Headers:**
- `X-Admin-Key` (required)

**Requirements:**
- Poll must be in `open` status

**Response:** `200 OK`
```json
{
  "closed_at": "2025-01-15T18:00:00Z",
  "snapshot": {
    "id": "snap123456789abc",
    "poll_id": "a1b2c3d4",
    "method": "bmj",
    "computed_at": "2025-01-15T18:00:00Z",
    "rankings": [
      {
        "option_id": "opt1",
        "label": "Sushi Palace",
        "median": 0.6,
        "p10": 0.2,
        "p90": 0.9,
        "mean": 0.55,
        "neg_share": 0.1,
        "veto": false,
        "rank": 1
      }
    ],
    "inputs_hash": "5-ballots"
  }
}
```

**Errors:**
- `409 Conflict` - Poll is not open

**Example:**
```bash
curl -X POST http://localhost:3318/polls/a1b2c3d4/close \
  -H "X-Admin-Key: Hk9X2mPqR5tYwZ3nL8vBcFgJdKsA7eNuQoMpCxIyTzU"
```

---

### Voting (Public)

#### POST /polls/{slug}/claim-username

Claim a unique username for voting on a poll.

**Request Body:**
```json
{
  "username": "Bob"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Unique name (2-50 characters) |

**Response:** `201 Created`
```json
{
  "voter_token": "K7Yz3mNxPqRsTuVwXyZ123AbCdEfGhIj"
}
```

**Errors:**
- `404 Not Found` - Poll not found
- `409 Conflict` - Username already taken OR poll is not open

**Example:**
```bash
curl -X POST http://localhost:3318/polls/k7Yz3mNx/claim-username \
  -H "Content-Type: application/json" \
  -d '{"username": "Bob"}'
```

---

#### POST /polls/{slug}/ballots

Submit or update a ballot.

**Headers:**
- `X-Voter-Token` (required)

**Request Body:**
```json
{
  "scores": {
    "opt1": 0.9,
    "opt2": 0.3,
    "opt3": 0.5
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scores` | object | Yes | Map of option_id to score (0.0-1.0) |

Score interpretation:
- `0.0` = Strong dislike (hate)
- `0.5` = Neutral (meh)
- `1.0` = Strong like (love)

**Response:** `201 Created`
```json
{
  "ballot_id": "ballot123456789abc",
  "message": "Ballot submitted successfully"
}
```

Or if updating an existing ballot:
```json
{
  "ballot_id": "ballot123456789abc",
  "message": "Ballot updated successfully"
}
```

**Errors:**
- `400 Bad Request` - Invalid option_id or score out of range
- `401 Unauthorized` - Invalid voter token
- `404 Not Found` - Poll not found
- `409 Conflict` - Poll is not open

**Example:**
```bash
curl -X POST http://localhost:3318/polls/k7Yz3mNx/ballots \
  -H "Content-Type: application/json" \
  -H "X-Voter-Token: K7Yz3mNxPqRsTuVwXyZ123AbCdEfGhIj" \
  -d '{"scores": {"opt1": 0.9, "opt2": 0.3, "opt3": 0.5}}'
```

---

#### GET /polls/{slug}/my-ballot

Get the current user's existing ballot for a poll.

**Headers:**
- `X-Voter-Token` (required)

**Response:** `200 OK`

If the user has voted:
```json
{
  "scores": {
    "opt1": 0.9,
    "opt2": 0.3,
    "opt3": 0.5
  },
  "submitted_at": "2025-01-15T14:30:00Z",
  "has_voted": true
}
```

If the user has not voted:
```json
{
  "scores": {},
  "submitted_at": null,
  "has_voted": false
}
```

**Errors:**
- `401 Unauthorized` - Missing or invalid voter token
- `404 Not Found` - Poll not found

**Example:**
```bash
curl http://localhost:3318/polls/k7Yz3mNx/my-ballot \
  -H "X-Voter-Token: K7Yz3mNxPqRsTuVwXyZ123AbCdEfGhIj"
```

---

### Results (Public)

#### GET /polls/{slug}

Get poll details and options (no results).

**Response:** `200 OK`
```json
{
  "poll": {
    "id": "a1b2c3d4",
    "title": "Where should we eat?",
    "description": "Friday lunch spot",
    "creator_name": "Alice",
    "method": "bmj",
    "status": "open",
    "share_slug": "k7Yz3mNx",
    "created_at": "2025-01-15T10:30:00Z"
  },
  "options": [
    {"id": "opt1", "poll_id": "a1b2c3d4", "label": "Sushi Palace"},
    {"id": "opt2", "poll_id": "a1b2c3d4", "label": "Pizza Place"},
    {"id": "opt3", "poll_id": "a1b2c3d4", "label": "Burger Joint"}
  ]
}
```

**Example:**
```bash
curl http://localhost:3318/polls/k7Yz3mNx
```

---

#### GET /polls/{slug}/results

Get final poll results (only available after poll is closed).

**Response:** `200 OK`
```json
{
  "poll": {
    "id": "a1b2c3d4",
    "title": "Where should we eat?",
    "status": "closed",
    "closed_at": "2025-01-15T18:00:00Z"
  },
  "rankings": [
    {
      "option_id": "opt1",
      "label": "Sushi Palace",
      "median": 0.6,
      "p10": 0.2,
      "p90": 0.9,
      "mean": 0.55,
      "neg_share": 0.1,
      "veto": false,
      "rank": 1
    },
    {
      "option_id": "opt3",
      "label": "Burger Joint",
      "median": 0.4,
      "p10": -0.2,
      "p90": 0.8,
      "mean": 0.35,
      "neg_share": 0.25,
      "veto": false,
      "rank": 2
    },
    {
      "option_id": "opt2",
      "label": "Pizza Place",
      "median": -0.1,
      "p10": -0.6,
      "p90": 0.4,
      "mean": -0.05,
      "neg_share": 0.45,
      "veto": true,
      "rank": 3
    }
  ],
  "ballot_count": 5
}
```

**Errors:**
- `403 Forbidden` - Results are hidden until poll is closed
- `404 Not Found` - Poll not found

**Example:**
```bash
curl http://localhost:3318/polls/k7Yz3mNx/results
```

---

#### GET /polls/{slug}/ballot-count

Get the number of submitted ballots (available while poll is open).

**Response:** `200 OK`
```json
{
  "ballot_count": 5
}
```

**Example:**
```bash
curl http://localhost:3318/polls/k7Yz3mNx/ballot-count
```

---

#### GET /polls/{slug}/preview

Get compact poll data for link previews (e.g., iMessage bubbles).

**Response:** `200 OK`
```json
{
  "title": "Where should we eat?",
  "status": "open",
  "option_count": 3,
  "ballot_count": 5
}
```

**Example:**
```bash
curl http://localhost:3318/polls/k7Yz3mNx/preview
```

---

### Device Management

#### POST /devices/register

Register a device for poll history tracking.

**Headers:**
- `X-Device-UUID` (required)

**Request Body:**
```json
{
  "platform": "ios"
}
```

| Field | Type | Required | Values |
|-------|------|----------|--------|
| `platform` | string | Yes | `ios`, `macos`, `android`, `web` |

**Response:** `201 Created` (new device) or `200 OK` (existing device)
```json
{
  "device_id": "dev123456789abc",
  "is_new": true
}
```

**Example:**
```bash
curl -X POST http://localhost:3318/devices/register \
  -H "Content-Type: application/json" \
  -H "X-Device-UUID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"platform": "ios"}'
```

---

#### GET /devices/me

Get current device information.

**Headers:**
- `X-Device-UUID` (required)

**Response:** `200 OK`
```json
{
  "id": "dev123456789abc",
  "platform": "ios",
  "created_at": "2025-01-15T10:00:00Z",
  "last_seen_at": "2025-01-15T12:30:00Z"
}
```

**Errors:**
- `404 Not Found` - Device not registered

**Example:**
```bash
curl http://localhost:3318/devices/me \
  -H "X-Device-UUID: 550e8400-e29b-41d4-a716-446655440000"
```

---

#### GET /devices/my-polls

Get all polls associated with this device.

**Headers:**
- `X-Device-UUID` (required)

**Response:** `200 OK`
```json
{
  "polls": [
    {
      "poll_id": "a1b2c3d4",
      "title": "Where should we eat?",
      "status": "closed",
      "share_slug": "k7Yz3mNx",
      "role": "admin",
      "ballot_count": 5,
      "linked_at": "2025-01-15T10:30:00Z"
    },
    {
      "poll_id": "x9y8z7w6",
      "title": "Movie night pick",
      "status": "open",
      "share_slug": "m3Np7Qrs",
      "role": "voter",
      "username": "Bob",
      "ballot_count": 3,
      "linked_at": "2025-01-16T14:00:00Z"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:3318/devices/my-polls \
  -H "X-Device-UUID: 550e8400-e29b-41d4-a716-446655440000"
```

---

## Poll Lifecycle

```
draft → open → closed
```

1. **Draft**: Admin creates poll, adds options
2. **Open**: Admin publishes, voters submit ballots
3. **Closed**: Admin closes, results computed and revealed

| Status | Admin Actions | Voter Actions | Results Visible |
|--------|---------------|---------------|-----------------|
| draft | Add options, publish | None | No |
| open | Close | Claim username, submit/update ballot | No |
| closed | None | None | Yes |
