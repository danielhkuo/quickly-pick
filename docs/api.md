# API Documentation

The Quickly Pick API provides REST endpoints for creating polls, voting, and retrieving results.

## Base URL

```
http://localhost:3318
```

## Authentication

- **Admin operations** require an `X-Admin-Key` header (returned when creating a poll).
- **Voting operations** require an `X-Voter-Token` header (obtained by claiming a username).
- **Read operations** are public for open/closed polls (via the share slug).

## Endpoints

### Poll Management

#### Create Poll

```http
POST /polls
Content-Type: application/json

{
  "title": "Best Pizza Topping",
  "description": "Vote for your favorite pizza topping",
  "creator_name": "Alice"
}
```

**Response:**
```json
{
  "poll_id": "poll_abc123",
  "admin_key": "admin_xyz789"
}
```

#### Get Poll (Admin View)

Retrieve poll details using the ID (not slug). Useful for the creator to check status.

```http
GET /polls/:id/admin
X-Admin-Key: admin_xyz789
```

**Response:**
```json
{
  "poll": {
    "id": "poll_abc123",
    "title": "Best Pizza Topping",
    "description": "Vote for your favorite pizza topping",
    "creator_name": "Alice",
    "method": "bmj",
    "status": "draft",
    "created_at": "2024-10-30T10:00:00Z"
  },
  "options": [
    {
      "id": "opt_def456",
      "poll_id": "poll_abc123",
      "label": "Pepperoni"
    }
  ]
}
```

#### Add Option

Only allowed when poll status is `draft`.

```http
POST /polls/:id/options
Content-Type: application/json
X-Admin-Key: admin_xyz789

{
  "label": "Pepperoni"
}
```

**Response:**
```json
{
  "option_id": "opt_def456"
}
```

#### Publish Poll

Changes status from `draft` to `open` and generates a shareable slug.

```http
POST /polls/:id/publish
X-Admin-Key: admin_xyz789
```

**Response:**
```json
{
  "share_slug": "pizza-poll-2024",
  "share_url": "https://quickly-pick.com/polls/pizza-poll-2024"
}
```

#### Close Poll

Changes status to `closed`, computes final BMJ rankings, and creates a result snapshot.

```http
POST /polls/:id/close
X-Admin-Key: admin_xyz789
```

**Response:**
```json
{
  "closed_at": "2024-10-30T12:00:00Z",
  "snapshot": {
    "id": "snap_ghi789",
    "poll_id": "poll_abc123",
    "method": "bmj",
    "computed_at": "2024-10-30T12:00:00Z",
    "rankings": [
      {
        "option_id": "opt_def456",
        "label": "Pepperoni",
        "median": 0.8,
        "p10": 0.2,
        "p90": 1.0,
        "mean": 0.75,
        "neg_share": 0.1,
        "veto": false,
        "rank": 1
      }
    ],
    "inputs_hash": "42-ballots"
  }
}
```

### Voting

#### Claim Username

Obtains a voter token for a specific poll. The username must be unique within that poll.

```http
POST /polls/:slug/claim-username
Content-Type: application/json

{
  "username": "bob_voter"
}
```

**Response:**
```json
{
  "voter_token": "voter_jkl012"
}
```

#### Submit Ballot

Submits or updates a ballot. Scores must be between 0.0 and 1.0.

```http
POST /polls/:slug/ballots
Content-Type: application/json
X-Voter-Token: voter_jkl012

{
  "scores": {
    "opt_def456": 0.8,
    "opt_mno345": 0.2
  }
}
```

**Response:**
```json
{
  "ballot_id": "ballot_pqr678",
  "message": "Ballot submitted successfully"
}
```

### Reading Data

#### Get Poll Info (Public)

Retrieves poll metadata and options. Does **not** include results.

```http
GET /polls/:slug
```

**Response:**
```json
{
  "poll": {
    "id": "poll_abc123",
    "title": "Best Pizza Topping",
    "description": "Vote for your favorite pizza topping",
    "creator_name": "Alice",
    "method": "bmj",
    "status": "open",
    "share_slug": "pizza-poll-2024",
    "created_at": "2024-10-30T10:00:00Z"
  },
  "options": [
    {
      "id": "opt_def456",
      "poll_id": "poll_abc123",
      "label": "Pepperoni"
    }
  ]
}
```

#### Get Results

Retrieves the final rankings. Only available if the poll is `closed`.

```http
GET /polls/:slug/results
```

**Response (if closed):**
```json
{
  "poll": {
    "id": "poll_abc123",
    "title": "Best Pizza Topping",
    "status": "closed",
    ...
  },
  "rankings": [
    {
      "option_id": "opt_def456",
      "label": "Pepperoni",
      "rank": 1,
      "median": 0.8,
      "p10": 0.2,
      "p90": 1.0,
      "mean": 0.75,
      "neg_share": 0.1,
      "veto": false
    }
  ],
  "ballot_count": 42
}
```

**Response (if open):**
```json
{
  "error": "Forbidden",
  "message": "Results are hidden until poll is closed"
}
```
*Status Code: 403 Forbidden*

#### Get Ballot Count

Publicly available count of submitted ballots.

```http
GET /polls/:slug/ballot-count
```

**Response:**
```json
{
  "ballot_count": 42
}
```

### Health Check

#### Health Status

```http
GET /health
```

**Response:**
```text
OK
```
*Content-Type: text/plain*

## Error Responses

All errors follow this JSON format:

```json
{
  "error": "Status Text",
  "message": "Specific error description"
}
```

### Common Error Codes

- `400 Bad Request` - Invalid JSON, missing fields, or invalid scores.
- `401 Unauthorized` - Invalid or missing `X-Admin-Key` or `X-Voter-Token`.
- `403 Forbidden` - Attempting to view results of an open poll.
- `404 Not Found` - Poll or Option does not exist.
- `409 Conflict` - Username taken, poll not in correct status (e.g., voting on closed poll), or publishing a poll with < 2 options.
- `500 Internal Server Error` - Database or server failure.

## Slider Semantics

The voting slider maps to a `value01` between 0 and 1:

- `0.0` = "Hate" (strongly dislike)
- `0.5` = "Meh" (neutral/indifferent)  
- `1.0` = "Love" (strongly like)

These values are converted to signed scores `s = 2*value01 - 1` for BMJ calculation, giving a range of [-1, 1].