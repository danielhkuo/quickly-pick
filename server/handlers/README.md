# Handlers Package

HTTP request handlers for the Quickly Pick polling service API endpoints.

## Overview

This package contains all HTTP handlers that implement the REST API endpoints defined in the [API Documentation](../../docs/api.md). Each handler is responsible for processing HTTP requests, validating input, performing business logic, and returning appropriate responses.

## Handler Structure

The package is organized into three main handler types:

- **PollHandler** - Poll management operations (create, add options, publish, close)
- **VotingHandler** - Voting operations (claim username, submit ballot)
- **ResultsHandler** - Data retrieval operations (get poll info, get results)

## Handler Types

### PollHandler

Manages poll lifecycle operations that require admin authentication.

```go
type PollHandler struct {
    db  *sql.DB
    cfg cliparse.Config
}

func NewPollHandler(db *sql.DB, cfg cliparse.Config) *PollHandler
```

#### Methods

**CreatePoll** - `POST /polls`

- Creates a new poll in draft status
- Generates poll ID and admin key
- Returns poll_id and admin_key for creator

**AddOption** - `POST /polls/:id/options`

- Adds an option to a draft poll
- Requires valid admin key
- Validates poll is in draft status

**PublishPoll** - `POST /polls/:id/publish`

- Publishes a draft poll (makes it open for voting)
- Generates share slug for public access
- Requires at least 2 options

**ClosePoll** - `POST /polls/:id/close`

- Closes an open poll and computes results
- Creates result snapshot with BMJ rankings
- Seals results for public access

### VotingHandler

Handles voter registration and ballot submission.

```go
type VotingHandler struct {
    db  *sql.DB
    cfg cliparse.Config
}

func NewVotingHandler(db *sql.DB, cfg cliparse.Config) *VotingHandler
```

#### Methods

**ClaimUsername** - `POST /polls/:slug/claim-username`

- Registers a unique username for a poll
- Generates and returns voter token
- Enforces username uniqueness per poll

**SubmitBallot** - `POST /polls/:slug/ballots`

- Submits or updates voter's ballot
- Requires valid voter token
- Supports ballot editing while poll is open

### ResultsHandler

Provides read-only access to poll data and results.

```go
type ResultsHandler struct {
    db  *sql.DB
    cfg cliparse.Config
}

func NewResultsHandler(db *sql.DB, cfg cliparse.Config) *ResultsHandler
```

#### Methods

**GetPoll** - `GET /polls/:slug`

- Returns poll metadata and options
- Available for open and closed polls
- Does NOT include results (results are sealed)

**GetResults** - `GET /polls/:slug/results`

- Returns final BMJ results and rankings
- Only available for closed polls (403 if open)
- Implements results sealing requirement

**GetBallotCount** - `GET /polls/:slug/ballot-count`

- Returns number of submitted ballots
- Available even while poll is open
- Useful for participation tracking

## Authentication & Authorization

### Admin Key Validation

Admin operations require `X-Admin-Key` header:

```go
adminKey := r.Header.Get("X-Admin-Key")
if err := auth.ValidateAdminKey(pollID, adminKey, h.cfg.AdminKeySalt); err != nil {
    middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid admin key")
    return
}
```

**Protected endpoints:**

- `POST /polls/:id/options`
- `POST /polls/:id/publish`
- `POST /polls/:id/close`

### Voter Token Validation

Voting operations require `X-Voter-Token` header:

```go
voterToken := r.Header.Get("X-Voter-Token")
if voterToken == "" {
    middleware.ErrorResponse(w, http.StatusUnauthorized, "X-Voter-Token header required")
    return
}
```

**Protected endpoints:**

- `POST /polls/:slug/ballots`

## Request/Response Flow

### Poll Creation Flow

```go
// 1. Create poll (draft status)
POST /polls
{
  "title": "Best Pizza Topping",
  "description": "Vote for your favorite",
  "creator_name": "Alice"
}
→ {"poll_id": "abc123", "admin_key": "xyz789"}

// 2. Add options
POST /polls/abc123/options
X-Admin-Key: xyz789
{"label": "Pepperoni"}
→ {"option_id": "opt456"}

// 3. Publish poll (open status)
POST /polls/abc123/publish
X-Admin-Key: xyz789
→ {"share_slug": "pizza-poll", "share_url": "https://..."}
```

### Voting Flow

```go
// 1. Claim username
POST /polls/pizza-poll/claim-username
{"username": "bob_voter"}
→ {"voter_token": "voter123"}

// 2. Submit ballot
POST /polls/pizza-poll/ballots
X-Voter-Token: voter123
{
  "scores": {
    "opt456": 0.8,
    "opt789": 0.2
  }
}
→ {"ballot_id": "ballot456", "message": "Ballot submitted successfully"}
```

### Results Flow

```go
// 1. Close poll (admin only)
POST /polls/abc123/close
X-Admin-Key: xyz789
→ {"closed_at": "2024-10-30T...", "snapshot": {...}}

// 2. Get results (public, only after close)
GET /polls/pizza-poll/results
→ {
  "id": "snap123",
  "rankings": [
    {
      "option_id": "opt456",
      "label": "Pepperoni",
      "rank": 1,
      "median": 0.8,
      "veto": false
    }
  ]
}
```

## Input Validation

### Poll Creation

- `title` is required (max 200 chars)
- `creator_name` is required (max 100 chars)
- `description` is optional (max 1000 chars)

