# Middleware Package

HTTP middleware and utility functions for the Quickly Pick polling service.

## Overview

This package provides common HTTP middleware functions and utilities used across all handlers. It includes logging, response formatting, request parsing, and client IP extraction functionality.

## Functions

### WithLogging

```go
func WithLogging(next http.HandlerFunc) http.HandlerFunc
```

Wraps a handler with structured request logging.

**Features:**
- Logs request start with method, path, and remote address
- Logs request completion with duration
- Uses structured logging (slog) for better observability

**Usage:**
```go
mux.HandleFunc("/polls", middleware.WithLogging(pollHandler.CreatePoll))
```

**Log Output:**
```
INFO request started method=POST path=/polls remote=127.0.0.1:54321
INFO request completed method=POST path=/polls duration_ms=45
```

### JSONResponse

```go
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{})
```

Writes a JSON response with proper headers and status code.

**Parameters:**
- `w` - HTTP response writer
- `statusCode` - HTTP status code (200, 201, 400, etc.)
- `data` - Any serializable data structure

**Features:**
- Sets `Content-Type: application/json` header
- Handles JSON encoding errors gracefully
- Logs encoding failures

**Usage:**
```go
response := models.CreatePollResponse{
    PollID:   pollID,
    AdminKey: adminKey,
}
middleware.JSONResponse(w, http.StatusCreated, response)
```

### ErrorResponse

```go
func ErrorResponse(w http.ResponseWriter, statusCode int, message string)
```

Writes a standardized JSON error response.

**Parameters:**
- `w` - HTTP response writer
- `statusCode` - HTTP error status code
- `message` - Specific error message for the client

**Response Format:**
```json
{
  "error": "Bad Request",
  "message": "title is required"
}
```

**Usage:**
```go
if req.Title == "" {
    middleware.ErrorResponse(w, http.StatusBadRequest, "title is required")
    return
}
```

**Common Status Codes:**
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (missing/invalid auth)
- `403` - Forbidden (results sealed)
- `404` - Not Found (poll/option not found)
- `409` - Conflict (username taken, wrong status)
- `500` - Internal Server Error (database issues)

### ParseJSONBody

```go
func ParseJSONBody(r *http.Request, v interface{}) error
```

Parses JSON request body into the provided struct.

**Parameters:**
- `r` - HTTP request with JSON body
- `v` - Pointer to struct to decode into

**Returns:**
- `error` - JSON parsing error, if any

**Features:**
- Automatically closes request body
- Handles malformed JSON gracefully
- Type-safe decoding

**Usage:**
```go
var req models.CreatePollRequest
if err := middleware.ParseJSONBody(r, &req); err != nil {
    middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
    return
}
```

### GetClientIP

```go
func GetClientIP(r *http.Request) string
```

Extracts the real client IP address from HTTP request.

**IP Source Priority:**
1. `X-Forwarded-For` header (load balancers, proxies)
2. `X-Real-IP` header (nginx reverse proxy)
3. `RemoteAddr` field (direct connection)

**Features:**
- Handles comma-separated IP lists in X-Forwarded-For
- Strips port numbers from addresses
- Works behind reverse proxies and load balancers

**Usage:**
```go
clientIP := middleware.GetClientIP(r)
ipHash := auth.HashIP(clientIP, cfg.AdminKeySalt)
```

**Example Headers:**
```
X-Forwarded-For: 203.0.113.1, 198.51.100.1
X-Real-IP: 203.0.113.1
```

## Middleware Patterns

### Request Logging

```go
func setupRoutes() *http.ServeMux {
    mux := http.NewServeMux()
    
    // Wrap all handlers with logging
    mux.HandleFunc("POST /polls", middleware.WithLogging(pollHandler.CreatePoll))
    mux.HandleFunc("GET /polls/{slug}", middleware.WithLogging(resultsHandler.GetPoll))
    
    return mux
}
```

### Error Handling Pattern

```go
func exampleHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Parse input
    var req ExampleRequest
    if err := middleware.ParseJSONBody(r, &req); err != nil {
        middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    
    // 2. Validate input
    if req.Field == "" {
        middleware.ErrorResponse(w, http.StatusBadRequest, "field is required")
        return
    }
    
    // 3. Process request
    result, err := processRequest(req)
    if err != nil {
        slog.Error("processing failed", "error", err)
        middleware.ErrorResponse(w, http.StatusInternalServerError, "Processing failed")
        return
    }
    
    // 4. Return success response
    middleware.JSONResponse(w, http.StatusOK, result)
}
```

## Advanced Middleware (Future Extensions)

### CORS Middleware

```go
func WithCORS(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Key, X-Voter-Token")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next(w, r)
    }
}
```

### Rate Limiting Middleware

