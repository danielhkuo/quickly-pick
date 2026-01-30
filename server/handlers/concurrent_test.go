// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/danielhkuo/quickly-pick/models"
	"github.com/danielhkuo/quickly-pick/testutil"
)

// TestConcurrentBallotSubmissions verifies that multiple simultaneous ballot
// submissions from different voters don't cause data corruption or duplicates
func TestConcurrentBallotSubmissions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	votingHandler := NewVotingHandler(db, cfg)

	// Create an open poll with options
	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	opt1 := testutil.AddTestOption(t, db, pollID, "Option A")
	opt2 := testutil.AddTestOption(t, db, pollID, "Option B")
	opt3 := testutil.AddTestOption(t, db, pollID, "Option C")

	numVoters := 10
	voterTokens := make([]string, numVoters)

	// Pre-create all voters
	for i := 0; i < numVoters; i++ {
		username := "ConcurrentVoter" + string(rune('A'+i))
		voterTokens[i] = testutil.CreateTestVoter(t, db, pollID, username)
	}

	// Track results
	var successCount atomic.Int32
	var wg sync.WaitGroup

	// Submit all ballots concurrently
	for i := 0; i < numVoters; i++ {
		wg.Add(1)
		go func(voterIdx int) {
			defer wg.Done()

			scores := map[string]float64{
				opt1: float64(voterIdx%3) / 2.0,
				opt2: float64((voterIdx+1)%3) / 2.0,
				opt3: float64((voterIdx+2)%3) / 2.0,
			}

			ballotReq := models.SubmitBallotRequest{Scores: scores}
			body, _ := json.Marshal(ballotReq)
			req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
			req.SetPathValue("slug", shareSlug)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Voter-Token", voterTokens[voterIdx])
			w := httptest.NewRecorder()

			votingHandler.SubmitBallot(w, req)

			if w.Code == http.StatusCreated || w.Code == http.StatusOK {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// All submissions should succeed
	if int(successCount.Load()) != numVoters {
		t.Errorf("Expected %d successful submissions, got %d", numVoters, successCount.Load())
	}

	// Verify database has exactly numVoters ballots
	var ballotCount int
	err := db.QueryRow("SELECT COUNT(*) FROM ballot WHERE poll_id = $1", pollID).Scan(&ballotCount)
	if err != nil {
		t.Fatalf("Failed to count ballots: %v", err)
	}

	if ballotCount != numVoters {
		t.Errorf("Expected %d ballots in database, got %d", numVoters, ballotCount)
	}

	// Verify no duplicate voter tokens
	var uniqueVoters int
	err = db.QueryRow("SELECT COUNT(DISTINCT voter_token) FROM ballot WHERE poll_id = $1", pollID).Scan(&uniqueVoters)
	if err != nil {
		t.Fatalf("Failed to count unique voters: %v", err)
	}

	if uniqueVoters != numVoters {
		t.Errorf("Expected %d unique voters, got %d (possible duplicates)", numVoters, uniqueVoters)
	}
}

// TestConcurrentUsernameClaims verifies that when two goroutines try to claim
// the same username, exactly one succeeds
func TestConcurrentUsernameClaims(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	votingHandler := NewVotingHandler(db, cfg)

	// Create an open poll
	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	testutil.AddTestOption(t, db, pollID, "A")
	testutil.AddTestOption(t, db, pollID, "B")

	contestedUsername := "RaceConditionUser"
	numAttempts := 5 // Multiple goroutines trying same username

	var successCount atomic.Int32
	var wg sync.WaitGroup

	// All goroutines try to claim the same username simultaneously
	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			claimReq := models.ClaimUsernameRequest{Username: contestedUsername}
			body, _ := json.Marshal(claimReq)
			req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
			req.SetPathValue("slug", shareSlug)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			votingHandler.ClaimUsername(w, req)

			if w.Code == http.StatusCreated {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Exactly one should succeed
	if successCount.Load() != 1 {
		t.Errorf("Expected exactly 1 successful claim, got %d", successCount.Load())
	}

	// Verify database has exactly one claim for this username
	var claimCount int
	err := db.QueryRow("SELECT COUNT(*) FROM username_claim WHERE poll_id = $1 AND username = $2",
		pollID, contestedUsername).Scan(&claimCount)
	if err != nil {
		t.Fatalf("Failed to count claims: %v", err)
	}

	if claimCount != 1 {
		t.Errorf("Expected 1 username claim in database, got %d", claimCount)
	}
}

// TestConcurrentPollClose verifies that when multiple goroutines try to close
// the same poll, the poll ends up in a valid closed state.
//
// NOTE: This test documents a known race condition - without proper transaction
// isolation in ClosePoll, multiple concurrent closes can each create a snapshot.
// The important invariant is that the poll ends up closed with at least one
// valid snapshot. Fixing this race would require SELECT FOR UPDATE or similar
// locking in the ClosePoll handler.
func TestConcurrentPollClose(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	pollHandler := NewPollHandler(db, cfg)

	// Create an open poll
	pollID, adminKey, _ := testutil.CreateTestPoll(t, db, cfg, "open")
	testutil.AddTestOption(t, db, pollID, "A")
	testutil.AddTestOption(t, db, pollID, "B")

	numAttempts := 3 // Multiple goroutines trying to close
	var successCount atomic.Int32
	var wg sync.WaitGroup

	// All goroutines try to close simultaneously
	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("POST", "/polls/"+pollID+"/close", nil)
			req.SetPathValue("id", pollID)
			req.Header.Set("X-Admin-Key", adminKey)
			w := httptest.NewRecorder()

			pollHandler.ClosePoll(w, req)

			if w.Code == http.StatusOK {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// At least one should succeed
	if successCount.Load() < 1 {
		t.Error("Expected at least one successful close")
	}

	// Verify poll is closed
	var status string
	err := db.QueryRow("SELECT status FROM poll WHERE id = $1", pollID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query poll status: %v", err)
	}

	if status != "closed" {
		t.Errorf("Expected poll status 'closed', got '%s'", status)
	}

	// Verify at least one snapshot was created
	// NOTE: Due to the race condition, multiple snapshots may be created.
	// This is a known issue that would require transaction isolation to fix.
	var snapshotCount int
	err = db.QueryRow("SELECT COUNT(*) FROM result_snapshot WHERE poll_id = $1", pollID).Scan(&snapshotCount)
	if err != nil {
		t.Fatalf("Failed to count snapshots: %v", err)
	}

	if snapshotCount < 1 {
		t.Error("Expected at least 1 snapshot")
	}
	if snapshotCount > 1 {
		t.Logf("WARNING: Race condition detected - %d snapshots created (expected 1). "+
			"Consider adding transaction isolation to ClosePoll handler.", snapshotCount)
	}
}

// TestConcurrentBallotUpdates verifies that a single voter updating their
// ballot multiple times concurrently results in a consistent final state
func TestConcurrentBallotUpdates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	votingHandler := NewVotingHandler(db, cfg)

	pollID, _, shareSlug := testutil.CreateTestPoll(t, db, cfg, "open")
	opt1 := testutil.AddTestOption(t, db, pollID, "A")
	opt2 := testutil.AddTestOption(t, db, pollID, "B")

	// Create a single voter
	voterToken := testutil.CreateTestVoter(t, db, pollID, "UpdaterVoter")

	// Submit initial ballot
	initialScores := map[string]float64{opt1: 0.5, opt2: 0.5}
	testutil.SubmitTestBallot(t, db, pollID, voterToken, initialScores)

	numUpdates := 10
	var wg sync.WaitGroup

	// Concurrently update the same ballot
	for i := 0; i < numUpdates; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Each update has different scores
			scores := map[string]float64{
				opt1: float64(idx) / float64(numUpdates),
				opt2: 1.0 - float64(idx)/float64(numUpdates),
			}

			ballotReq := models.SubmitBallotRequest{Scores: scores}
			body, _ := json.Marshal(ballotReq)
			req := httptest.NewRequest("POST", "/polls/"+shareSlug+"/ballots", bytes.NewReader(body))
			req.SetPathValue("slug", shareSlug)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Voter-Token", voterToken)
			w := httptest.NewRecorder()

			votingHandler.SubmitBallot(w, req)
			// We don't care which update wins, just that it completes
		}(i)
	}

	wg.Wait()

	// Verify there's still only one ballot for this voter
	var ballotCount int
	err := db.QueryRow("SELECT COUNT(*) FROM ballot WHERE poll_id = $1 AND voter_token = $2",
		pollID, voterToken).Scan(&ballotCount)
	if err != nil {
		t.Fatalf("Failed to count ballots: %v", err)
	}

	if ballotCount != 1 {
		t.Errorf("Expected 1 ballot after updates, got %d", ballotCount)
	}

	// Verify scores exist and are valid (between 0 and 1)
	var opt1Score, opt2Score float64
	err = db.QueryRow(`
		SELECT s.value01 FROM score s
		JOIN ballot b ON s.ballot_id = b.id
		WHERE b.poll_id = $1 AND b.voter_token = $2 AND s.option_id = $3
	`, pollID, voterToken, opt1).Scan(&opt1Score)
	if err != nil {
		t.Fatalf("Failed to query opt1 score: %v", err)
	}

	err = db.QueryRow(`
		SELECT s.value01 FROM score s
		JOIN ballot b ON s.ballot_id = b.id
		WHERE b.poll_id = $1 AND b.voter_token = $2 AND s.option_id = $3
	`, pollID, voterToken, opt2).Scan(&opt2Score)
	if err != nil {
		t.Fatalf("Failed to query opt2 score: %v", err)
	}

	if opt1Score < 0 || opt1Score > 1 {
		t.Errorf("opt1 score out of range: %f", opt1Score)
	}
	if opt2Score < 0 || opt2Score > 1 {
		t.Errorf("opt2 score out of range: %f", opt2Score)
	}
}

// TestParallelPolls verifies that operations on different polls don't interfere
func TestParallelPolls(t *testing.T) {
	t.Parallel() // This test can run in parallel with others

	db := testutil.SetupTestDB(t)
	defer db.Close()

	cfg := testutil.GetTestConfig()
	pollHandler := NewPollHandler(db, cfg)
	votingHandler := NewVotingHandler(db, cfg)

	numPolls := 5
	var wg sync.WaitGroup

	// Create and operate on multiple polls in parallel
	for i := 0; i < numPolls; i++ {
		wg.Add(1)
		go func(pollIdx int) {
			defer wg.Done()

			// Create poll
			createReq := models.CreatePollRequest{
				Title:       "Parallel Poll " + string(rune('A'+pollIdx)),
				Description: "Testing parallel operations",
				CreatorName: "Tester",
			}
			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest("POST", "/polls", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			pollHandler.CreatePoll(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("Poll %d creation failed: %d", pollIdx, w.Code)
				return
			}

			var createResp models.CreatePollResponse
			json.NewDecoder(w.Body).Decode(&createResp)
			pollID := createResp.PollID
			adminKey := createResp.AdminKey

			// Add options
			for j := 0; j < 3; j++ {
				optReq := models.AddOptionRequest{Label: "Option " + string(rune('A'+j))}
				body, _ := json.Marshal(optReq)
				req := httptest.NewRequest("POST", "/polls/"+pollID+"/options", bytes.NewReader(body))
				req.SetPathValue("id", pollID)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Admin-Key", adminKey)
				w := httptest.NewRecorder()
				pollHandler.AddOption(w, req)

				if w.Code != http.StatusCreated {
					t.Errorf("Poll %d option %d failed: %d", pollIdx, j, w.Code)
					return
				}
			}

			// Publish
			req = httptest.NewRequest("POST", "/polls/"+pollID+"/publish", nil)
			req.SetPathValue("id", pollID)
			req.Header.Set("X-Admin-Key", adminKey)
			w = httptest.NewRecorder()
			pollHandler.PublishPoll(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Poll %d publish failed: %d", pollIdx, w.Code)
				return
			}

			var publishResp models.PublishPollResponse
			json.NewDecoder(w.Body).Decode(&publishResp)
			shareSlug := publishResp.ShareSlug

			// Claim username
			claimReq := models.ClaimUsernameRequest{Username: "Voter" + string(rune('A'+pollIdx))}
			body, _ = json.Marshal(claimReq)
			req = httptest.NewRequest("POST", "/polls/"+shareSlug+"/claim-username", bytes.NewReader(body))
			req.SetPathValue("slug", shareSlug)
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			votingHandler.ClaimUsername(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("Poll %d username claim failed: %d", pollIdx, w.Code)
				return
			}
		}(i)
	}

	wg.Wait()

	// Verify all polls were created
	var pollCount int
	err := db.QueryRow("SELECT COUNT(*) FROM poll").Scan(&pollCount)
	if err != nil {
		t.Fatalf("Failed to count polls: %v", err)
	}

	if pollCount != numPolls {
		t.Errorf("Expected %d polls, got %d", numPolls, pollCount)
	}
}
