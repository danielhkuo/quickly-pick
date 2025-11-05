// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielhkuo/quickly-pick/auth"
	"github.com/danielhkuo/quickly-pick/cliparse"
	"github.com/danielhkuo/quickly-pick/models"
	_ "github.com/lib/pq"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "postgres://quicklypick:devpassword@localhost:5432/quickly_pick_dev?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Clean up tables before each test
	_, err = db.Exec(`
		DROP TABLE IF EXISTS result_snapshot CASCADE;
		DROP TABLE IF EXISTS score CASCADE;
		DROP TABLE IF EXISTS ballot CASCADE;
		DROP TABLE IF EXISTS username_claim CASCADE;
		DROP TABLE IF EXISTS option CASCADE;
		DROP TABLE IF EXISTS poll CASCADE;
	`)
	if err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE poll (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			creator_name TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT 'bmj',
			status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
			share_slug TEXT UNIQUE,
			closes_at TIMESTAMP,
			closed_at TIMESTAMP,
			final_snapshot_id TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE option (
			id TEXT PRIMARY KEY,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			label TEXT NOT NULL
		);

		CREATE TABLE username_claim (
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			username TEXT NOT NULL,
			voter_token TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (poll_id, voter_token),
			UNIQUE (poll_id, username)
		);

		CREATE TABLE ballot (
			id TEXT PRIMARY KEY,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			voter_token TEXT NOT NULL,
			submitted_at TIMESTAMP NOT NULL DEFAULT NOW(),
			ip_hash TEXT,
			user_agent TEXT,
			UNIQUE (poll_id, voter_token)
		);

		CREATE TABLE score (
			ballot_id TEXT NOT NULL REFERENCES ballot(id) ON DELETE CASCADE,
			option_id TEXT NOT NULL REFERENCES option(id) ON DELETE CASCADE,
			value01 REAL NOT NULL CHECK (value01 >= 0 AND value01 <= 1),
			PRIMARY KEY (ballot_id, option_id)
		);

		CREATE TABLE result_snapshot (
			id TEXT PRIMARY KEY,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			method TEXT NOT NULL,
			computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
			payload JSONB NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func getTestConfig() cliparse.Config {
	return cliparse.Config{
		Port:         8080,
		DatabaseURL:  "postgres://test",
		AdminKeySalt: "test-admin-salt",
		PollSlugSalt: "test-slug-salt",
	}
}

func TestCreatePoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.CreatePollResponse)
	}{
		{
			name: "valid poll creation",
			requestBody: models.CreatePollRequest{
				Title:       "Test Poll",
				Description: "Test description",
				CreatorName: "Alice",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *models.CreatePollResponse) {
				if resp.PollID == "" {
					t.Error("Expected non-empty poll_id")
				}
				if resp.AdminKey == "" {
					t.Error("Expected non-empty admin_key")
				}

				// Verify admin key is valid
				expectedKey := auth.GenerateAdminKey(resp.PollID, cfg.AdminKeySalt)
				if resp.AdminKey != expectedKey {
					t.Error("Admin key does not match expected value")
				}

				// Verify poll was created in database
				var status string
				err := db.QueryRow("SELECT status FROM poll WHERE id = $1", resp.PollID).Scan(&status)
				if err != nil {
					t.Fatalf("Failed to query poll: %v", err)
				}
				if status != models.StatusDraft {
					t.Errorf("Expected status 'draft', got '%s'", status)
				}
			},
		},
		{
			name: "missing title",
			requestBody: models.CreatePollRequest{
				Description: "Test description",
				CreatorName: "Alice",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing creator name",
			requestBody: models.CreatePollRequest{
				Title:       "Test Poll",
				Description: "Test description",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/polls", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreatePoll(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
				var resp models.CreatePollResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestAddOption(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	// Create a test poll
	pollID, _ := auth.GenerateID(16)
	adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'draft', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	tests := []struct {
		name           string
		pollID         string
		adminKey       string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.AddOptionResponse)
	}{
		{
			name:     "valid option addition",
			pollID:   pollID,
			adminKey: adminKey,
			requestBody: models.AddOptionRequest{
				Label: "Option A",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *models.AddOptionResponse) {
				if resp.OptionID == "" {
					t.Error("Expected non-empty option_id")
				}

				// Verify option was created
				var label string
				err := db.QueryRow("SELECT label FROM option WHERE id = $1", resp.OptionID).Scan(&label)
				if err != nil {
					t.Fatalf("Failed to query option: %v", err)
				}
				if label != "Option A" {
					t.Errorf("Expected label 'Option A', got '%s'", label)
				}
			},
		},
		{
			name:     "missing label",
			pollID:   pollID,
			adminKey: adminKey,
			requestBody: models.AddOptionRequest{
				Label: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid admin key",
			pollID:         pollID,
			adminKey:       "invalid-key",
			requestBody:    models.AddOptionRequest{Label: "Option B"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing admin key",
			pollID:         pollID,
			adminKey:       "",
			requestBody:    models.AddOptionRequest{Label: "Option C"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "poll not found",
			pollID:         "nonexistent",
			adminKey:       auth.GenerateAdminKey("nonexistent", cfg.AdminKeySalt),
			requestBody:    models.AddOptionRequest{Label: "Option D"},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest("POST", "/polls/"+tt.pollID+"/options", bytes.NewReader(body))
			req.SetPathValue("id", tt.pollID)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Admin-Key", tt.adminKey)
			w := httptest.NewRecorder()

			handler.AddOption(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
				var resp models.AddOptionResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestAddOptionToNonDraftPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	// Create a poll in 'open' status
	pollID, _ := auth.GenerateID(16)
	adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Open Poll', 'Alice', 'open', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	body, _ := json.Marshal(models.AddOptionRequest{Label: "Too Late Option"})
	req := httptest.NewRequest("POST", "/polls/"+pollID+"/options", bytes.NewReader(body))
	req.SetPathValue("id", pollID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", adminKey)
	w := httptest.NewRecorder()

	handler.AddOption(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestPublishPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	// Create a poll with options
	pollID, _ := auth.GenerateID(16)
	adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'draft', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	// Add two options
	for i, label := range []string{"Option A", "Option B"} {
		optionID, _ := auth.GenerateID(12)
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, optionID, pollID, label)
		if err != nil {
			t.Fatalf("Failed to create option %d: %v", i, err)
		}
	}

	tests := []struct {
		name           string
		pollID         string
		adminKey       string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.PublishPollResponse)
	}{
		{
			name:           "valid publish",
			pollID:         pollID,
			adminKey:       adminKey,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.PublishPollResponse) {
				if resp.ShareSlug == "" {
					t.Error("Expected non-empty share_slug")
				}
				if resp.ShareURL == "" {
					t.Error("Expected non-empty share_url")
				}

				// Verify poll status changed to 'open'
				var status string
				var shareSlug sql.NullString
				err := db.QueryRow("SELECT status, share_slug FROM poll WHERE id = $1", pollID).Scan(&status, &shareSlug)
				if err != nil {
					t.Fatalf("Failed to query poll: %v", err)
				}
				if status != models.StatusOpen {
					t.Errorf("Expected status 'open', got '%s'", status)
				}
				if !shareSlug.Valid || shareSlug.String != resp.ShareSlug {
					t.Error("Share slug mismatch in database")
				}

				// Verify slug is deterministic
				expectedSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
				if resp.ShareSlug != expectedSlug {
					t.Error("Share slug does not match expected value")
				}
			},
		},
		{
			name:           "invalid admin key",
			pollID:         pollID,
			adminKey:       "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "poll not found",
			pollID:         "nonexistent",
			adminKey:       auth.GenerateAdminKey("nonexistent", cfg.AdminKeySalt),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/polls/"+tt.pollID+"/publish", nil)
			req.SetPathValue("id", tt.pollID)
			req.Header.Set("X-Admin-Key", tt.adminKey)
			w := httptest.NewRecorder()

			handler.PublishPoll(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.PublishPollResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestPublishPollWithInsufficientOptions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	// Create a poll with only one option
	pollID, _ := auth.GenerateID(16)
	adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'draft', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	optionID, _ := auth.GenerateID(12)
	_, err = db.Exec(`
		INSERT INTO option (id, poll_id, label)
		VALUES ($1, $2, 'Only Option')
	`, optionID, pollID)
	if err != nil {
		t.Fatalf("Failed to create option: %v", err)
	}

	req := httptest.NewRequest("POST", "/polls/"+pollID+"/publish", nil)
	req.SetPathValue("id", pollID)
	req.Header.Set("X-Admin-Key", adminKey)
	w := httptest.NewRecorder()

	handler.PublishPoll(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestClosePoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	// Create an open poll
	pollID, _ := auth.GenerateID(16)
	adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	tests := []struct {
		name           string
		pollID         string
		adminKey       string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.ClosePollResponse)
	}{
		{
			name:           "valid close",
			pollID:         pollID,
			adminKey:       adminKey,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.ClosePollResponse) {
				if resp.ClosedAt.IsZero() {
					t.Error("Expected non-zero closed_at timestamp")
				}
				if resp.Snapshot.ID == "" {
					t.Error("Expected non-empty snapshot ID")
				}

				// Verify poll status changed to 'closed'
				var status string
				var closedAt sql.NullTime
				var snapshotID sql.NullString
				err := db.QueryRow("SELECT status, closed_at, final_snapshot_id FROM poll WHERE id = $1", pollID).Scan(&status, &closedAt, &snapshotID)
				if err != nil {
					t.Fatalf("Failed to query poll: %v", err)
				}
				if status != models.StatusClosed {
					t.Errorf("Expected status 'closed', got '%s'", status)
				}
				if !closedAt.Valid {
					t.Error("Expected closed_at to be set")
				}
				if !snapshotID.Valid {
					t.Error("Expected final_snapshot_id to be set")
				}

				// Verify snapshot was created
				var snapshotExists bool
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM result_snapshot WHERE id = $1)", resp.Snapshot.ID).Scan(&snapshotExists)
				if err != nil {
					t.Fatalf("Failed to check snapshot: %v", err)
				}
				if !snapshotExists {
					t.Error("Snapshot was not created in database")
				}
			},
		},
		{
			name:           "invalid admin key",
			pollID:         pollID,
			adminKey:       "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "poll not found",
			pollID:         "nonexistent",
			adminKey:       auth.GenerateAdminKey("nonexistent", cfg.AdminKeySalt),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/polls/"+tt.pollID+"/close", nil)
			req.SetPathValue("id", tt.pollID)
			req.Header.Set("X-Admin-Key", tt.adminKey)
			w := httptest.NewRecorder()

			handler.ClosePoll(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.ClosePollResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestCloseDraftPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db, cfg)

	// Create a draft poll
	pollID, _ := auth.GenerateID(16)
	adminKey := auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Draft Poll', 'Alice', 'draft', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	req := httptest.NewRequest("POST", "/polls/"+pollID+"/close", nil)
	req.SetPathValue("id", pollID)
	req.Header.Set("X-Admin-Key", adminKey)
	w := httptest.NewRecorder()

	handler.ClosePoll(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}
