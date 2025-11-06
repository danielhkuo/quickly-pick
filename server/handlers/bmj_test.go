// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"testing"
	"time"

	"github.com/danielhkuo/quickly-pick/auth"
	"github.com/danielhkuo/quickly-pick/models"
	_ "github.com/lib/pq"
)

func TestComputeBMJRankings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a test poll with options
	pollID, _ := auth.GenerateID(16)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'BMJ Test Poll', 'Alice', 'open', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Create three options
	optionA, _ := auth.GenerateID(12)
	optionB, _ := auth.GenerateID(12)
	optionC, _ := auth.GenerateID(12)

	for _, opt := range []struct {
		id    string
		label string
	}{
		{optionA, "Option A"},
		{optionB, "Option B"},
		{optionC, "Option C"},
	} {
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, opt.id, pollID, opt.label)
		if err != nil {
			t.Fatalf("Failed to create option: %v", err)
		}
	}

	// Create ballots with scores
	// Option A: [0.9, 0.8, 0.7, 0.2, 0.6] -> mostly positive
	// Option B: [0.1, 0.2, 0.8, 0.1, 0.3] -> mixed, many negative
	// Option C: [0.6, 0.4, 0.5, 0.7, 0.3] -> middle ground
	ballots := []struct {
		voterToken string
		scores     map[string]float64
	}{
		{"voter1", map[string]float64{optionA: 0.9, optionB: 0.1, optionC: 0.6}},
		{"voter2", map[string]float64{optionA: 0.8, optionB: 0.2, optionC: 0.4}},
		{"voter3", map[string]float64{optionA: 0.7, optionB: 0.8, optionC: 0.5}},
		{"voter4", map[string]float64{optionA: 0.2, optionB: 0.1, optionC: 0.7}},
		{"voter5", map[string]float64{optionA: 0.6, optionB: 0.3, optionC: 0.3}},
	}

	for _, ballot := range ballots {
		ballotID, _ := auth.GenerateID(16)
		_, err := db.Exec(`
			INSERT INTO ballot (id, poll_id, voter_token, submitted_at)
			VALUES ($1, $2, $3, $4)
		`, ballotID, pollID, ballot.voterToken, time.Now())
		if err != nil {
			t.Fatalf("Failed to create ballot: %v", err)
		}

		for optionID, score := range ballot.scores {
			_, err := db.Exec(`
				INSERT INTO score (ballot_id, option_id, value01)
				VALUES ($1, $2, $3)
			`, ballotID, optionID, score)
			if err != nil {
				t.Fatalf("Failed to create score: %v", err)
			}
		}
	}

	// Compute rankings
	rankings, err := ComputeBMJRankings(db, pollID)
	if err != nil {
		t.Fatalf("ComputeBMJRankings failed: %v", err)
	}

	// Verify we got 3 rankings
	if len(rankings) != 3 {
		t.Fatalf("Expected 3 rankings, got %d", len(rankings))
	}

	// Verify rankings are properly ranked
	for i, ranking := range rankings {
		if ranking.Rank != i+1 {
			t.Errorf("Expected rank %d, got %d for option %s", i+1, ranking.Rank, ranking.Label)
		}
	}

	// Option A should win (high median, no veto)
	if rankings[0].OptionID != optionA {
		t.Errorf("Expected Option A to be rank 1, got %s", rankings[0].Label)
	}

	// Option B should have high neg_share (4 out of 5 voters gave < 0.5)
	optionBRanking := findRanking(rankings, optionB)
	if optionBRanking == nil {
		t.Fatal("Option B not found in rankings")
	}
	if optionBRanking.NegShare < 0.6 {
		t.Errorf("Expected Option B to have high neg_share, got %f", optionBRanking.NegShare)
	}

	// Verify all statistics are in expected ranges
	for _, ranking := range rankings {
		if ranking.Median < -1.0 || ranking.Median > 1.0 {
			t.Errorf("Median out of range for %s: %f", ranking.Label, ranking.Median)
		}
		if ranking.P10 < -1.0 || ranking.P10 > 1.0 {
			t.Errorf("P10 out of range for %s: %f", ranking.Label, ranking.P10)
		}
		if ranking.P90 < -1.0 || ranking.P90 > 1.0 {
			t.Errorf("P90 out of range for %s: %f", ranking.Label, ranking.P90)
		}
		if ranking.Mean < -1.0 || ranking.Mean > 1.0 {
			t.Errorf("Mean out of range for %s: %f", ranking.Label, ranking.Mean)
		}
		if ranking.NegShare < 0.0 || ranking.NegShare > 1.0 {
			t.Errorf("NegShare out of range for %s: %f", ranking.Label, ranking.NegShare)
		}
	}
}

