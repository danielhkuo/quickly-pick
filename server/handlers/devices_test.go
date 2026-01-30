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
	"github.com/danielhkuo/quickly-pick/models"
)

// setupTestDBWithDevices extends setupTestDB to include device tables
func setupTestDBWithDevices(t *testing.T) *testDB {
	db, err := sql.Open("postgres", "postgres://quicklypick:devpassword@localhost:5432/quickly_pick_dev?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Clean up all tables including device tables before each test
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

	// Create schema including device tables
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

		CREATE TABLE device (
			id TEXT PRIMARY KEY,
			device_uuid TEXT NOT NULL UNIQUE,
			platform TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			last_seen_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE device_poll (
			device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
			poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
			voter_token TEXT,
			role TEXT NOT NULL DEFAULT 'voter',
			linked_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (device_id, poll_id)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return &testDB{DB: db}
}

type testDB struct {
	*sql.DB
}

func TestDeviceRegister(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewDeviceHandler(db.DB, cfg)

	tests := []struct {
		name           string
		deviceUUID     string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.RegisterDeviceResponse)
	}{
		{
			name:       "new device registration",
			deviceUUID: "test-uuid-123",
			requestBody: models.RegisterDeviceRequest{
				Platform: "ios",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *models.RegisterDeviceResponse) {
				if resp.DeviceID == "" {
					t.Error("Expected non-empty device_id")
				}
				if !resp.IsNew {
					t.Error("Expected is_new to be true for new device")
				}

				// Verify device was created in database
				var platform string
				err := db.QueryRow("SELECT platform FROM device WHERE id = $1", resp.DeviceID).Scan(&platform)
				if err != nil {
					t.Fatalf("Failed to query device: %v", err)
				}
				if platform != "ios" {
					t.Errorf("Expected platform 'ios', got '%s'", platform)
				}
			},
		},
		{
			name:       "existing device registration",
			deviceUUID: "existing-uuid-456",
			requestBody: models.RegisterDeviceRequest{
				Platform: "android",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.RegisterDeviceResponse) {
				if resp.DeviceID == "" {
					t.Error("Expected non-empty device_id")
				}
				if resp.IsNew {
					t.Error("Expected is_new to be false for existing device")
				}
			},
		},
		{
			name:           "missing X-Device-UUID header",
			deviceUUID:     "",
			requestBody:    models.RegisterDeviceRequest{Platform: "ios"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid platform",
			deviceUUID:     "test-uuid-789",
			requestBody:    models.RegisterDeviceRequest{Platform: "windows"},
			expectedStatus: http.StatusBadRequest,
		},
	}

	// Pre-create existing device for "existing device" test case
	existingDeviceID, _ := auth.GenerateID(16)
	_, err := db.Exec(`
		INSERT INTO device (id, device_uuid, platform, created_at, last_seen_at)
		VALUES ($1, 'existing-uuid-456', 'android', $2, $2)
	`, existingDeviceID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create existing device: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest("POST", "/devices/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.deviceUUID != "" {
				req.Header.Set("X-Device-UUID", tt.deviceUUID)
			}
			w := httptest.NewRecorder()

			handler.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if (tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusOK) && tt.checkResponse != nil {
				var resp models.RegisterDeviceResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestDeviceGetMe(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewDeviceHandler(db.DB, cfg)

	// Create a device
	deviceID, _ := auth.GenerateID(16)
	deviceUUID := "get-me-test-uuid"
	_, err := db.Exec(`
		INSERT INTO device (id, device_uuid, platform, created_at, last_seen_at)
		VALUES ($1, $2, 'macos', $3, $3)
	`, deviceID, deviceUUID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	tests := []struct {
		name           string
		deviceUUID     string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.DeviceInfo)
	}{
		{
			name:           "get existing device",
			deviceUUID:     deviceUUID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.DeviceInfo) {
				if resp.ID != deviceID {
					t.Errorf("Expected device ID '%s', got '%s'", deviceID, resp.ID)
				}
				if resp.Platform != "macos" {
					t.Errorf("Expected platform 'macos', got '%s'", resp.Platform)
				}
			},
		},
		{
			name:           "device not found",
			deviceUUID:     "nonexistent-uuid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing header",
			deviceUUID:     "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/devices/me", nil)
			if tt.deviceUUID != "" {
				req.Header.Set("X-Device-UUID", tt.deviceUUID)
			}
			w := httptest.NewRecorder()

			handler.GetMe(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.DeviceInfo
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestDeviceGetMyPolls(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewDeviceHandler(db.DB, cfg)

	// Create a device
	deviceID, _ := auth.GenerateID(16)
	deviceUUID := "my-polls-test-uuid"
	_, err := db.Exec(`
		INSERT INTO device (id, device_uuid, platform, created_at, last_seen_at)
		VALUES ($1, $2, 'ios', $3, $3)
	`, deviceID, deviceUUID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	// Create polls and link to device
	poll1ID, _ := auth.GenerateID(16)
	shareSlug1 := auth.GenerateShareSlug(poll1ID, cfg.PollSlugSalt)
	_, err = db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Admin Poll', 'Alice', 'open', $2, $3)
	`, poll1ID, shareSlug1, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll 1: %v", err)
	}

	poll2ID, _ := auth.GenerateID(16)
	shareSlug2 := auth.GenerateShareSlug(poll2ID, cfg.PollSlugSalt)
	_, err = db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Voter Poll', 'Bob', 'open', $2, $3)
	`, poll2ID, shareSlug2, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll 2: %v", err)
	}

	// Link device as admin to poll 1
	_, err = db.Exec(`
		INSERT INTO device_poll (device_id, poll_id, role, linked_at)
		VALUES ($1, $2, 'admin', $3)
	`, deviceID, poll1ID, time.Now())
	if err != nil {
		t.Fatalf("Failed to link device to poll 1: %v", err)
	}

	// Create username claim and link device as voter to poll 2
	voterToken, _ := auth.GenerateVoterToken()
	_, err = db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, 'TestUser', $2, $3)
	`, poll2ID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create username claim: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO device_poll (device_id, poll_id, voter_token, role, linked_at)
		VALUES ($1, $2, $3, 'voter', $4)
	`, deviceID, poll2ID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to link device to poll 2: %v", err)
	}

	tests := []struct {
		name           string
		deviceUUID     string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.GetMyPollsResponse)
	}{
		{
			name:           "get polls for device",
			deviceUUID:     deviceUUID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.GetMyPollsResponse) {
				if len(resp.Polls) != 2 {
					t.Fatalf("Expected 2 polls, got %d", len(resp.Polls))
				}

				// Check for admin poll
				foundAdmin := false
				foundVoter := false
				for _, p := range resp.Polls {
					if p.Role == "admin" && p.Title == "Admin Poll" {
						foundAdmin = true
					}
					if p.Role == "voter" && p.Title == "Voter Poll" {
						foundVoter = true
						if p.Username == nil || *p.Username != "TestUser" {
							t.Error("Expected username 'TestUser' for voter poll")
						}
					}
				}
				if !foundAdmin {
					t.Error("Expected to find admin poll")
				}
				if !foundVoter {
					t.Error("Expected to find voter poll")
				}
			},
		},
		{
			name:           "device not found",
			deviceUUID:     "nonexistent-uuid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing header",
			deviceUUID:     "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/devices/my-polls", nil)
			if tt.deviceUUID != "" {
				req.Header.Set("X-Device-UUID", tt.deviceUUID)
			}
			w := httptest.NewRecorder()

			handler.GetMyPolls(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.GetMyPollsResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestGetOrCreateDevice(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	// Test with no header
	req := httptest.NewRequest("GET", "/test", nil)
	deviceID, err := GetOrCreateDevice(db.DB, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deviceID != "" {
		t.Error("Expected empty device ID when no header")
	}

	// Test creating new device
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Device-UUID", "auto-create-uuid")
	deviceID, err = GetOrCreateDevice(db.DB, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deviceID == "" {
		t.Error("Expected non-empty device ID")
	}

	// Verify device was created with 'web' platform
	var platform string
	err = db.QueryRow("SELECT platform FROM device WHERE id = $1", deviceID).Scan(&platform)
	if err != nil {
		t.Fatalf("Failed to query device: %v", err)
	}
	if platform != "web" {
		t.Errorf("Expected platform 'web', got '%s'", platform)
	}

	// Test finding existing device
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Device-UUID", "auto-create-uuid")
	deviceID2, err := GetOrCreateDevice(db.DB, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deviceID2 != deviceID {
		t.Error("Expected same device ID for same UUID")
	}
}

func TestLinkDeviceToPoll(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()

	// Create a device
	deviceID, _ := auth.GenerateID(16)
	_, err := db.Exec(`
		INSERT INTO device (id, device_uuid, platform, created_at, last_seen_at)
		VALUES ($1, 'link-test-uuid', 'ios', $2, $2)
	`, deviceID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	// Create a poll
	pollID, _ := auth.GenerateID(16)
	_, err = db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'draft', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Test linking as admin
	err = LinkDeviceToPoll(db.DB, deviceID, pollID, models.RoleAdmin, nil)
	if err != nil {
		t.Fatalf("Failed to link device as admin: %v", err)
	}

	var role string
	err = db.QueryRow("SELECT role FROM device_poll WHERE device_id = $1 AND poll_id = $2", deviceID, pollID).Scan(&role)
	if err != nil {
		t.Fatalf("Failed to query device_poll: %v", err)
	}
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}

	// Test linking again (should not downgrade from admin)
	voterToken := "test-voter-token"
	err = LinkDeviceToPoll(db.DB, deviceID, pollID, models.RoleVoter, &voterToken)
	if err != nil {
		t.Fatalf("Failed to re-link device: %v", err)
	}

	err = db.QueryRow("SELECT role FROM device_poll WHERE device_id = $1 AND poll_id = $2", deviceID, pollID).Scan(&role)
	if err != nil {
		t.Fatalf("Failed to query device_poll: %v", err)
	}
	if role != "admin" {
		t.Errorf("Expected role to remain 'admin', got '%s'", role)
	}

	// Test empty deviceID (should be no-op)
	err = LinkDeviceToPoll(db.DB, "", pollID, models.RoleVoter, nil)
	if err != nil {
		t.Errorf("Expected no error for empty deviceID, got %v", err)
	}

	_ = cfg // Satisfy unused variable
}

func TestPreviewEndpoint(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db.DB, cfg)

	// Create a poll with options and ballots
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Preview Test Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Add options
	for _, label := range []string{"Option A", "Option B", "Option C"} {
		optionID, _ := auth.GenerateID(12)
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, optionID, pollID, label)
		if err != nil {
			t.Fatalf("Failed to create option: %v", err)
		}
	}

	// Add ballots
	for i := 0; i < 5; i++ {
		ballotID, _ := auth.GenerateID(16)
		voterToken, _ := auth.GenerateVoterToken()
		_, err := db.Exec(`
			INSERT INTO ballot (id, poll_id, voter_token, submitted_at)
			VALUES ($1, $2, $3, $4)
		`, ballotID, pollID, voterToken, time.Now())
		if err != nil {
			t.Fatalf("Failed to create ballot: %v", err)
		}
	}

	tests := []struct {
		name           string
		slug           string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.PollPreviewResponse)
	}{
		{
			name:           "valid preview",
			slug:           shareSlug,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.PollPreviewResponse) {
				if resp.Title != "Preview Test Poll" {
					t.Errorf("Expected title 'Preview Test Poll', got '%s'", resp.Title)
				}
				if resp.Status != "open" {
					t.Errorf("Expected status 'open', got '%s'", resp.Status)
				}
				if resp.OptionCount != 3 {
					t.Errorf("Expected option_count 3, got %d", resp.OptionCount)
				}
				if resp.BallotCount != 5 {
					t.Errorf("Expected ballot_count 5, got %d", resp.BallotCount)
				}
			},
		},
		{
			name:           "poll not found",
			slug:           "nonexistent-slug",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/polls/"+tt.slug+"/preview", nil)
			req.SetPathValue("slug", tt.slug)
			w := httptest.NewRecorder()

			handler.GetPreview(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.PollPreviewResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestCreatePollWithDeviceLinking(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewPollHandler(db.DB, cfg)

	// Test creating poll with device UUID
	deviceUUID := "create-poll-device-uuid"
	body, _ := json.Marshal(models.CreatePollRequest{
		Title:       "Device Linked Poll",
		Description: "Testing device linking",
		CreatorName: "Alice",
	})

	req := httptest.NewRequest("POST", "/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-UUID", deviceUUID)
	w := httptest.NewRecorder()

	handler.CreatePoll(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp models.CreatePollResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify device was created and linked
	var deviceID string
	err := db.QueryRow("SELECT id FROM device WHERE device_uuid = $1", deviceUUID).Scan(&deviceID)
	if err != nil {
		t.Fatalf("Device not created: %v", err)
	}

	var role string
	err = db.QueryRow("SELECT role FROM device_poll WHERE device_id = $1 AND poll_id = $2", deviceID, resp.PollID).Scan(&role)
	if err != nil {
		t.Fatalf("Device not linked to poll: %v", err)
	}
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
}

func TestClaimUsernameWithDeviceLinking(t *testing.T) {
	db := setupTestDBWithDevices(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db.DB, cfg)

	// Create an open poll
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Voting Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Test claiming username with device UUID
	deviceUUID := "claim-username-device-uuid"
	body, _ := json.Marshal(models.ClaimUsernameRequest{
		Username: "TestVoter",
	})

	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-UUID", deviceUUID)
	w := httptest.NewRecorder()

	handler.ClaimUsername(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp models.ClaimUsernameResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify device was created and linked
	var deviceID string
	err = db.QueryRow("SELECT id FROM device WHERE device_uuid = $1", deviceUUID).Scan(&deviceID)
	if err != nil {
		t.Fatalf("Device not created: %v", err)
	}

	var role string
	var voterToken string
	err = db.QueryRow("SELECT role, voter_token FROM device_poll WHERE device_id = $1 AND poll_id = $2", deviceID, pollID).Scan(&role, &voterToken)
	if err != nil {
		t.Fatalf("Device not linked to poll: %v", err)
	}
	if role != "voter" {
		t.Errorf("Expected role 'voter', got '%s'", role)
	}
	if voterToken != resp.VoterToken {
		t.Error("Expected voter_token to match response")
	}
}
