// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielhkuo/quickly-pick/testutil"
)

func TestHealthEndpoint(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	mux := NewRouter(db, cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestRootEndpoint(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	mux := NewRouter(db, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expected := "quickly-pick API v1"
	if w.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, w.Body.String())
	}
}

func TestRouteExistence(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	mux := NewRouter(db, cfg)

	// Test that routes respond (handler is invoked)
	// Note: Some routes return 404 when data doesn't exist, which is valid handler behavior
	testCases := []struct {
		method string
		path   string
	}{
		// Health and root
		{"GET", "/health"},
		{"GET", "/"},

		// Poll management routes (these use {id} param and may return auth errors)
		{"POST", "/polls"},
		{"GET", "/polls/test-id/admin"},
		{"POST", "/polls/test-id/options"},
		{"POST", "/polls/test-id/publish"},
		{"POST", "/polls/test-id/close"},

		// Voting routes (these use {slug} param)
		{"POST", "/polls/test-slug/claim-username"},
		{"POST", "/polls/test-slug/ballots"},

		// Device routes
		{"POST", "/devices/register"},
		{"GET", "/devices/me"},
		{"GET", "/devices/my-polls"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			// Route should be matched (not 405 Method Not Allowed for these specific routes)
			// 400, 401, 404 are all valid responses depending on handler logic
			if w.Code == http.StatusMethodNotAllowed {
				t.Errorf("Route %s %s returned 405, expected route handler to exist", tc.method, tc.path)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	mux := NewRouter(db, cfg)

	// Test that unsupported methods on defined routes return 405
	testCases := []struct {
		method string
		path   string
	}{
		{"POST", "/health"}, // Only GET is defined
		{"DELETE", "/polls/test-id/admin"}, // Only GET is defined
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405 for %s %s, got %d", tc.method, tc.path, w.Code)
			}
		})
	}
}

func TestPathParameterExtraction(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()

	// Create a test poll to verify path parameters work
	pollID, adminKey, _ := testutil.CreateTestPoll(t, db, cfg, "draft")

	mux := NewRouter(db, cfg)

	// Test that {id} parameter extracts correctly
	t.Run("poll ID extraction", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/polls/"+pollID+"/admin", nil)
		req.Header.Set("X-Admin-Key", adminKey)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should not be 404 (route matched) and not 400 (ID extracted)
		if w.Code == http.StatusNotFound {
			t.Error("Route should have matched")
		}
		// With valid admin key and poll, should return 200
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with valid admin key, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

func TestSpecificMethodRouting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	mux := NewRouter(db, cfg)

	// Test that method-specific routes are enforced
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		// POST /health doesn't exist, should return 405
		{"POST to health endpoint", "POST", "/health", http.StatusMethodNotAllowed},
		// PUT /polls/test/options doesn't exist, POST does
		{"PUT to options endpoint", "PUT", "/polls/test-id/options", http.StatusMethodNotAllowed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected %d for %s %s, got %d", tc.expectedStatus, tc.method, tc.path, w.Code)
			}
		})
	}
}
