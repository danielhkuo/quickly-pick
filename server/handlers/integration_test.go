// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielhkuo/quickly-pick/models"
	"github.com/danielhkuo/quickly-pick/testutil"
)

// TestFullVotingWorkflow tests the complete end-to-end workflow:
// 1. Create poll
// 2. Add options
// 3. Publish poll
// 4. Voters claim usernames
// 5. Voters submit ballots
// 6. Update a ballot
// 7. Close poll
// 8. Verify results
func TestFullVotingWorkflow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	pollHandler := NewPollHandler(db, cfg)
	votingHandler := NewVotingHandler(db, cfg)
	resultsHandler := NewResultsHandler(db, cfg)

	// Step 1: Create a poll
	createReq := models.CreatePollRequest{
		Title:       "Integration Test Poll",
		Description: "Testing the full voting workflow",
		CreatorName: "IntegrationTester",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	pollHandler.CreatePoll(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Step 1 - Create poll failed: %d - %s", w.Code, w.Body.String())
	}

	var createResp models.CreatePollResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	pollID := createResp.PollID
	adminKey := createResp.AdminKey

	if pollID == "" || adminKey == "" {
		t.Fatal("Step 1 - Missing poll_id or admin_key")
	}
	t.Logf("Step 1 - Created poll: %s", pollID)

	// Step 2: Add 3 options
	options := []string{"Pizza", "Sushi", "Tacos"}
	optionIDs := make([]string, 0, len(options))

	for _, label := range options {
		optionReq := models.AddOptionRequest{Label: label}
		body, _ := json.Marshal(optionReq)
		req := httptest.NewRequest("POST", "/polls/"+pollID+"/options", bytes.NewReader(body))
		req.SetPathValue("id", pollID)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Key", adminKey)
		w := httptest.NewRecorder()
		pollHandler.AddOption(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Step 2 - Add option '%s' failed: %d - %s", label, w.Code, w.Body.String())
		}

		var optionResp models.AddOptionResponse
		json.NewDecoder(w.Body).Decode(&optionResp)
		optionIDs = append(optionIDs, optionResp.OptionID)
	}
	t.Logf("Step 2 - Added %d options", len(optionIDs))

	// Step 3: Publish poll
	req = httptest.NewRequest("POST", "/polls/"+pollID+"/publish", nil)
	req.SetPathValue("id", pollID)
	req.Header.Set("X-Admin-Key", adminKey)
	w = httptest.NewRecorder()
	pollHandler.PublishPoll(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 3 - Publish failed: %d - %s", w.Code, w.Body.String())
	}

	var publishResp models.PublishPollResponse
	json.NewDecoder(w.Body).Decode(&publishResp)
	shareSlug := publishResp.ShareSlug

	if shareSlug == "" {
		t.Fatal("Step 3 - Missing share_slug")
	}
	t.Logf("Step 3 - Published poll with slug: %s", shareSlug)

	// Step 4: 3 voters claim usernames
	voters := []string{"Alice", "Bob", "Charlie"}
	voterTokens := make([]string, 0, len(voters))

	for _, username := range voters {
		claimReq := models.ClaimUsernameRequest{Username: username}
		body, _ := json.Marshal(claimReq)
		req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
		req.SetPathValue("slug", shareSlug)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		votingHandler.ClaimUsername(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Step 4 - Claim username '%s' failed: %d - %s", username, w.Code, w.Body.String())
		}

		var claimResp models.ClaimUsernameResponse
		json.NewDecoder(w.Body).Decode(&claimResp)
		voterTokens = append(voterTokens, claimResp.VoterToken)
	}
	t.Logf("Step 4 - %d voters claimed usernames", len(voterTokens))

	// Step 5: 3 voters submit ballots
	// Alice: loves Pizza (1.0), hates Sushi (0.0), meh on Tacos (0.5)
	// Bob: hates Pizza (0.0), loves Sushi (1.0), likes Tacos (0.7)
	// Charlie: likes Pizza (0.8), meh Sushi (0.5), loves Tacos (1.0)
	ballotScores := []map[string]float64{
		{optionIDs[0]: 1.0, optionIDs[1]: 0.0, optionIDs[2]: 0.5}, // Alice
		{optionIDs[0]: 0.0, optionIDs[1]: 1.0, optionIDs[2]: 0.7}, // Bob
		{optionIDs[0]: 0.8, optionIDs[1]: 0.5, optionIDs[2]: 1.0}, // Charlie
	}

	ballotIDs := make([]string, 0, len(voters))
	for i, scores := range ballotScores {
		ballotReq := models.SubmitBallotRequest{Scores: scores}
		body, _ := json.Marshal(ballotReq)
		req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
		req.SetPathValue("slug", shareSlug)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Voter-Token", voterTokens[i])
		w := httptest.NewRecorder()
		votingHandler.SubmitBallot(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Fatalf("Step 5 - Submit ballot for voter %d failed: %d - %s", i, w.Code, w.Body.String())
		}

		var ballotResp models.SubmitBallotResponse
		json.NewDecoder(w.Body).Decode(&ballotResp)
		ballotIDs = append(ballotIDs, ballotResp.BallotID)
	}
	t.Logf("Step 5 - %d ballots submitted", len(ballotIDs))

	// Step 6: Alice updates her ballot (changes mind on Sushi)
	updateScores := map[string]float64{
		optionIDs[0]: 1.0, // Still loves Pizza
		optionIDs[1]: 0.3, // Warmed up to Sushi a bit
		optionIDs[2]: 0.5, // Still meh on Tacos
	}
	ballotReq := models.SubmitBallotRequest{Scores: updateScores}
	body, _ = json.Marshal(ballotReq)
	req = httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Voter-Token", voterTokens[0]) // Alice's token
	w = httptest.NewRecorder()
	votingHandler.SubmitBallot(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("Step 6 - Update ballot failed: %d - %s", w.Code, w.Body.String())
	}

	var updateResp models.SubmitBallotResponse
	json.NewDecoder(w.Body).Decode(&updateResp)
	t.Logf("Step 6 - Ballot updated: %s", updateResp.Message)

	// Verify ballot count
	req = httptest.NewRequest("GET", "/polls/"+shareSlug+"/ballot-count", nil)
	req.SetPathValue("slug", shareSlug)
	w = httptest.NewRecorder()
	resultsHandler.GetBallotCount(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ballot count check failed: %d - %s", w.Code, w.Body.String())
	}

	var countResp struct {
		Count int `json:"ballot_count"`
	}
	json.NewDecoder(w.Body).Decode(&countResp)
	if countResp.Count != 3 {
		t.Errorf("Expected 3 ballots, got %d", countResp.Count)
	}

	// Step 7: Close the poll
	req = httptest.NewRequest("POST", "/polls/"+pollID+"/close", nil)
	req.SetPathValue("id", pollID)
	req.Header.Set("X-Admin-Key", adminKey)
	w = httptest.NewRecorder()
	pollHandler.ClosePoll(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 7 - Close poll failed: %d - %s", w.Code, w.Body.String())
	}

	var closeResp models.ClosePollResponse
	json.NewDecoder(w.Body).Decode(&closeResp)

	if closeResp.ClosedAt.IsZero() {
		t.Error("Step 7 - Expected non-zero closed_at")
	}
	if closeResp.Snapshot.ID == "" {
		t.Error("Step 7 - Expected snapshot ID")
	}
	t.Logf("Step 7 - Poll closed at %v", closeResp.ClosedAt)

	// Step 8: Verify results
	req = httptest.NewRequest("GET", "/polls/"+shareSlug+"/results", nil)
	req.SetPathValue("slug", shareSlug)
	w = httptest.NewRecorder()
	resultsHandler.GetResults(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 8 - Get results failed: %d - %s", w.Code, w.Body.String())
	}

	var resultsResp models.ResultSnapshot
	json.NewDecoder(w.Body).Decode(&resultsResp)

	if len(resultsResp.Rankings) != 3 {
		t.Errorf("Step 8 - Expected 3 ranked options, got %d", len(resultsResp.Rankings))
	}

	// Verify rankings have expected fields
	for i, ranking := range resultsResp.Rankings {
		if ranking.OptionID == "" {
			t.Errorf("Step 8 - Ranking %d missing option_id", i)
		}
		if ranking.Label == "" {
			t.Errorf("Step 8 - Ranking %d missing label", i)
		}
		if ranking.Rank == 0 {
			t.Errorf("Step 8 - Ranking %d has zero rank", i)
		}
		t.Logf("Step 8 - Rank %d: %s (median=%.2f)", ranking.Rank, ranking.Label, ranking.Median)
	}

	t.Log("Integration test completed successfully!")
}

