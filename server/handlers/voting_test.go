package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielhkuo/quickly-pick/auth"
	"github.com/danielhkuo/quickly-pick/models"
)

func TestClaimUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db, cfg)

	// Create an open poll
	pollID, _ := auth.GenerateID(16)
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
		shareSlug      string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.ClaimUsernameResponse)
	}{
		{
			name:      "valid username claim",
			shareSlug: shareSlug,
			requestBody: models.ClaimUsernameRequest{
				Username: "bob",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *models.ClaimUsernameResponse) {
				if resp.VoterToken == "" {
					t.Error("Expected non-empty voter_token")
				}

				// Verify username claim was created
				var exists bool
				err := db.QueryRow(`
					SELECT EXISTS(
						SELECT 1 FROM username_claim
						WHERE poll_id = $1 AND username = $2
					)
				`, pollID, "bob").Scan(&exists)
				if err != nil {
					t.Fatalf("Failed to check username claim: %v", err)
				}
				if !exists {
					t.Error("Username claim was not created in database")
				}

				// Verify voter token matches
				var storedToken string
				err = db.QueryRow(`
					SELECT voter_token FROM username_claim
					WHERE poll_id = $1 AND username = $2
				`, pollID, "bob").Scan(&storedToken)
				if err != nil {
					t.Fatalf("Failed to query voter token: %v", err)
				}
				if storedToken != resp.VoterToken {
					t.Error("Voter token mismatch")
				}
			},
		},
		{
			name:      "missing username",
			shareSlug: shareSlug,
			requestBody: models.ClaimUsernameRequest{
				Username: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "username too short",
			shareSlug: shareSlug,
			requestBody: models.ClaimUsernameRequest{
				Username: "a",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "username too long",
			shareSlug: shareSlug,
			requestBody: models.ClaimUsernameRequest{
				Username: "this_is_a_very_long_username_that_exceeds_fifty_characters_limit",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "poll not found",
			shareSlug:      "nonexistent-slug",
			requestBody:    models.ClaimUsernameRequest{Username: "charlie"},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest("POST", "/polls/"+tt.shareSlug+"/claim-username", bytes.NewReader(body))
			req.SetPathValue("slug", tt.shareSlug)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ClaimUsername(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
				var resp models.ClaimUsernameResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestClaimDuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db, cfg)

	// Create an open poll
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'open', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	// Claim a username
	voterToken, _ := auth.GenerateVoterToken()
	_, err = db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, 'existinguser', $2, $3)
	`, pollID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create username claim: %v", err)
	}

	// Try to claim the same username again
	body, _ := json.Marshal(models.ClaimUsernameRequest{Username: "existinguser"})
	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ClaimUsername(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestClaimUsernameForClosedPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db, cfg)

	// Create a closed poll
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Closed Poll', 'Alice', 'closed', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	body, _ := json.Marshal(models.ClaimUsernameRequest{Username: "toolate"})
	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ClaimUsername(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestSubmitBallot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db, cfg)

	// Create an open poll with options
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'open', $2, $3)
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

	// Claim a username
	voterToken, _ := auth.GenerateVoterToken()
	_, err = db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, 'voter1', $2, $3)
	`, pollID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create username claim: %v", err)
	}

	tests := []struct {
		name           string
		shareSlug      string
		voterToken     string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *models.SubmitBallotResponse)
	}{
		{
			name:       "valid ballot submission",
			shareSlug:  shareSlug,
			voterToken: voterToken,
			requestBody: models.SubmitBallotRequest{
				Scores: map[string]float64{
					optionID1: 0.8,
					optionID2: 0.3,
				},
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *models.SubmitBallotResponse) {
				if resp.BallotID == "" {
					t.Error("Expected non-empty ballot_id")
				}

				// Verify ballot was created
				var ballotExists bool
				err := db.QueryRow(`
					SELECT EXISTS(
						SELECT 1 FROM ballot
						WHERE id = $1 AND poll_id = $2 AND voter_token = $3
					)
				`, resp.BallotID, pollID, voterToken).Scan(&ballotExists)
				if err != nil {
					t.Fatalf("Failed to check ballot: %v", err)
				}
				if !ballotExists {
					t.Error("Ballot was not created in database")
				}

				// Verify scores were created
				rows, err := db.Query(`
					SELECT option_id, value01 FROM score WHERE ballot_id = $1
				`, resp.BallotID)
				if err != nil {
					t.Fatalf("Failed to query scores: %v", err)
				}
				defer rows.Close()

				scores := make(map[string]float64)
				for rows.Next() {
					var optionID string
					var value float64
					if err := rows.Scan(&optionID, &value); err != nil {
						t.Fatalf("Failed to scan score: %v", err)
					}
					scores[optionID] = value
				}

				if len(scores) != 2 {
					t.Errorf("Expected 2 scores, got %d", len(scores))
				}
				if scores[optionID1] != 0.8 {
					t.Errorf("Expected score 0.8 for option1, got %f", scores[optionID1])
				}
				if scores[optionID2] != 0.3 {
					t.Errorf("Expected score 0.3 for option2, got %f", scores[optionID2])
				}
			},
		},
		{
			name:       "score out of range (too high)",
			shareSlug:  shareSlug,
			voterToken: voterToken,
			requestBody: models.SubmitBallotRequest{
				Scores: map[string]float64{
					optionID1: 1.5,
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "score out of range (negative)",
			shareSlug:  shareSlug,
			voterToken: voterToken,
			requestBody: models.SubmitBallotRequest{
				Scores: map[string]float64{
					optionID1: -0.1,
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid option ID",
			shareSlug:  shareSlug,
			voterToken: voterToken,
			requestBody: models.SubmitBallotRequest{
				Scores: map[string]float64{
					"invalid-option-id": 0.5,
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "empty scores",
			shareSlug:  shareSlug,
			voterToken: voterToken,
			requestBody: models.SubmitBallotRequest{
				Scores: map[string]float64{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing voter token",
			shareSlug:      shareSlug,
			voterToken:     "",
			requestBody:    models.SubmitBallotRequest{Scores: map[string]float64{optionID1: 0.5}},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid voter token",
			shareSlug:      shareSlug,
			voterToken:     "invalid-token",
			requestBody:    models.SubmitBallotRequest{Scores: map[string]float64{optionID1: 0.5}},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "poll not found",
			shareSlug:      "nonexistent",
			voterToken:     voterToken,
			requestBody:    models.SubmitBallotRequest{Scores: map[string]float64{optionID1: 0.5}},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest("POST", "/polls/"+tt.shareSlug+"/ballots", bytes.NewReader(body))
			req.SetPathValue("slug", tt.shareSlug)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Voter-Token", tt.voterToken)
			w := httptest.NewRecorder()

			handler.SubmitBallot(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
				var resp models.SubmitBallotResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestUpdateBallot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db, cfg)

	// Create an open poll with options
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Test Poll', 'Alice', 'open', $2, $3)
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

	// Claim a username and submit initial ballot
	voterToken, _ := auth.GenerateVoterToken()
	_, err = db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, 'voter1', $2, $3)
	`, pollID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create username claim: %v", err)
	}

	ballotID, _ := auth.GenerateID(16)
	_, err = db.Exec(`
		INSERT INTO ballot (id, poll_id, voter_token, submitted_at)
		VALUES ($1, $2, $3, $4)
	`, ballotID, pollID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create ballot: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO score (ballot_id, option_id, value01)
		VALUES ($1, $2, 0.5), ($1, $3, 0.5)
	`, ballotID, optionID1, optionID2)
	if err != nil {
		t.Fatalf("Failed to create scores: %v", err)
	}

	// Submit updated ballot
	body, _ := json.Marshal(models.SubmitBallotRequest{
		Scores: map[string]float64{
			optionID1: 0.9,
			optionID2: 0.2,
		},
	})

	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Voter-Token", voterToken)
	w := httptest.NewRecorder()

	handler.SubmitBallot(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp models.SubmitBallotResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify ballot ID is the same (update, not insert)
	if resp.BallotID != ballotID {
		t.Errorf("Expected ballot ID to remain %s, got %s", ballotID, resp.BallotID)
	}

	// Verify message indicates update
	if resp.Message != "Ballot updated successfully" {
		t.Errorf("Expected update message, got: %s", resp.Message)
	}

	// Verify scores were updated
	rows, err := db.Query(`
		SELECT option_id, value01 FROM score WHERE ballot_id = $1
	`, ballotID)
	if err != nil {
		t.Fatalf("Failed to query scores: %v", err)
	}
	defer rows.Close()

	scores := make(map[string]float64)
	for rows.Next() {
		var optionID string
		var value float64
		if err := rows.Scan(&optionID, &value); err != nil {
			t.Fatalf("Failed to scan score: %v", err)
		}
		scores[optionID] = value
	}

	if scores[optionID1] != 0.9 {
		t.Errorf("Expected updated score 0.9, got %f", scores[optionID1])
	}
	if scores[optionID2] != 0.2 {
		t.Errorf("Expected updated score 0.2, got %f", scores[optionID2])
	}
}

func TestSubmitBallotToClosedPoll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := getTestConfig()
	handler := NewVotingHandler(db, cfg)

	// Create a closed poll
	pollID, _ := auth.GenerateID(16)
	shareSlug := auth.GenerateShareSlug(pollID, cfg.PollSlugSalt)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, share_slug, created_at)
		VALUES ($1, 'Closed Poll', 'Alice', 'closed', $2, $3)
	`, pollID, shareSlug, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}

	// Add option
	optionID, _ := auth.GenerateID(12)
	_, err = db.Exec(`
		INSERT INTO option (id, poll_id, label)
		VALUES ($1, $2, 'Option A')
	`, optionID, pollID)
	if err != nil {
		t.Fatalf("Failed to create option: %v", err)
	}

	// Claim username
	voterToken, _ := auth.GenerateVoterToken()
	_, err = db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, 'voter1', $2, $3)
	`, pollID, voterToken, time.Now())
	if err != nil {
		t.Fatalf("Failed to create username claim: %v", err)
	}

	// Try to submit ballot
	body, _ := json.Marshal(models.SubmitBallotRequest{
		Scores: map[string]float64{optionID: 0.5},
	})

	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Voter-Token", voterToken)
	w := httptest.NewRecorder()

	handler.SubmitBallot(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}
