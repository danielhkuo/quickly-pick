# Models Package

Data types and structures for the Quickly Pick polling service.

## Overview

This package defines all data models, request/response types, and constants used throughout the application. It provides a centralized location for type definitions and ensures consistency across the API.

## Constants

### Poll Status

```go
const (
    StatusDraft  = "draft"   // Poll is being created, not yet open for voting
    StatusOpen   = "open"    // Poll is published and accepting votes
    StatusClosed = "closed"  // Poll is closed, results are available
)
```

**Status Flow:**
```
draft → open → closed
```

- **draft**: Poll creation phase, options can be added/modified
- **open**: Voting phase, ballots can be submitted/updated
- **closed**: Results phase, final rankings are available

### Voting Method

```go
const (
    MethodBMJ = "bmj"  // Balanced Majority Judgment
)
```

Currently only BMJ is supported, but the design allows for future voting methods.

## Request Types

### CreatePollRequest

```go
type CreatePollRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    CreatorName string `json:"creator_name"`
}
```

**Used by:** `POST /polls`

**Validation:**
- `title` is required, max 200 characters
- `description` is optional, max 1000 characters  
- `creator_name` is required, max 100 characters

### AddOptionRequest

```go
type AddOptionRequest struct {
    Label string `json:"label"`
}
```

**Used by:** `POST /polls/:id/options`

**Validation:**
- `label` is required, max 200 characters
- Poll must be in draft status

### ClaimUsernameRequest

```go
type ClaimUsernameRequest struct {
    Username string `json:"username"`
}
```

**Used by:** `POST /polls/:slug/claim-username`

**Validation:**
- `username` is required, 2-50 characters
- Must be unique within the poll
- Poll must be open

### SubmitBallotRequest

```go
type SubmitBallotRequest struct {
    Scores map[string]float64 `json:"scores"`  // option_id -> value01
}
```

**Used by:** `POST /polls/:slug/ballots`

**Format:**
```json
{
  "scores": {
    "option_abc123": 0.8,
    "option_def456": 0.2,
    "option_ghi789": 0.5
  }
}
```

**Validation:**
- `scores` map is required and non-empty
- All values must be in range [0.0, 1.0]
- All keys must be valid option IDs for the poll
- Voter must have valid token for the poll

## Response Types

### CreatePollResponse

```go
type CreatePollResponse struct {
    PollID   string `json:"poll_id"`
    AdminKey string `json:"admin_key"`
}
```

**Returned by:** `POST /polls`

**Security:** Admin key should be stored securely by the creator.

### AddOptionResponse

```go
type AddOptionResponse struct {
    OptionID string `json:"option_id"`
}
```

**Returned by:** `POST /polls/:id/options`

### PublishPollResponse

```go
type PublishPollResponse struct {
    ShareSlug string `json:"share_slug"`
    ShareURL  string `json:"share_url"`
}
```

**Returned by:** `POST /polls/:id/publish`

**Usage:** Share the `share_url` with voters to participate in the poll.

### ClaimUsernameResponse

```go
type ClaimUsernameResponse struct {
    VoterToken string `json:"voter_token"`
}
```

**Returned by:** `POST /polls/:slug/claim-username`

**Security:** Voter token should be stored by the client for ballot submission.

### SubmitBallotResponse

```go
type SubmitBallotResponse struct {
    BallotID string `json:"ballot_id"`
    Message  string `json:"message"`
}
```

**Returned by:** `POST /polls/:slug/ballots`

**Messages:**
- "Ballot submitted successfully" (new ballot)
- "Ballot updated successfully" (existing ballot edited)

### ClosePollResponse

```go
type ClosePollResponse struct {
    ClosedAt time.Time      `json:"closed_at"`
    Snapshot ResultSnapshot `json:"snapshot"`
}
```

**Returned by:** `POST /polls/:id/close`

Contains the final BMJ results and timestamp.

## Domain Types

### Poll