func TestSoftVeto(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create poll
	pollID, _ := auth.GenerateID(16)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'Veto Test Poll', 'Bob', 'open', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Create two options
	optionGood, _ := auth.GenerateID(12)
	optionBad, _ := auth.GenerateID(12)

	for _, opt := range []struct {
		id    string
		label string
	}{
		{optionGood, "Good Option"},
		{optionBad, "Bad Option"},
	} {
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, opt.id, pollID, opt.label)
		if err != nil {
			t.Fatalf("Failed to create option: %v", err)
		}
	}

	// Create ballots where Bad Option gets vetoed
	// Bad Option: [0.1, 0.1, 0.2, 0.3] -> all negative, median negative, >33% negative
	// Good Option: [0.8, 0.7, 0.9, 0.6] -> all positive
	ballots := []struct {
		voterToken string
		scores     map[string]float64
	}{
		{"voter1", map[string]float64{optionGood: 0.8, optionBad: 0.1}},
		{"voter2", map[string]float64{optionGood: 0.7, optionBad: 0.1}},
		{"voter3", map[string]float64{optionGood: 0.9, optionBad: 0.2}},
		{"voter4", map[string]float64{optionGood: 0.6, optionBad: 0.3}},
	}

	for _, ballot := range ballots {
		ballotID, _ := auth.GenerateID(16)
		_, err := db.Exec(`
			INSERT INTO ballot (id, poll_id, voter_token, submitted_at)
			VALUES ($1, $2, $3, $4)
		`, ballotID, pollID, ballot.voterToken, time.Now())
		if err != nil {
			t.Fatalf("Failed to create ballot: %v", err)
		}

		for optionID, score := range ballot.scores {
			_, err := db.Exec(`
				INSERT INTO score (ballot_id, option_id, value01)
				VALUES ($1, $2, $3)
			`, ballotID, optionID, score)
			if err != nil {
				t.Fatalf("Failed to create score: %v", err)
			}
		}
	}

	// Compute rankings
	rankings, err := ComputeBMJRankings(db, pollID)
	if err != nil {
		t.Fatalf("ComputeBMJRankings failed: %v", err)
	}

	// Good Option should be rank 1
	if rankings[0].OptionID != optionGood {
		t.Errorf("Expected Good Option to be rank 1")
	}
	if rankings[0].Veto {
		t.Error("Good Option should not be vetoed")
	}

	// Bad Option should be rank 2 and vetoed
	if rankings[1].OptionID != optionBad {
		t.Errorf("Expected Bad Option to be rank 2")
	}
	if !rankings[1].Veto {
		t.Error("Bad Option should be vetoed")
	}
	if rankings[1].NegShare < 0.33 {
		t.Errorf("Bad Option should have neg_share >= 0.33, got %f", rankings[1].NegShare)
	}
	if rankings[1].Median > 0 {
		t.Errorf("Bad Option should have median <= 0, got %f", rankings[1].Median)
	}
}

func TestNoVotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create poll with options but no ballots
	pollID, _ := auth.GenerateID(16)
	_, err := db.Exec(`
		INSERT INTO poll (id, title, creator_name, status, created_at)
		VALUES ($1, 'No Votes Poll', 'Charlie', 'open', $2)
	`, pollID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Create options
	optionA, _ := auth.GenerateID(12)
	optionB, _ := auth.GenerateID(12)

	for _, opt := range []struct {
		id    string
		label string
	}{
		{optionA, "Option A"},
		{optionB, "Option B"},
	} {
		_, err := db.Exec(`
			INSERT INTO option (id, poll_id, label)
			VALUES ($1, $2, $3)
		`, opt.id, pollID, opt.label)
		if err != nil {
			t.Fatalf("Failed to create option: %v", err)
		}
	}

	// Compute rankings (no ballots)
	rankings, err := ComputeBMJRankings(db, pollID)
	if err != nil {
		t.Fatalf("ComputeBMJRankings failed: %v", err)
	}

	// Should get 2 rankings with default values
	if len(rankings) != 2 {
		t.Fatalf("Expected 2 rankings, got %d", len(rankings))
	}

	// All statistics should be zero
	for _, ranking := range rankings {
		if ranking.Median != 0.0 {
			t.Errorf("Expected median 0.0 for no votes, got %f", ranking.Median)
		}
		if ranking.P10 != 0.0 {
			t.Errorf("Expected p10 0.0 for no votes, got %f", ranking.P10)
		}
		if ranking.P90 != 0.0 {
			t.Errorf("Expected p90 0.0 for no votes, got %f", ranking.P90)
		}
		if ranking.Mean != 0.0 {
			t.Errorf("Expected mean 0.0 for no votes, got %f", ranking.Mean)
		}
		if ranking.NegShare != 0.0 {
			t.Errorf("Expected neg_share 0.0 for no votes, got %f", ranking.NegShare)
		}
		if ranking.Veto {
			t.Error("Expected no veto for no votes")
		}
	}
}

func TestPercentileCalculation(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		p        float64
		expected float64
	}{
		{"empty", []float64{}, 0.5, 0.0},
		{"single value", []float64{5.0}, 0.5, 5.0},
		{"median of odd count", []float64{1.0, 2.0, 3.0}, 0.5, 2.0},
		{"median of even count", []float64{1.0, 2.0, 3.0, 4.0}, 0.5, 2.5},
		{"10th percentile", []float64{1.0, 2.0, 3.0, 4.0, 5.0}, 0.1, 1.4},
		{"90th percentile", []float64{1.0, 2.0, 3.0, 4.0, 5.0}, 0.9, 4.6},
		{"min (p=0)", []float64{1.0, 2.0, 3.0}, 0.0, 1.0},
		{"max (p=1)", []float64{1.0, 2.0, 3.0}, 1.0, 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sort data for percentile calculation
			sorted := make([]float64, len(tt.data))
			copy(sorted, tt.data)

			result := percentile(sorted, tt.p)
			if result != tt.expected {
				t.Errorf("percentile(%v, %f) = %f, want %f", tt.data, tt.p, result, tt.expected)
			}
		})
	}
}

func TestMean(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		expected float64
	}{
		{"empty", []float64{}, 0.0},
		{"single value", []float64{5.0}, 5.0},
		{"positive values", []float64{1.0, 2.0, 3.0}, 2.0},
		{"negative values", []float64{-1.0, -2.0, -3.0}, -2.0},
		{"mixed values", []float64{-1.0, 0.0, 1.0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mean(tt.data)
			if result != tt.expected {
				t.Errorf("mean(%v) = %f, want %f", tt.data, result, tt.expected)
			}
		})
	}
}

func TestNegativeShare(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		expected float64
	}{
		{"empty", []float64{}, 0.0},
		{"all positive", []float64{0.1, 0.5, 1.0}, 0.0},
		{"all negative", []float64{-0.1, -0.5, -1.0}, 1.0},
		{"half negative", []float64{-1.0, -0.5, 0.5, 1.0}, 0.5},
		{"one third negative", []float64{-1.0, 0.5, 1.0}, 1.0 / 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := negativeShare(tt.data)
			if result != tt.expected {
				t.Errorf("negativeShare(%v) = %f, want %f", tt.data, result, tt.expected)
			}
		})
	}
}

// Helper function to find a ranking by option ID
func findRanking(rankings []models.OptionStats, optionID string) *models.OptionStats {
	for i := range rankings {
		if rankings[i].OptionID == optionID {
			return &rankings[i]
		}
	}
	return nil
}