```go
func WithRateLimit(requestsPerMinute int) func(http.HandlerFunc) http.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(requestsPerMinute/60), requestsPerMinute)
    
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                middleware.ErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded")
                return
            }
            next(w, r)
        }
    }
}
```

### Authentication Middleware

```go
func RequireAdminKey(cfg cliparse.Config) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            pollID := r.PathValue("id")
            adminKey := r.Header.Get("X-Admin-Key")
            
            if err := auth.ValidateAdminKey(pollID, adminKey, cfg.AdminKeySalt); err != nil {
                middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid admin key")
                return
            }
            
            next(w, r)
        }
    }
}
```

## Response Headers

### Security Headers

```go
func WithSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next(w, r)
    }
}
```

### Cache Control

```go
func WithCacheControl(maxAge int) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
            next(w, r)
        }
    }
}
```

## Error Response Standards

### HTTP Status Code Usage

| Status | When to Use | Example |
|--------|-------------|---------|
| 200 | Successful GET/PUT | Poll details retrieved |
| 201 | Successful POST | Poll created, ballot submitted |
| 400 | Invalid input | Missing required field, invalid JSON |
| 401 | Authentication required | Missing admin key, invalid voter token |
| 403 | Forbidden action | Results sealed, insufficient permissions |
| 404 | Resource not found | Poll not found, option not found |
| 409 | Conflict with current state | Username taken, poll wrong status |
| 500 | Server error | Database failure, internal error |

### Error Message Guidelines

**Good error messages:**
- "title is required"
- "score for option_abc must be between 0 and 1"
- "Username already taken"
- "Poll is not open for voting"

**Avoid:**
- Generic messages: "Bad request"
- Technical details: "SQL constraint violation"
- Security information: "Admin key validation failed with HMAC mismatch"

## Logging Best Practices

### Structured Logging

```go
slog.Info("poll created", 
    "poll_id", pollID,
    "creator", creatorName,
    "option_count", len(options))

slog.Error("database error",
    "error", err,
    "operation", "insert_poll",
    "poll_id", pollID)
```

### Log Levels

- **INFO** - Normal operations (poll created, ballot submitted)
- **WARN** - Unusual but handled situations (rate limit hit)
- **ERROR** - Errors that affect functionality (database failures)

### Sensitive Data

**Never log:**
- Admin keys
- Voter tokens
- Full IP addresses (use hashed IPs)
- User agent strings (unless debugging)

**Safe to log:**
- Poll IDs
- Option IDs
- Ballot IDs (they're not sensitive)
- Usernames (they're public within polls)
- Error messages (without sensitive details)

## Testing

### Middleware Testing

```go
func TestWithLogging(t *testing.T) {
    handler := middleware.WithLogging(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    req := httptest.NewRequest("GET", "/test", nil)
    w := httptest.NewRecorder()
    
    handler(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    // Verify logs were written (if using test logger)
}
```

### Response Testing

```go
func TestJSONResponse(t *testing.T) {
    w := httptest.NewRecorder()
    data := map[string]string{"message": "test"}
    
    middleware.JSONResponse(w, http.StatusOK, data)
    
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
    
    var response map[string]string
    json.NewDecoder(w.Body).Decode(&response)
    assert.Equal(t, "test", response["message"])
}
```

### Error Response Testing

```go
func TestErrorResponse(t *testing.T) {
    w := httptest.NewRecorder()
    
    middleware.ErrorResponse(w, http.StatusBadRequest, "test error")
    
    assert.Equal(t, http.StatusBadRequest, w.Code)
    
    var response models.ErrorResponse
    json.NewDecoder(w.Body).Decode(&response)
    assert.Equal(t, "Bad Request", response.Error)
    assert.Equal(t, "test error", response.Message)
}
```

## Performance Considerations

### JSON Encoding

- Use streaming encoder for large responses
- Consider response compression for large payloads
- Cache frequently accessed data

### Logging Performance

- Use structured logging (faster than string formatting)
- Avoid logging in hot paths
- Use appropriate log levels

### Memory Usage

- Don't hold references to request/response bodies
- Close resources properly
- Use appropriate buffer sizes

## Dependencies

- `encoding/json` - JSON encoding/decoding
- `log/slog` - Structured logging
- `net/http` - HTTP handling
- `time` - Duration measurement
- `github.com/danielhkuo/quickly-pick/models` - Error response types

## Best Practices

1. **Consistent error format** - Always use ErrorResponse for errors
2. **Proper status codes** - Use appropriate HTTP status codes
3. **Structured logging** - Include relevant context in logs
4. **Security headers** - Set appropriate security headers
5. **Input validation** - Validate all inputs before processing
6. **Resource cleanup** - Close request bodies and other resources
7. **Performance monitoring** - Log request durations for monitoring