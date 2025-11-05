// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/danielhkuo/quickly-pick/auth"
	"github.com/danielhkuo/quickly-pick/models"
)

func TestGetPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create a poll with options
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, description, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Test Poll', 'Test description', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	// Add options
	optionID1, _ := auth.GenerateID(12)
	optionID2, _ := auth.GenerateID(12)
	for _, opt := range []struct {
		id    string
		label string
	}{
		{optionID1, "Option A"},
		{optionID2, "Option B"},
	} {
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, opt.id, pollID, opt.label)
		if err != nil {
			t.Fatalf("Failed to create option: %v", err)
		}
	}

	tests := []struct {
		name           string
		shareSlug      string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.PollWithOptions)
	}{
		{
			name:           "valid poll retrieval",
			shareSlug:      shareSlug,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.PollWithOptions) {
				if resp.Poll.ID != pollID {
					t.Errorf("Expected poll ID %s, got %s", pollID, resp.Poll.ID)
				}
				if resp.Poll.Title != "Test Poll" {
					t.Errorf("Expected title 'Test Poll', got '%s'", resp.Poll.Title)
				}
				if resp.Poll.Description != "Test description" {
					t.Errorf("Expected description 'Test description', got '%s'", resp.Poll.Description)
				}
				if resp.Poll.CreatorName != "Alice" {
					t.Errorf("Expected creator 'Alice', got '%s'", resp.Poll.CreatorName)
				}
				if resp.Poll.Status != models.StatusOpen {
					t.Errorf("Expected status 'open', got '%s'", resp.Poll.Status)
				}

				if len(resp.Options) != 2 {
					t.Fatalf("Expected 2 options, got %d", len(resp.Options))
				}

				// Verify options
				optionLabels := make(map[string]string)
				for _, opt := range resp.Options {
					optionLabels[opt.ID] = opt.Label
				}
				if optionLabels[optionID1] != "Option A" {
					t.Error("Option A label mismatch")
				}
				if optionLabels[optionID2] != "Option B" {
					t.Error("Option B label mismatch")
				}
			},
		},
		{
			name:           "poll not found",
			shareSlug:      "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/polls/"+tt.shareSlug, nil)
			req.SetPathValue("slug", tt.shareSlug)
			w := httptest.NewRecorder()

			handler.GetPoll(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.PollWithOptions
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestGetResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create a closed poll with results
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	snapshotID, _ := auth.GenerateID(16)

	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, final_snapshot_id, created_at)
		VALUES ($1, 'Closed Poll', 'Alice', 'closed', $2, $3, $4)
	`, pollID, shareSlug, snapshotID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	// Create options
	optionID1, _ := auth.GenerateID(12)
	optionID2, _ := auth.GenerateID(12)
	for _, opt := range []struct {
		id    string
		label string
	}{
		{optionID1, "Option A"},
		{optionID2, "Option B"},
	} {
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, opt.id, pollID, opt.label)
		if err != nil {
			t.Fatalf("Failed to create option: %v", err)
		}
	}

	// Create snapshot with mock results
	payload := map[string]interface{}{
		"rankings": []models.OptionStats{
			{
				OptionID: optionID1,
				Label:    "Option A",
				Median:   0.8,
				P10:      0.5,
				P90:      0.95,
				Mean:     0.75,
				NegShare: 0.1,
				Veto:     false,
				Rank:     1,
			},
			{
				OptionID: optionID2,
				Label:    "Option B",
				Median:   0.3,
				P10:      0.1,
				P90:      0.5,
				Mean:     0.35,
				NegShare: 0.4,
				Veto:     false,
				Rank:     2,
			},
		},
		"inputs_hash": "test-hash",
	}
	payloadJSON, _ := json.Marshal(payload)

	_, err = db.Exec(`
		INSERT INTO result_snapshot (id, poll_id, method, computed_at, payload)
		VALUES ($1, $2, 'bmj', $3, $4)
	`, snapshotID, pollID, time.Now(), payloadJSON)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	tests := []struct {
		name           string
		shareSlug      string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.ResultSnapshot)
	}{
		{
			name:           "valid results retrieval",
			shareSlug:      shareSlug,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *models.ResultSnapshot) {
				if resp.ID != snapshotID {
					t.Errorf("Expected snapshot ID %s, got %s", snapshotID, resp.ID)
				}
				if resp.PollID != pollID {
					t.Errorf("Expected poll ID %s, got %s", pollID, resp.PollID)
				}
				if resp.Method != models.MethodBMJ {
					t.Errorf("Expected method 'bmj', got '%s'", resp.Method)
				}
				if resp.InputsHash != "test-hash" {
					t.Errorf("Expected inputs_hash 'test-hash', got '%s'", resp.InputsHash)
				}

				if len(resp.Rankings) != 2 {
					t.Fatalf("Expected 2 rankings, got %d", len(resp.Rankings))
				}

				// Verify first ranking
				rank1 := resp.Rankings[0]
				if rank1.OptionID != optionID1 {
					t.Errorf("Expected first option ID %s, got %s", optionID1, rank1.OptionID)
				}
				if rank1.Rank != 1 {
					t.Errorf("Expected rank 1, got %d", rank1.Rank)
				}
				if rank1.Median != 0.8 {
					t.Errorf("Expected median 0.8, got %f", rank1.Median)
				}

				// Verify second ranking
				rank2 := resp.Rankings[1]
				if rank2.OptionID != optionID2 {
					t.Errorf("Expected second option ID %s, got %s", optionID2, rank2.OptionID)
				}
				if rank2.Rank != 2 {
					t.Errorf("Expected rank 2, got %d", rank2.Rank)
				}
			},
		},
		{
			name:           "poll not found",
			shareSlug:      "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/polls/"+tt.shareSlug+"/results", nil)
			req.SetPathValue("slug", tt.shareSlug)
			w := httptest.NewRecorder()

			handler.GetResults(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp models.ResultSnapshot
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestGetResultsForOpenPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create an open poll
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Open Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/results", nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()

	handler.GetResults(w, req)

	// Results should be sealed (403 Forbidden) for open polls
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for open poll, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetResultsForDraftPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create a draft poll
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Draft Poll', 'Alice', 'draft', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/results", nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()

	handler.GetResults(w, req)

	// Results should be sealed (403 Forbidden) for draft polls
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for draft poll, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetBallotCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create a poll with ballots
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	// Add 3 ballots
	for i := 0; i < 3; i++ {
		ballotID, _ := auth.GenerateID(16)
		voterToken, _ := auth.GenerateVoterToken()
		_, err := db.Exec(`
			INSERT INTO ballot (id, poll_id, voter_token, submitted_at)
			VALUES ($1, $2, $3, $4)
		`, ballotID, pollID, voterToken, time.Now())
		if err != nil {
			t.Fatalf("Failed to create ballot %d: %v", i, err)
		}
	}

	tests := []struct {
		name           string
		shareSlug      string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "valid ballot count",
			shareSlug:      shareSlug,
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "poll not found",
			shareSlug:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/polls/"+tt.shareSlug+"/ballot-count", nil)
			req.SetPathValue("slug", tt.shareSlug)
			w := httptest.NewRecorder()

			handler.GetBallotCount(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]int
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				count, ok := resp["ballot_count"]
				if !ok {
					t.Error("Response missing 'ballot_count' field")
				}
				if count != tt.expectedCount {
					t.Errorf("Expected ballot count %d, got %d", tt.expectedCount, count)
				}
			}
		})
	}
}

func TestGetBallotCountForPollWithNoBallots(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create a poll without ballots
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Empty Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/ballot-count", nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()

	handler.GetBallotCount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	count := resp["ballot_count"]
	if count != 0 {
		t.Errorf("Expected ballot count 0, got %d", count)
	}
}

func TestGetPollWithoutOptions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewResultsHandler(db, cfg)

	// Create a poll without options
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
    INSERT INTO poll (id, title, description, creator_name, status, share_slug, created_at)
    VALUES ($1, 'Empty Poll', '', 'Alice', 'draft', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	req := httptest.NewRequest("GET", "/polls/"+shareSlug, nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()

	handler.GetPoll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.PollWithOptions
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Options) != 0 {
		t.Errorf("Expected 0 options, got %d", len(resp.Options))
	}
}
