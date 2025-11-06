// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/danielhkuo/quickly-pick/models"
)

// BMJStats represents the statistical aggregates for a single option
type BMJStats struct {
	OptionID string
	Label    string
	Median   float64
	P10      float64
	P90      float64
	Mean     float64
	NegShare float64
	Veto     bool
}

// ComputeBMJRankings calculates Balanced Majority Judgment rankings for a poll
func ComputeBMJRankings(db *sql.DB, pollID string) ([]models.OptionStats, error) {
	// Get all options for the poll
	optionLabels, err := getOptionLabels(db, pollID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option labels: %w", err)
	}

	// Get all scores grouped by option
	optionScores, err := getOptionScores(db, pollID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option scores: %w", err)
	}

	// Compute statistics for each option
	var stats []BMJStats
	for optionID, rawScores := range optionScores {
		// Convert to signed scores: s = 2*value01 - 1
		signedScores := make([]float64, len(rawScores))
		for i, v := range rawScores {
			signedScores[i] = 2.0*v - 1.0
		}

		// Sort for percentile calculations
		sort.Float64s(signedScores)

		stat := BMJStats{
			OptionID: optionID,
			Label:    optionLabels[optionID],
			Median:   percentile(signedScores, 0.5),
			P10:      percentile(signedScores, 0.1),
			P90:      percentile(signedScores, 0.9),
			Mean:     mean(signedScores),
			NegShare: negativeShare(signedScores),
		}

		// Apply soft veto rule
		stat.Veto = stat.NegShare >= 0.33 && stat.Median <= 0

		stats = append(stats, stat)
	}

	// Handle options with no votes
	for optionID, label := range optionLabels {
		if _, hasScores := optionScores[optionID]; !hasScores {
			stats = append(stats, BMJStats{
				OptionID: optionID,
				Label:    label,
				Median:   0.0,
				P10:      0.0,
				P90:      0.0,
				Mean:     0.0,
				NegShare: 0.0,
				Veto:     false,
			})
		}
	}

	// Sort by BMJ ranking criteria (lexicographic order)
	sort.Slice(stats, func(i, j int) bool {
		a, b := stats[i], stats[j]

		// 1. Non-vetoed options come first
		if a.Veto != b.Veto {
			return !a.Veto // false (non-vetoed) < true (vetoed)
		}

		// 2. Higher median wins
		if a.Median != b.Median {
			return a.Median > b.Median
		}

		// 3. Higher p10 wins (least-misery tiebreaker)
		if a.P10 != b.P10 {
			return a.P10 > b.P10
		}

		// 4. Higher p90 wins (upside tiebreaker)
		if a.P90 != b.P90 {
			return a.P90 > b.P90
		}

		// 5. Higher mean wins
		if a.Mean != b.Mean {
			return a.Mean > b.Mean
		}

		// 6. Stable tie-breaking by option ID (ascending)
		return a.OptionID < b.OptionID
	})

	// Convert to models.OptionStats with ranks
	results := make([]models.OptionStats, len(stats))
	for i, stat := range stats {
		results[i] = models.OptionStats{
			OptionID: stat.OptionID,
			Label:    stat.Label,
			Median:   stat.Median,
			P10:      stat.P10,
			P90:      stat.P90,
			Mean:     stat.Mean,
			NegShare: stat.NegShare,
			Veto:     stat.Veto,
			Rank:     i + 1, // 1-indexed ranking
		}
	}

	return results, nil
}

// getOptionLabels retrieves option labels for a poll
func getOptionLabels(db *sql.DB, pollID string) (map[string]string, error) {
	rows, err := db.Query(`
		SELECT id, label FROM option WHERE poll_id = $1
	`, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := make(map[string]string)
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		labels[id] = label
	}

	return labels, rows.Err()
}

// getOptionScores retrieves all scores grouped by option
func getOptionScores(db *sql.DB, pollID string) (map[string][]float64, error) {
	rows, err := db.Query(`
		SELECT s.option_id, s.value01
		FROM score s
		JOIN ballot b ON s.ballot_id = b.id
		WHERE b.poll_id = $1
		ORDER BY s.option_id
	`, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := make(map[string][]float64)
	for rows.Next() {
		var optionID string
		var value float64
		if err := rows.Scan(&optionID, &value); err != nil {
			return nil, err
		}
		scores[optionID] = append(scores[optionID], value)
	}

	return scores, rows.Err()
}

// percentile calculates the p-th percentile of sorted data
// p should be in range [0, 1]
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	// Linear interpolation between closest ranks
	rank := p * float64(len(sorted)-1)
	lower := int(rank)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Interpolate
	weight := rank - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// mean calculates the arithmetic mean
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// negativeShare calculates the fraction of negative scores
func negativeShare(signedScores []float64) float64 {
	if len(signedScores) == 0 {
		return 0.0
	}

	negCount := 0
	for _, s := range signedScores {
		if s < 0 {
			negCount++
		}
	}
	return float64(negCount) / float64(len(signedScores))
}

// computeInputsHash creates a hash of all ballot IDs for verification
func computeInputsHash(db *sql.DB, pollID string) string {
	rows, err := db.Query(`
		SELECT id FROM ballot WHERE poll_id = $1 ORDER BY id
	`, pollID)
	if err != nil {
		return "error"
	}
	defer rows.Close()

	var ballotIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "error"
		}
		ballotIDs = append(ballotIDs, id)
	}

	// Simple concatenation hash (could be improved with proper hashing)
	if len(ballotIDs) == 0 {
		return "no-ballots"
	}

	// Return count as simple verification
	return fmt.Sprintf("%d-ballots", len(ballotIDs))
}
