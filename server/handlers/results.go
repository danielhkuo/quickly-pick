// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielhkuo/quickly-pick/cliparse"
	"github.com/danielhkuo/quickly-pick/middleware"
	"github.com/danielhkuo/quickly-pick/models"
)

type ResultsHandler struct {
	db  *sql.DB
	cfg cliparse.Config
}

func NewResultsHandler(db *sql.DB, cfg cliparse.Config) *ResultsHandler {
	return &ResultsHandler{db: db, cfg: cfg}
}

// GetPoll handles GET /polls/:slug
// Returns poll details and options, but NOT results (results are sealed until closed)
func (h *ResultsHandler) GetPoll(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("slug")
	if shareSlug == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// Get poll by share slug
	var poll models.Poll
	err := h.db.QueryRow(`
		SELECT id, title, description, creator_name, method, status, 
		       share_slug, closes_at, closed_at, final_snapshot_id, created_at
		FROM poll
		WHERE share_slug = $1
	`, shareSlug).Scan(
		&poll.ID, &poll.Title, &poll.Description, &poll.CreatorName,
		&poll.Method, &poll.Status, &poll.ShareSlug, &poll.ClosesAt,
		&poll.ClosedAt, &poll.FinalSnapshotID, &poll.CreatedAt,
	)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get options
	rows, err := h.db.Query(`
		SELECT id, poll_id, label
		FROM option
		WHERE poll_id = $1
		ORDER BY id
	`, poll.ID)

	if err != nil {
		slog.Error("failed to query options", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	options := []models.Option{}
	for rows.Next() {
		var opt models.Option
		if err := rows.Scan(&opt.ID, &opt.PollID, &opt.Label); err != nil {
			slog.Error("failed to scan option", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
			return
		}
		options = append(options, opt)
	}

	response := models.PollWithOptions{
		Poll:    poll,
		Options: options,
	}

	middleware.JSONResponse(w, http.StatusOK, response)
}

// GetResults handles GET /polls/:slug/results
// Returns 403 if poll is open (results are sealed)
// Returns final snapshot if poll is closed
func (h *ResultsHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("slug")
	if shareSlug == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// Get poll status and snapshot ID
	var status string
	var snapshotID sql.NullString
	err := h.db.QueryRow(`
		SELECT status, final_snapshot_id
		FROM poll
		WHERE share_slug = $1
	`, shareSlug).Scan(&status, &snapshotID)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// CRITICAL: Results are sealed while poll is open
	if status != models.StatusClosed {
		middleware.ErrorResponse(w, http.StatusForbidden, "Results are hidden until poll is closed")
		return
	}

	// Poll is closed, return final snapshot
	if !snapshotID.Valid {
		slog.Error("closed poll has no snapshot", "slug", shareSlug)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Results not available")
		return
	}

	// Get snapshot
	var snapshot models.ResultSnapshot
	var payloadJSON []byte
	err = h.db.QueryRow(`
		SELECT id, poll_id, method, computed_at, payload
		FROM result_snapshot
		WHERE id = $1
	`, snapshotID.String).Scan(
		&snapshot.ID, &snapshot.PollID, &snapshot.Method,
		&snapshot.ComputedAt, &payloadJSON,
	)

	if err != nil {
		slog.Error("failed to query snapshot", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Parse JSON payload
	var payload struct {
		Rankings   []models.OptionStats `json:"rankings"`
		InputsHash string               `json:"inputs_hash"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		slog.Error("failed to parse snapshot payload", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to parse results")
		return
	}

	snapshot.Rankings = payload.Rankings
	snapshot.InputsHash = payload.InputsHash

	// Get poll information for the response
	var poll models.Poll
	err = h.db.QueryRow(`
		SELECT id, title, description, creator_name, method, status, 
		       share_slug, closes_at, closed_at, final_snapshot_id, created_at
		FROM poll
		WHERE share_slug = $1
	`, shareSlug).Scan(
		&poll.ID, &poll.Title, &poll.Description, &poll.CreatorName,
		&poll.Method, &poll.Status, &poll.ShareSlug, &poll.ClosesAt,
		&poll.ClosedAt, &poll.FinalSnapshotID, &poll.CreatedAt,
	)

	if err != nil {
		slog.Error("failed to query poll for results", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get ballot count
	var ballotCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM ballot WHERE poll_id = $1
	`, poll.ID).Scan(&ballotCount)

	if err != nil {
		slog.Error("failed to count ballots for results", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Return results in the format expected by frontend
	response := map[string]interface{}{
		"poll":         poll,
		"rankings":     snapshot.Rankings,
		"ballot_count": ballotCount,
	}

	middleware.JSONResponse(w, http.StatusOK, response)
}

// GetBallotCount handles GET /polls/:slug/ballot-count (optional convenience endpoint)
// Returns the number of ballots submitted (visible even while open)
func (h *ResultsHandler) GetBallotCount(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("slug")
	if shareSlug == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// Get poll ID
	var pollID string
	err := h.db.QueryRow(`
		SELECT id FROM poll WHERE share_slug = $1
	`, shareSlug).Scan(&pollID)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Count ballots
	var count int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM ballot WHERE poll_id = $1
	`, pollID).Scan(&count)

	if err != nil {
		slog.Error("failed to count ballots", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]int{
		"ballot_count": count,
	})
}

// GetPreview handles GET /polls/:slug/preview
// Returns compact poll data for iMessage bubble display
func (h *ResultsHandler) GetPreview(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("slug")
	if shareSlug == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// Get poll info with counts
	var title, status string
	var pollID string
	err := h.db.QueryRow(`
		SELECT id, title, status FROM poll WHERE share_slug = $1
	`, shareSlug).Scan(&pollID, &title, &status)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get option count
	var optionCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM option WHERE poll_id = $1
	`, pollID).Scan(&optionCount)
	if err != nil {
		slog.Error("failed to count options", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get ballot count
	var ballotCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM ballot WHERE poll_id = $1
	`, pollID).Scan(&ballotCount)
	if err != nil {
		slog.Error("failed to count ballots", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	middleware.JSONResponse(w, http.StatusOK, models.PollPreviewResponse{
		Title:       title,
		Status:      status,
		OptionCount: optionCount,
		BallotCount: ballotCount,
	})
}
