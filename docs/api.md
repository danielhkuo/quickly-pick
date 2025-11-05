# API Documentation

The Quickly Pick API provides REST endpoints for creating polls, voting, and retrieving results.

## Base URL

```
http://localhost:8080
```

## Authentication

- **Admin operations** require an `admin_key` (returned when creating a poll)
- **Voting operations** require a `voter_token` (obtained by claiming a username)
- **Read operations** are public for open/closed polls

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

#### Add Option

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

```http
POST /polls/:id/publish
X-Admin-Key: admin_xyz789
```

**Response:**
```json
{
  "share_slug": "pizza-poll-2024"
}
```

#### Close Poll

```http
POST /polls/:id/close
X-Admin-Key: admin_xyz789
```

**Response:**
```json
{
  "snapshot": {
    "id": "snap_ghi789",
    "ranking": [
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
    ]
  }
}
```

### Voting

#### Claim Username

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
  "ballot_id": "ballot_pqr678"
}
```

### Reading Data

#### Get Poll Info

```http
GET /polls/:slug
```

**Response (while open):**
```json
{
  "id": "poll_abc123",
  "title": "Best Pizza Topping",
  "description": "Vote for your favorite pizza topping",
  "creator_name": "Alice",
  "status": "open",
  "options": [
    {
      "id": "opt_def456",
      "label": "Pepperoni"
    }
  ]
}
```

#### Get Results

```http
GET /polls/:slug/results
```

**Response (if closed):**
```json
{
  "poll": {
    "id": "poll_abc123",
    "title": "Best Pizza Topping",
    "status": "closed"
  },
  "results": {
    "method": "bmj",
    "ranking": [
      {
        "option_id": "opt_def456",
        "label": "Pepperoni",
        "rank": 1,
        "stats": {
          "median": 0.8,
          "p10": 0.2,
          "p90": 1.0,
          "mean": 0.75,
          "neg_share": 0.1,
          "veto": false
        }
      }
    ]
  }
}
```

**Response (if open):**
```json
{
  "error": "Results not available while poll is open",
  "status": 403
}
```

#### Get Ballot Count

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
```json
{
  "status": "ok",
  "timestamp": "2024-10-30T10:30:00Z"
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Error description",
  "code": "ERROR_CODE",
  "status": 400
}
```

### Common Error Codes

- `POLL_NOT_FOUND` (404) - Poll does not exist
- `POLL_CLOSED` (409) - Cannot vote on closed poll
- `UNAUTHORIZED` (401) - Invalid or missing admin_key/voter_token
- `USERNAME_TAKEN` (409) - Username already claimed for this poll
- `INVALID_SCORE` (400) - Score value must be between 0 and 1
- `RESULTS_SEALED` (403) - Results not available while poll is open

## Rate Limiting

- `/claim-username`: 5 requests per minute per IP
- `/ballots`: 10 requests per minute per voter_token
- Other endpoints: 100 requests per minute per IP

## Slider Semantics

The voting slider maps to a `value01` between 0 and 1:

- `0.0` = "Hate" (strongly dislike)
- `0.5` = "Meh" (neutral/indifferent)  
- `1.0` = "Love" (strongly like)

These values are converted to signed scores `s = 2*value01 - 1` for BMJ calculation, giving a range of [-1, 1].