### Option Addition

- `label` is required (max 200 chars)
- Poll must be in draft status
- Admin key must be valid

### Username Claiming

- `username` is required (2-50 chars)
- Must be unique within poll
- Poll must be open

### Ballot Submission

- `scores` map is required and non-empty
- All score values must be in range [0, 1]
- All option_ids must be valid for the poll
- Voter token must be valid for the poll

## Error Handling

All handlers use consistent error response format:

```go
type ErrorResponse struct {
    Error   string `json:"error"`     // HTTP status text
    Message string `json:"message"`   // Specific error details
}
```

### Common Error Scenarios

**400 Bad Request**

- Invalid JSON in request body
- Missing required fields
- Invalid score values (not in [0,1])
- Invalid option IDs

**401 Unauthorized**

- Missing or invalid admin key
- Missing or invalid voter token

**403 Forbidden**

- Attempting to get results while poll is open
- Results sealing enforcement

**404 Not Found**

- Poll not found by ID or slug
- Option not found

**409 Conflict**

- Username already taken
- Poll not in expected status (e.g., trying to add options to open poll)
- Attempting to vote on closed poll

**500 Internal Server Error**

- Database connection issues
- Transaction failures
- ID generation failures

## Database Operations

### Transaction Usage

Handlers use database transactions for multi-step operations:

```go
// Ballot submission (UPSERT pattern)
tx, err := h.db.Begin()
if err != nil {
    // handle error
}
defer tx.Rollback()

// Update or insert ballot
// Delete old scores
// Insert new scores

if err := tx.Commit(); err != nil {
    // handle error
}
```

### UPSERT Pattern

Ballot submission supports editing via UPSERT:

1. Check if ballot exists for voter_token
2. If exists: UPDATE ballot, DELETE old scores
3. If new: INSERT ballot
4. INSERT new scores

### Results Sealing

Critical security feature - results are hidden until poll closes:

```go
if status != models.StatusClosed {
    middleware.ErrorResponse(w, http.StatusForbidden, "Results are hidden until poll is closed")
    return
}
```

## Logging

All handlers use structured logging with relevant context:

```go
slog.Info("poll created", "poll_id", pollID, "creator", req.CreatorName)
slog.Error("failed to insert poll", "error", err)
```

**Logged events:**

- Poll creation, publishing, closing
- Username claims
- Ballot submissions (with update flag)
- Database errors
- Authentication failures

## Testing

### Unit Testing

Each handler should be tested with:

```go
func TestCreatePoll(t *testing.T) {
    db := setupTestDB(t)
    cfg := testConfig()
    handler := NewPollHandler(db, cfg)

    req := httptest.NewRequest("POST", "/polls", strings.NewReader(`{
        "title": "Test Poll",
        "creator_name": "Alice"
    }`))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    handler.CreatePoll(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)

    var resp models.CreatePollResponse
    json.NewDecoder(w.Body).Decode(&resp)
    assert.NotEmpty(t, resp.PollID)
    assert.NotEmpty(t, resp.AdminKey)
}
```

### Integration Testing

Test complete workflows:

```go
func TestPollWorkflow(t *testing.T) {
    // 1. Create poll
    // 2. Add options
    // 3. Publish poll
    // 4. Claim usernames
    // 5. Submit ballots
    // 6. Close poll
    // 7. Verify results
}
```

### Error Case Testing

Test all error conditions:

```go
func TestCreatePoll_MissingTitle(t *testing.T) {
    // Test 400 error for missing title
}

func TestAddOption_InvalidAdminKey(t *testing.T) {
    // Test 401 error for invalid admin key
}

func TestGetResults_PollOpen(t *testing.T) {
    // Test 403 error for results sealing
}
```

## Performance Considerations

### Database Queries

- Use prepared statements for repeated queries
- Minimize N+1 query problems
- Use transactions appropriately
- Index on frequently queried columns

### Response Size

- Limit option count per poll (reasonable limits)
- Paginate results if needed for large polls
- Use appropriate JSON field selection

### Concurrency

- Handle concurrent ballot submissions gracefully
- Use database constraints for data integrity
- Implement proper transaction isolation

## Security Considerations

### Input Sanitization

- Validate all input parameters
- Use parameterized queries (prevent SQL injection)
- Limit string lengths
- Validate numeric ranges

### Authentication

- Never log sensitive tokens
- Use secure token generation
- Validate tokens on every request
- Implement proper HMAC validation

### Rate Limiting

Consider implementing rate limiting for:

- Username claiming (prevent spam)
- Ballot submission (prevent abuse)
- Poll creation (prevent DoS)

## Dependencies

- `database/sql` - Database operations
- `net/http` - HTTP handling
- `encoding/json` - JSON processing
- `log/slog` - Structured logging
- `time` - Timestamp handling
- `github.com/danielhkuo/quickly-pick/auth` - Authentication utilities
- `github.com/danielhkuo/quickly-pick/middleware` - HTTP middleware
- `github.com/danielhkuo/quickly-pick/models` - Data types

## Best Practices

1. **Validate early** - Check input before database operations
2. **Use transactions** - For multi-step operations
3. **Log appropriately** - Include relevant context, avoid sensitive data
4. **Handle errors gracefully** - Return appropriate HTTP status codes
5. **Follow REST conventions** - Use proper HTTP methods and status codes
6. **Maintain consistency** - Use same patterns across all handlers
7. **Test thoroughly** - Cover happy path and error cases