// TestPreviewDuringVoting tests that preview returns data during voting
func TestPreviewDuringVoting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	resultsHandler := NewResultsHandler(db, cfg)

	// Create an open poll with options and ballots
	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	opt1 := testutil.AddTestOption(t, db, pollID, "Option A")
	opt2 := testutil.AddTestOption(t, db, pollID, "Option B")

	// Add a voter and ballot
	voterToken := testutil.CreateTestVoter(t, db, pollID, "Voter1")
	testutil.SubmitTestBallot(t, db, pollID, voterToken, map[string]float64{
		opt1: 0.8,
		opt2: 0.3,
	})

	// Get preview
	req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/preview", nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()
	resultsHandler.GetPreview(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Preview request failed: %d - %s", w.Code, w.Body.String())
	}

	var preview models.PollPreviewResponse
	json.NewDecoder(w.Body).Decode(&preview)

	if preview.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", preview.Status)
	}
	if preview.OptionCount != 2 {
		t.Errorf("Expected 2 options, got %d", preview.OptionCount)
	}
	if preview.BallotCount != 1 {
		t.Errorf("Expected 1 ballot, got %d", preview.BallotCount)
	}
}

// TestBallotCountAccuracy verifies ballot count is accurate during voting
func TestBallotCountAccuracy(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	resultsHandler := NewResultsHandler(db, cfg)

	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	opt1 := testutil.AddTestOption(t, db, pollID, "A")
	opt2 := testutil.AddTestOption(t, db, pollID, "B")

	// Initially 0 ballots
	req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/ballot-count", nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()
	resultsHandler.GetBallotCount(w, req)

	var countResp struct {
		Count int `json:"ballot_count"`
	}
	json.NewDecoder(w.Body).Decode(&countResp)
	if countResp.Count != 0 {
		t.Errorf("Expected 0 ballots initially, got %d", countResp.Count)
	}

	// Add voters and ballots incrementally
	for i := 1; i <= 5; i++ {
		voterToken := testutil.CreateTestVoter(t, db, pollID, "Voter"+string(rune('0'+i)))
		testutil.SubmitTestBallot(t, db, pollID, voterToken, map[string]float64{
			opt1: float64(i) / 5.0,
			opt2: 1.0 - float64(i)/5.0,
		})

		req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/ballot-count", nil)
		req.SetPathValue("slug", shareSlug)
		w := httptest.NewRecorder()
		resultsHandler.GetBallotCount(w, req)

		json.NewDecoder(w.Body).Decode(&countResp)
		if countResp.Count != i {
			t.Errorf("After %d ballots, count was %d", i, countResp.Count)
		}
	}
}

