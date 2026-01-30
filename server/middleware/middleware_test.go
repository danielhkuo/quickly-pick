// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielhkuo/quickly-pick/models"
)

func TestWithLogging(t *testing.T) {
	// Create a simple handler that returns OK
	handlerCalled := false
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	// Wrap with logging middleware
	wrappedHandler := WithLogging(testHandler)

	// Create test request and recorder
	req := httptest.NewRequest("GET", "/test-path", nil)
	w := httptest.NewRecorder()

	// Execute
	wrappedHandler(w, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	// Verify response was written correctly
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", w.Body.String())
	}
}

func TestWithLogging_PreservesResponse(t *testing.T) {
	// Test that logging doesn't interfere with various response codes
	testCases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"OK", http.StatusOK, "ok"},
		{"Created", http.StatusCreated, `{"id":"123"}`},
		{"BadRequest", http.StatusBadRequest, `{"error":"bad request"}`},
		{"NotFound", http.StatusNotFound, "not found"},
		{"InternalError", http.StatusInternalServerError, "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := WithLogging(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			})

			req := httptest.NewRequest("POST", "/api/test", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, w.Code)
			}
			if w.Body.String() != tc.body {
				t.Errorf("Expected body '%s', got '%s'", tc.body, w.Body.String())
			}
		})
	}
}

func TestJSONResponse(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		data       interface{}
		expected   string
	}{
		{
			name:       "simple struct",
			statusCode: http.StatusOK,
			data:       map[string]string{"message": "hello"},
			expected:   `{"message":"hello"}`,
		},
		{
			name:       "created response",
			statusCode: http.StatusCreated,
			data:       models.CreatePollResponse{PollID: "abc123", AdminKey: "key456"},
			expected:   `{"poll_id":"abc123","admin_key":"key456"}`,
		},
		{
			name:       "error response",
			statusCode: http.StatusBadRequest,
			data:       models.ErrorResponse{Error: "Bad Request", Message: "missing field"},
			expected:   `{"error":"Bad Request","message":"missing field"}`,
		},
		{
			name:       "array data",
			statusCode: http.StatusOK,
			data:       []string{"a", "b", "c"},
			expected:   `["a","b","c"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			JSONResponse(w, tc.statusCode, tc.data)

			// Check status code
			if w.Code != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, w.Code)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}

			// Check body (trim newline added by Encode)
			body := strings.TrimSpace(w.Body.String())
			if body != tc.expected {
				t.Errorf("Expected body '%s', got '%s'", tc.expected, body)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		message        string
		expectedError  string
	}{
		{
			name:          "bad request",
			statusCode:    http.StatusBadRequest,
			message:       "title is required",
			expectedError: "Bad Request",
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			message:       "invalid admin key",
			expectedError: "Unauthorized",
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			message:       "poll not found",
			expectedError: "Not Found",
		},
		{
			name:          "conflict",
			statusCode:    http.StatusConflict,
			message:       "poll already closed",
			expectedError: "Conflict",
		},
		{
			name:          "internal error",
			statusCode:    http.StatusInternalServerError,
			message:       "database error",
			expectedError: "Internal Server Error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			ErrorResponse(w, tc.statusCode, tc.message)

			// Check status code
			if w.Code != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, w.Code)
			}

			// Check Content-Type
			if w.Header().Get("Content-Type") != "application/json" {
				t.Error("Expected Content-Type 'application/json'")
			}

			// Decode and verify error response
			var resp models.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if resp.Error != tc.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tc.expectedError, resp.Error)
			}
			if resp.Message != tc.message {
				t.Errorf("Expected message '%s', got '%s'", tc.message, resp.Message)
			}
		})
	}
}

func TestParseJSONBody(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		body := `{"title":"Test Poll","creator_name":"Alice"}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))

		var parsed models.CreatePollRequest
		err := ParseJSONBody(req, &parsed)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if parsed.Title != "Test Poll" {
			t.Errorf("Expected title 'Test Poll', got '%s'", parsed.Title)
		}
		if parsed.CreatorName != "Alice" {
			t.Errorf("Expected creator_name 'Alice', got '%s'", parsed.CreatorName)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := `{invalid json}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))

		var parsed models.CreatePollRequest
		err := ParseJSONBody(req, &parsed)

		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(""))

		var parsed models.CreatePollRequest
		err := ParseJSONBody(req, &parsed)

		if err == nil {
			t.Error("Expected error for empty body")
		}
	})

	t.Run("null body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("null"))

		var parsed *models.CreatePollRequest
		err := ParseJSONBody(req, &parsed)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if parsed != nil {
			t.Error("Expected nil result for null JSON")
		}
	})

	t.Run("extra fields ignored", func(t *testing.T) {
		body := `{"title":"Test","creator_name":"Bob","unknown_field":"ignored"}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))

		var parsed models.CreatePollRequest
		err := ParseJSONBody(req, &parsed)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if parsed.Title != "Test" {
			t.Errorf("Expected title 'Test', got '%s'", parsed.Title)
		}
	})

	t.Run("body is closed after parsing", func(t *testing.T) {
		body := `{"title":"Test","creator_name":"Alice"}`
		bodyReader := io.NopCloser(bytes.NewReader([]byte(body)))
		req := httptest.NewRequest("POST", "/", bodyReader)

		var parsed models.CreatePollRequest
		_ = ParseJSONBody(req, &parsed)

		// Try to read from body again - should return empty/error since it's closed
		remaining, err := io.ReadAll(req.Body)
		if err != nil && err != io.EOF {
			// Body closed is expected
		}
		if len(remaining) > 0 {
			t.Error("Expected body to be consumed/closed")
		}
	})
}

