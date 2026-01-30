// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package testutil

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
	_ "github.com/lib/pq"
)

// TestDBURL is the connection string for the test database
const TestDBURL = "postgres://quicklypick:devpassword@localhost:5432/quickly_pick_dev?sslmode=disable"

// SetupTestDB creates a fresh test database with the full schema
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", TestDBURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Clean up tables before each test
	_, err = db.Exec(`
		DROP TABLE IF EXISTS device_poll CASCADE;
		DROP TABLE IF EXISTS device CASCADE;
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

	// Create full schema
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

		CREATE INDEX idx_poll_share_slug ON poll(share_slug);
		CREATE INDEX idx_poll_status ON poll(status);

		CREATE TABLE option (
			id TEXT PRIMARY KEY,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			label TEXT NOT NULL
		);

		CREATE INDEX idx_option_poll_id ON option(poll_id);

		CREATE TABLE username_claim (
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			username TEXT NOT NULL,
			voter_token TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (poll_id, voter_token),
			UNIQUE (poll_id, username)
		);

		CREATE INDEX idx_username_claim_poll_id ON username_claim(poll_id);

		CREATE TABLE ballot (
			id TEXT PRIMARY KEY,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			voter_token TEXT NOT NULL,
			submitted_at TIMESTAMP NOT NULL DEFAULT NOW(),
			ip_hash TEXT,
			user_agent TEXT,
			UNIQUE (poll_id, voter_token)
		);

		CREATE INDEX idx_ballot_poll_id ON ballot(poll_id);
		CREATE INDEX idx_ballot_voter_token ON ballot(poll_id, voter_token);

		CREATE TABLE score (
			ballot_id TEXT NOT NULL REFERENCES ballot(id) ON DELETE CASCADE,
			option_id TEXT NOT NULL REFERENCES option(id) ON DELETE CASCADE,
			value01 REAL NOT NULL CHECK (value01 >= 0 AND value01 <= 1),
			PRIMARY KEY (ballot_id, option_id)
		);

		CREATE INDEX idx_score_option_id ON score(option_id);

		CREATE TABLE result_snapshot (
			id TEXT PRIMARY KEY,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			method TEXT NOT NULL,
			computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
			payload JSONB NOT NULL
		);

		CREATE INDEX idx_result_snapshot_poll_id ON result_snapshot(poll_id);

		CREATE TABLE device (
			id TEXT PRIMARY KEY,
			device_uuid TEXT NOT NULL UNIQUE,
			platform TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			last_seen_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX idx_device_uuid ON device(device_uuid);

		CREATE TABLE device_poll (
			device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			voter_token TEXT,
			role TEXT NOT NULL DEFAULT 'voter',
			linked_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (device_id, poll_id)
		);

		CREATE INDEX idx_device_poll_device ON device_poll(device_id);
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// GetTestConfig returns a standard test configuration
func GetTestConfig() cliparse.Config {
	return cliparse.Config{
		Port:         3318,
		DatabaseURL:  TestDBURL,
		AdminKeySalt: "test-admin-salt",
		PollSlugSalt: "test-slug-salt",
	}
}

// CreateTestPoll creates a poll in the database and returns its ID and admin key
// status should be "draft", "open", or "closed"
func CreateTestPoll(t *testing.T, db *sql.DB, cfg cliparse.Config, status string) (pollID, adminKey, shareSlug string) {
	t.Helper()

	pollID, _ = auth.GenerateID(16)
	adminKey = auth.GenerateAdminKey(pollID, cfg.AdminKeySalt)

	var slug *string
	if status == "open" || status == "closed" {
		s := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
		slug = &s
		shareSlug = s
	}

	var closedAt *time.Time
	if status == "closed" {
		now := time.Now()
		closedAt = &now
	}

	_, err := db.Exec(`
		INSERT INTO poll (id, title, description, creator_name, status, share_slug, closed_at, created_at)
		VALUES ($1, 'Test Poll', 'A test poll', 'TestUser', $2, $3, $4, $5)
	`, pollID, status, slug, closedAt, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	return pollID, adminKey, shareSlug
}

// AddTestOption adds an option to a poll and returns the option ID
func AddTestOption(t *testing.T, db *sql.DB, pollID, label string) string {
	t.Helper()

	optionID, _ := auth.GenerateID(12)
	_, err := db.Exec(`
		INSERT INTO option (id, poll_id, label)
		VALUES ($1, $2, $3)
	`, optionID, pollID, label)
	if err != nil {
		t.Fatalf("Failed to create test option: %v", err)
	}

	return optionID
}

// CreateTestVoter claims a username for a poll and returns the voter token
func CreateTestVoter(t *testing.T, db *sql.DB, pollID, username string) string {
	t.Helper()

	voterToken, _ := auth.GenerateVoterToken()
	_, err := db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, $2, $3, $4)
	`, pollID, username, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test voter: %v", err)
	}

	return voterToken
}

// SubmitTestBallot creates a ballot with scores for a voter
func SubmitTestBallot(t *testing.T, db *sql.DB, pollID, voterToken string, scores map[string]float64) string {
	t.Helper()

	ballotID, _ := auth.GenerateID(16)
	_, err := db.Exec(`
		INSERT INTO ballot (id, poll_id, voter_token, submitted_at)
		VALUES ($1, $2, $3, $4)
	`, ballotID, pollID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test ballot: %v", err)
	}

	for optionID, value := range scores {
		_, err := db.Exec(`
			INSERT INTO score (ballot_id, option_id, value01)
			VALUES ($1, $2, $3)
		`, ballotID, optionID, value)
		if err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	return ballotID
}

// MakeRequest creates an HTTP test request
func MakeRequest(method, path string, body interface{}, headers map[string]string) *http.Request {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req
}

// AssertStatus checks that the response has the expected status code
func AssertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}

// AssertJSON decodes the response body into the provided struct
func AssertJSON(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}
}