```go
type Poll struct {
    ID              string     `json:"id"`
    Title           string     `json:"title"`
    Description     string     `json:"description"`
    CreatorName     string     `json:"creator_name"`
    Method          string     `json:"method"`
    Status          string     `json:"status"`
    ShareSlug       *string    `json:"share_slug,omitempty"`
    ClosesAt        *time.Time `json:"closes_at,omitempty"`
    ClosedAt        *time.Time `json:"closed_at,omitempty"`
    FinalSnapshotID *string    `json:"final_snapshot_id,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}
```

**Core poll entity with lifecycle tracking.**

**Field Notes:**
- `ShareSlug` is only set when poll is published (status = open)
- `ClosesAt` is optional auto-close time (not implemented in MVP)
- `ClosedAt` is set when poll is manually closed
- `FinalSnapshotID` references the result snapshot

### Option

```go
type Option struct {
    ID     string `json:"id"`
    PollID string `json:"poll_id"`
    Label  string `json:"label"`
}
```

**Poll choice/option that voters can score.**

### PollWithOptions

```go
type PollWithOptions struct {
    Poll    Poll     `json:"poll"`
    Options []Option `json:"options"`
}
```

**Returned by:** `GET /polls/:slug`

**Composite type for poll details with all options.**

### Ballot

```go
type Ballot struct {
    ID          string    `json:"id"`
    PollID      string    `json:"poll_id"`
    VoterToken  string    `json:"-"`        // Never expose in JSON
    SubmittedAt time.Time `json:"submitted_at"`
    IPHash      *string   `json:"-"`        // Never expose in JSON
    UserAgent   *string   `json:"-"`        // Never expose in JSON
}
```

**Voter's submission with privacy protection.**

**Security Notes:**
- `VoterToken` is never exposed in API responses
- `IPHash` and `UserAgent` are for abuse detection only
- Only `ID`, `PollID`, and `SubmittedAt` are public

### Score

```go
type Score struct {
    BallotID string  `json:"ballot_id"`
    OptionID string  `json:"option_id"`
    Value01  float64 `json:"value01"`
}
```

**Individual option score within a ballot.**

**Value Semantics:**
- `0.0` = "Hate" (strongly dislike)
- `0.5` = "Meh" (neutral/indifferent)
- `1.0` = "Love" (strongly like)

## BMJ Result Types

### OptionStats

```go
type OptionStats struct {
    OptionID string  `json:"option_id"`
    Label    string  `json:"label"`
    Median   float64 `json:"median"`
    P10      float64 `json:"p10"`
    P90      float64 `json:"p90"`
    Mean     float64 `json:"mean"`
    NegShare float64 `json:"neg_share"`
    Veto     bool    `json:"veto"`
    Rank     int     `json:"rank"`  // 1-indexed ranking
}
```

**BMJ statistics for a single option.**

**Field Descriptions:**
- `Median` - 50th percentile of signed scores
- `P10` - 10th percentile (least-misery measure)
- `P90` - 90th percentile (upside measure)
- `Mean` - Arithmetic average of signed scores
- `NegShare` - Fraction of negative scores [0, 1]
- `Veto` - True if soft veto applied (neg_share ≥ 0.33 AND median ≤ 0)
- `Rank` - Final ranking position (1 = winner)

### ResultSnapshot

```go
type ResultSnapshot struct {
    ID         string        `json:"id"`
    PollID     string        `json:"poll_id"`
    Method     string        `json:"method"`
    ComputedAt time.Time     `json:"computed_at"`
    Rankings   []OptionStats `json:"rankings"`
    InputsHash string        `json:"inputs_hash"`
}
```

**Immutable snapshot of poll results.**

**Returned by:** `GET /polls/:slug/results` (only when poll is closed)

**Field Notes:**
- `Rankings` are sorted by final BMJ ranking (rank 1 first)
- `InputsHash` is a hash of all ballot IDs for verification
- Snapshots are immutable once created

## Error Types

### ErrorResponse

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}
```

**Standard error response format.**

**Usage:**
```json
{
  "error": "Bad Request",
  "message": "title is required"
}
```

## Data Validation

### String Length Limits

| Field | Min | Max | Notes |
|-------|-----|-----|-------|
| Poll Title | 1 | 200 | Required |
| Poll Description | 0 | 1000 | Optional |
| Creator Name | 1 | 100 | Required |
| Option Label | 1 | 200 | Required |
| Username | 2 | 50 | Required, unique per poll |