// TestResultsSealedUntilClosed verifies results aren't available until poll is closed
func TestResultsSealedUntilClosed(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	resultsHandler := NewResultsHandler(db, cfg)

	// Create an open poll
	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	opt1 := testutil.AddTestOption(t, db, pollID, "A")
	testutil.AddTestOption(t, db, pollID, "B")

	// Add a ballot
	voterToken := testutil.CreateTestVoter(t, db, pollID, "Voter")
	testutil.SubmitTestBallot(t, db, pollID, voterToken, map[string]float64{opt1: 0.5})

	// Try to get results - should fail
	req := httptest.NewRequest("GET", "/polls/"+shareSlug+"/results", nil)
	req.SetPathValue("slug", shareSlug)
	w := httptest.NewRecorder()
	resultsHandler.GetResults(w, req)

	// Should return an error status (likely 404 or 409)
	if w.Code == http.StatusOK {
		t.Error("Expected results to be unavailable for open poll")
	}
}

// TestDuplicateUsernameClaim verifies same username can't be claimed twice
func TestDuplicateUsernameClaim(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	votingHandler := NewVotingHandler(db, cfg)

	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	testutil.AddTestOption(t, db, pollID, "A")
	testutil.AddTestOption(t, db, pollID, "B")

	// First claim should succeed
	claimReq := models.ClaimUsernameRequest{Username: "UniqueUser"}
	body, _ := json.Marshal(claimReq)
	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	votingHandler.ClaimUsername(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("First claim should succeed: %d - %s", w.Code, w.Body.String())
	}

	// Second claim with same username should fail
	body, _ = json.Marshal(claimReq)
	req = httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	votingHandler.ClaimUsername(w, req)

	if w.Code == http.StatusCreated {
		t.Error("Second claim with same username should fail")
	}
}

// TestCannotVoteOnClosedPoll verifies voting is blocked after poll closes
func TestCannotVoteOnClosedPoll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	votingHandler := NewVotingHandler(db, cfg)

	// Create a closed poll
	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "closed")
	opt1 := testutil.AddTestOption(t, db, pollID, "A")
	opt2 := testutil.AddTestOption(t, db, pollID, "B")

	// Create a voter (would have been created before close in real scenario)
	voterToken := testutil.CreateTestVoter(t, db, pollID, "LateVoter")

	// Try to submit ballot - should fail
	ballotReq := models.SubmitBallotRequest{Scores: map[string]float64{opt1: 0.5, opt2: 0.5}}
	body, _ := json.Marshal(ballotReq)
	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Voter-Token", voterToken)
	w := httptest.NewRecorder()
	votingHandler.SubmitBallot(w, req)

	if w.Code == http.StatusCreated || w.Code == http.StatusOK {
		t.Error("Should not be able to vote on closed poll")
	}
}

// TestCannotClaimUsernameOnClosedPoll verifies username claims blocked after close
func TestCannotClaimUsernameOnClosedPoll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	votingHandler := NewVotingHandler(db, cfg)

	// Create a closed poll
	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "closed")
	testutil.AddTestOption(t, db, pollID, "A")

	// Try to claim username - should fail
	claimReq := models.ClaimUsernameRequest{Username: "TooLate"}
	body, _ := json.Marshal(claimReq)
	req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
	req.SetPathValue("slug", shareSlug)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	votingHandler.ClaimUsername(w, req)

	if w.Code == http.StatusCreated {
		t.Error("Should not be able to claim username on closed poll")
	}
}