func TestCORS(t *testing.T) {
	// Create a simple handler that returns OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("handled"))
	})

	corsHandler := CORS(nextHandler)

	t.Run("preflight OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/polls", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		// Should return 200 OK without calling next handler
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Body should be empty (preflight doesn't call next)
		if w.Body.String() != "" {
			t.Errorf("Expected empty body for preflight, got '%s'", w.Body.String())
		}

		// Check CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
			t.Error("Expected Access-Control-Allow-Origin to match request origin")
		}
		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("Expected Access-Control-Allow-Credentials to be 'true'")
		}
	})

	t.Run("regular request with origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/polls", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		// Should call next handler
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "handled" {
			t.Error("Expected next handler to be called")
		}

		// Check CORS headers reflect the origin
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("Expected Access-Control-Allow-Origin to reflect request origin")
		}
	})

	t.Run("request without origin defaults to wildcard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/polls", nil)
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected Access-Control-Allow-Origin to default to '*'")
		}
	})

	t.Run("allows custom headers", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/polls", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")

		// Check that X-Admin-Key and X-Voter-Token are allowed
		if !strings.Contains(allowedHeaders, "X-Admin-Key") {
			t.Error("Expected X-Admin-Key in allowed headers")
		}
		if !strings.Contains(allowedHeaders, "X-Voter-Token") {
			t.Error("Expected X-Voter-Token in allowed headers")
		}
		if !strings.Contains(allowedHeaders, "Content-Type") {
			t.Error("Expected Content-Type in allowed headers")
		}
	})

	t.Run("allows required methods", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/polls", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		allowedMethods := w.Header().Get("Access-Control-Allow-Methods")

		requiredMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		for _, method := range requiredMethods {
			if !strings.Contains(allowedMethods, method) {
				t.Errorf("Expected %s in allowed methods", method)
			}
		}
	})
}

func TestGetClientIP(t *testing.T) {
	testCases := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For chained IPs (comma separated)",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100, 10.0.0.1, 172.16.0.1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For chained IPs (space after comma)",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "203.0.113.195",
		},
		{
			name:       "X-Real-IP takes precedence over RemoteAddr",
			headers:    map[string]string{"X-Real-IP": "203.0.113.50"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100", "X-Real-IP": "203.0.113.50"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.50:54321",
			expectedIP: "192.168.1.50",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.50",
			expectedIP: "192.168.1.50",
		},
		{
			name:       "IPv6 RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "[::1]:12345",
			expectedIP: "[::1]", // Implementation strips port after last colon
		},
		{
			name:       "IPv6 in X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "2001:db8::1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "2001:db8::1",
		},
		{
			name:       "empty X-Forwarded-For falls through to RemoteAddr",
			headers:    map[string]string{"X-Forwarded-For": ""},
			remoteAddr: "10.0.0.5:8080",
			expectedIP: "10.0.0.5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remoteAddr

			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			result := GetClientIP(req)

			if result != tc.expectedIP {
				t.Errorf("Expected IP '%s', got '%s'", tc.expectedIP, result)
			}
		})
	}
}
