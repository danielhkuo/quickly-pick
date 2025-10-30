# Router Package

HTTP routing and request handling for the Quickly Pick polling service.

## Overview

This package defines all HTTP endpoints and their handlers. It provides a clean separation between routing logic and business logic, with proper middleware integration and error handling.

## Functions

### `NewRouter(db *sql.DB, cfg cliparse.Config) *http.ServeMux`

Creates and configures the main HTTP router with all endpoints and middleware.

**Parameters:**
- `db *sql.DB` - Database connection for handlers
- `cfg cliparse.Config` - Application configuration

**Returns:**
- `*http.ServeMux` - Configured HTTP router

**Usage:**
```go
import (
    "database/sql"
    "net/http"
    
    "github.com/danielhkuo/quickly-pick/router"
    "github.com/danielhkuo/quickly-pick/cliparse"
)

func main() {
    db, _ := sql.Open("postgres", databaseURL)
    cfg, _ := cliparse.ParseFlags(os.Args[1:])
    
    mux := router.NewRouter(db, cfg)
    
    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }
    
    log.Fatal(server.ListenAndServe())
}
```

## Current Endpoints

### Health Check
- **GET /health** - Service health status

### Root
- **GET /** - API information

## Planned Endpoints (MVP Implementation)

Based on the [API Documentation](../../docs/api.md), the router will implement these endpoints:

### Poll Management
```go
// Poll CRUD operations
mux.HandleFunc("POST /polls", createPollHandler)
mux.HandleFunc("POST /polls/{id}/options", addOptionHandler)
mux.HandleFunc("POST /polls/{id}/publish", publishPollHandler)
mux.HandleFunc("POST /polls/{id}/close", closePollHandler)
```

### Voting Operations
```go
// Voter registration and ballot submission
mux.HandleFunc("POST /polls/{slug}/claim-username", claimUsernameHandler)
mux.HandleFunc("POST /polls/{slug}/ballots", submitBallotHandler)
```

### Data Retrieval
```go
// Public read endpoints
mux.HandleFunc("GET /polls/{slug}", getPollHandler)
mux.HandleFunc("GET /polls/{slug}/results", getResultsHandler)
```

## Handler Implementation Pattern

Each handler follows a consistent pattern:

```go
func exampleHandler(db *sql.DB, cfg *cliparse.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse and validate input
        var req ExampleRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }
        
        // 2. Validate business logic
        if err := validateExampleRequest(req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // 3. Perform database operations
        result, err := performExampleOperation(db, req)
        if err != nil {
            http.Error(w, "Internal error", http.StatusInternalServerError)
            log.Printf("Example operation failed: %v", err)
            return
        }
        
        // 4. Return response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}
```

## Middleware Integration

The router supports middleware for cross-cutting concerns:

### CORS Middleware
```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Rate Limiting Middleware
```go
func rateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(100), 10) // 100 req/sec, burst 10
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Logging Middleware
```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap ResponseWriter to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(wrapped, r)
        
        log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
    })
}
```

## Authentication Handlers

### Admin Key Validation
```go
func requireAdminKey(db *sql.DB, cfg *cliparse.Config) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            pollID := r.PathValue("id")
            adminKey := r.Header.Get("Authorization")
            
            // Remove "Bearer " prefix if present
            if strings.HasPrefix(adminKey, "Bearer ") {
                adminKey = adminKey[7:]
            }
            
            if err := auth.ValidateAdminKey(pollID, adminKey, cfg.AdminKeySalt); err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            next(w, r)
        }
    }
}
```

### Voter Token Validation
```go
func requireVoterToken(db *sql.DB) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            voterToken := r.Header.Get("Authorization")
            
            if strings.HasPrefix(voterToken, "Bearer ") {
                voterToken = voterToken[7:]
            }
            
            if voterToken == "" {
                http.Error(w, "Voter token required", http.StatusUnauthorized)
                return
            }
            
            // Store token in request context for handler use
            ctx := context.WithValue(r.Context(), "voter_token", voterToken)
            next(w, r.WithContext(ctx))
        }
    }
}
```

## Error Handling

### Standard Error Response
```go
type ErrorResponse struct {
    Error  string `json:"error"`
    Code   string `json:"code,omitempty"`
    Status int    `json:"status"`
}

func writeError(w http.ResponseWriter, message string, code string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    
    response := ErrorResponse{
        Error:  message,
        Code:   code,
        Status: status,
    }
    
    json.NewEncoder(w).Encode(response)
}
```

### Common Error Handlers
```go
func handleNotFound(w http.ResponseWriter, r *http.Request) {
    writeError(w, "Resource not found", "NOT_FOUND", http.StatusNotFound)
}

func handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
    writeError(w, "Method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
}

func handleInternalError(w http.ResponseWriter, err error) {
    log.Printf("Internal error: %v", err)
    writeError(w, "Internal server error", "INTERNAL_ERROR", http.StatusInternalServerError)
}
```

## Request/Response Types

### Poll Creation
```go
type CreatePollRequest struct {
    Title       string `json:"title" validate:"required,max=200"`
    Description string `json:"description" validate:"max=1000"`
    CreatorName string `json:"creator_name" validate:"required,max=100"`
}

type CreatePollResponse struct {
    PollID   string `json:"poll_id"`
    AdminKey string `json:"admin_key"`
}
```

### Option Addition
```go
type AddOptionRequest struct {
    Label string `json:"label" validate:"required,max=200"`
}

type AddOptionResponse struct {
    OptionID string `json:"option_id"`
}
```

### Username Claiming
```go
type ClaimUsernameRequest struct {
    Username string `json:"username" validate:"required,max=50"`
}

type ClaimUsernameResponse struct {
    VoterToken string `json:"voter_token"`
}
```

### Ballot Submission
```go
type SubmitBallotRequest struct {
    Scores []ScoreInput `json:"scores" validate:"required,dive"`
}

type ScoreInput struct {
    OptionID string  `json:"option_id" validate:"required"`
    Value01  float64 `json:"value01" validate:"min=0,max=1"`
}

type SubmitBallotResponse struct {
    BallotID string `json:"ballot_id"`
}
```

## Path Parameter Extraction

Using Go 1.22+ path parameters:

```go
func getPollHandler(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")
    if slug == "" {
        http.Error(w, "Missing poll slug", http.StatusBadRequest)
        return
    }
    
    // Use slug to fetch poll...
}
```

## Database Integration

Handlers receive database connection and configuration:

```go
func createHandlers(db *sql.DB, cfg *cliparse.Config) {
    mux.HandleFunc("POST /polls", func(w http.ResponseWriter, r *http.Request) {
        // Handler has access to db and cfg
        createPollHandler(db, cfg)(w, r)
    })
}
```

## Testing

### Handler Testing
```go
func TestCreatePollHandler(t *testing.T) {
    db := setupTestDB(t)
    cfg := &cliparse.Config{AdminKeySalt: "test-salt"}
    
    handler := createPollHandler(db, cfg)
    
    reqBody := `{"title":"Test Poll","creator_name":"Alice"}`
    req := httptest.NewRequest("POST", "/polls", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    w := httptest.NewRecorder()
    handler(w, req)
    
    if w.Code != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", w.Code)
    }
    
    var response CreatePollResponse
    json.NewDecoder(w.Body).Decode(&response)
    
    if response.PollID == "" {
        t.Error("Expected poll_id in response")
    }
}
```

### Integration Testing
```go
func TestPollWorkflow(t *testing.T) {
    server := httptest.NewServer(router.NewRouter(db, cfg))
    defer server.Close()
    
    // 1. Create poll
    createResp := createPoll(t, server.URL, "Test Poll", "Alice")
    
    // 2. Add options
    addOption(t, server.URL, createResp.PollID, createResp.AdminKey, "Option A")
    addOption(t, server.URL, createResp.PollID, createResp.AdminKey, "Option B")
    
    // 3. Publish poll
    publishResp := publishPoll(t, server.URL, createResp.PollID, createResp.AdminKey)
    
    // 4. Vote
    claimResp := claimUsername(t, server.URL, publishResp.ShareSlug, "voter1")
    submitBallot(t, server.URL, publishResp.ShareSlug, claimResp.VoterToken, scores)
    
    // 5. Close and check results
    closePoll(t, server.URL, createResp.PollID, createResp.AdminKey)
    results := getResults(t, server.URL, publishResp.ShareSlug)
    
    // Verify results...
}
```

## Performance Considerations

### Connection Pooling
```go
// Configure database connection pool
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Request Timeouts
```go
server := &http.Server{
    Addr:         ":8080",
    Handler:      mux,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

### Response Caching
```go
func cacheMiddleware(duration time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(duration.Seconds())))
            next.ServeHTTP(w, r)
        })
    }
}
```

## Security Considerations

### Input Validation
- Validate all request parameters
- Sanitize user input
- Use parameterized queries
- Implement rate limiting

### Headers
- Set security headers (CORS, CSP, etc.)
- Validate Content-Type
- Handle OPTIONS requests properly

### Error Information
- Don't expose internal details in error messages
- Log detailed errors server-side
- Return generic error messages to clients

## Dependencies

- `net/http` - Standard HTTP server
- `database/sql` - Database operations
- `encoding/json` - JSON encoding/decoding
- `log` - Logging
- `context` - Request context management

## Best Practices

1. **Keep handlers focused** - One responsibility per handler
2. **Use middleware** for cross-cutting concerns
3. **Validate input early** - Fail fast on invalid requests
4. **Handle errors gracefully** - Always return appropriate status codes
5. **Log important events** - Track poll creation, voting, etc.
6. **Use structured responses** - Consistent JSON format
7. **Test thoroughly** - Unit and integration tests for all endpoints