### Numeric Ranges

| Field | Min | Max | Notes |
|-------|-----|-----|-------|
| Score Value01 | 0.0 | 1.0 | Slider position |
| BMJ Statistics | -1.0 | 1.0 | Signed scores |
| Neg Share | 0.0 | 1.0 | Fraction |

### Status Transitions

Valid status transitions:
- `draft` → `open` (via publish)
- `open` → `closed` (via close)

Invalid transitions (will return 409 Conflict):
- `open` → `draft`
- `closed` → `open`
- `closed` → `draft`

## JSON Serialization

### Omitempty Fields

Fields with `omitempty` tag are excluded when nil/empty:
- `Poll.ShareSlug` - Only present when published
- `Poll.ClosesAt` - Only if auto-close is set
- `Poll.ClosedAt` - Only when poll is closed
- `Poll.FinalSnapshotID` - Only when results computed

### Privacy Protection

Fields with `json:"-"` are never serialized:
- `Ballot.VoterToken` - Sensitive credential
- `Ballot.IPHash` - Privacy protection
- `Ballot.UserAgent` - Privacy protection

### Time Format

All timestamps use RFC3339 format:
```json
{
  "created_at": "2024-10-30T14:30:00Z",
  "closed_at": "2024-10-30T16:45:00Z"
}
```

## Usage Examples

### Creating a Poll

```go
req := models.CreatePollRequest{
    Title:       "Best Programming Language",
    Description: "Vote for your favorite language",
    CreatorName: "Alice",
}

// Handler processes request and returns:
resp := models.CreatePollResponse{
    PollID:   "poll_abc123",
    AdminKey: "admin_xyz789",
}
```

### Submitting Scores

```go
req := models.SubmitBallotRequest{
    Scores: map[string]float64{
        "option_go":     0.9,  // Love Go
        "option_python": 0.7,  // Like Python
        "option_java":   0.2,  // Dislike Java
    },
}
```

### BMJ Results

```go
snapshot := models.ResultSnapshot{
    ID:     "snap_def456",
    Method: models.MethodBMJ,
    Rankings: []models.OptionStats{
        {
            OptionID: "option_go",
            Label:    "Go",
            Rank:     1,
            Median:   0.8,
            P10:      0.2,
            P90:      1.0,
            Mean:     0.75,
            NegShare: 0.1,
            Veto:     false,
        },
        // ... more options
    },
}
```

## Testing Utilities

### Test Data Creation

```go
func CreateTestPoll() models.Poll {
    return models.Poll{
        ID:          "test_poll_123",
        Title:       "Test Poll",
        Description: "A test poll",
        CreatorName: "Test User",
        Method:      models.MethodBMJ,
        Status:      models.StatusDraft,
        CreatedAt:   time.Now(),
    }
}

func CreateTestOption(pollID string) models.Option {
    return models.Option{
        ID:     "test_option_123",
        PollID: pollID,
        Label:  "Test Option",
    }
}
```

### Validation Testing

```go
func TestScoreValidation(t *testing.T) {
    validScores := map[string]float64{
        "opt1": 0.0,   // Valid: hate
        "opt2": 0.5,   // Valid: meh
        "opt3": 1.0,   // Valid: love
    }
    
    invalidScores := map[string]float64{
        "opt1": -0.1,  // Invalid: below range
        "opt2": 1.1,   // Invalid: above range
    }
    
    // Test validation logic...
}
```

## Dependencies

- `time` - Timestamp handling
- `encoding/json` - JSON serialization (implicit via struct tags)

## Best Practices

1. **Use constants** - Always use defined constants for status and method values
2. **Validate ranges** - Check numeric values are within expected bounds
3. **Protect sensitive data** - Use `json:"-"` for sensitive fields
4. **Use omitempty** - For optional fields that should be excluded when empty
5. **Document constraints** - Include validation rules in comments
6. **Test serialization** - Verify JSON output matches expectations
7. **Version compatibility** - Consider backwards compatibility when modifying types