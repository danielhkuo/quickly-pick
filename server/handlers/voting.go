// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielhkuo/quickly-pick/auth"
	"github.com/danielhkuo/quickly-pick/cliparse"
	"github.com/danielhkuo/quickly-pick/middleware"
	"github.com/danielhkuo/quickly-pick/models"
)

type VotingHandler struct {
	db  *sql.DB
	cfg cliparse.Config
}

func NewVotingHandler(db *sql.DB, cfg cliparse.Config) *VotingHandler {
	return &VotingHandler{db: db, cfg: cfg}
}

// ClaimUsername handles POST /polls/:slug/claim-username
func (h *VotingHandler) ClaimUsername(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("slug")
	if shareSlug == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// Parse request
	var req models.ClaimUsernameRequest
	if err := middleware.ParseJSONBody(r, &req); err != nil {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Username == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "username is required")
		return
	}

	// Validate username (basic validation)
	if len(req.Username) < 2 || len(req.Username) > 50 {
		middleware.ErrorResponse(w, http.StatusBadRequest, "username must be 2-50 characters")
		return
	}

	// Find poll by share slug
	var pollID string
	var status string
	err := h.db.QueryRow(`
		SELECT id, status FROM poll WHERE share_slug = $1
	`, shareSlug).Scan(&pollID, &status)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Can only claim username for open polls
	if status != models.StatusOpen {
		middleware.ErrorResponse(w, http.StatusConflict, "Poll is not open for voting")
		return
	}

	// Generate voter token
	voterToken, err := auth.GenerateVoterToken()
	if err != nil {
		slog.Error("failed to generate voter token", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to claim username")
		return
	}

	// Insert username claim (UNIQUE constraint will prevent duplicates)
	_, err = h.db.Exec(`
		INSERT INTO username_claim (poll_id, username, voter_token, created_at)
		VALUES ($1, $2, $3, $4)
	`, pollID, req.Username, voterToken, time.Now())

	if err != nil {
		// Check if it's a uniqueness violation
		if err.Error() == "UNIQUE constraint failed: username_claim.poll_id, username_claim.username" ||
			err.Error() == "pq: duplicate key value violates unique constraint \"username_claim_poll_id_username_key\"" {
			middleware.ErrorResponse(w, http.StatusConflict, "Username already taken")
			return
		}
		slog.Error("failed to insert username claim", "error", err, "poll_id", pollID)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to claim username")
		return
	}

	// Link device to poll as voter (if X-Device-UUID header present)
	deviceID, err := GetOrCreateDevice(h.db, r)
	if err != nil {
		slog.Warn("failed to get/create device", "error", err)
		// Non-fatal: username was claimed, just no device linking
	} else if deviceID != "" {
		if err := LinkDeviceToPoll(h.db, deviceID, pollID, models.RoleVoter, &voterToken); err != nil {
			slog.Warn("failed to link device to poll", "error", err)
		}
	}

	slog.Info("username claimed", "poll_id", pollID, "username", req.Username)

	middleware.JSONResponse(w, http.StatusCreated, models.ClaimUsernameResponse{
		VoterToken: voterToken,
	})
}

// SubmitBallot handles POST /polls/:slug/ballots
func (h *VotingHandler) SubmitBallot(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("slug")
	if shareSlug == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// Get voter token from header
	voterToken := r.Header.Get("X-Voter-Token")
	if voterToken == "" {
		middleware.ErrorResponse(w, http.StatusUnauthorized, "X-Voter-Token header required")
		return
	}

	// Parse request
	var req models.SubmitBallotRequest
	if err := middleware.ParseJSONBody(r, &req); err != nil {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(req.Scores) == 0 {
		middleware.ErrorResponse(w, http.StatusBadRequest, "scores cannot be empty")
		return
	}

	// Validate all scores are in range [0, 1]
	for optionID, score := range req.Scores {
		if score < 0 || score > 1 {
			middleware.ErrorResponse(w, http.StatusBadRequest, "score for "+optionID+" must be between 0 and 1")
			return
		}
	}

	// Find poll by share slug
	var pollID string
	var status string
	err := h.db.QueryRow(`
		SELECT id, status FROM poll WHERE share_slug = $1
	`, shareSlug).Scan(&pollID, &status)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Can only vote on open polls
	if status != models.StatusOpen {
		middleware.ErrorResponse(w, http.StatusConflict, "Poll is not open for voting")
		return
	}

	// Verify voter token is valid for this poll
	var exists bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM username_claim
			WHERE poll_id = $1 AND voter_token = $2
		)
	`, pollID, voterToken).Scan(&exists)

	if err != nil {
		slog.Error("failed to verify voter token", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	if !exists {
		middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid voter token for this poll")
		return
	}

	// Get all valid option IDs for this poll
	rows, err := h.db.Query(`
		SELECT id FROM option WHERE poll_id = $1
	`, pollID)
	if err != nil {
		slog.Error("failed to query options", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	validOptions := make(map[string]bool)
	for rows.Next() {
		var optionID string
		if err := rows.Scan(&optionID); err != nil {
			slog.Error("failed to scan option", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
			return
		}
		validOptions[optionID] = true
	}

	// Verify all submitted scores are for valid options
	for optionID := range req.Scores {
		if !validOptions[optionID] {
			middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid option_id: "+optionID)
			return
		}
	}

	// Get IP hash for tracking
	clientIP := middleware.GetClientIP(r)
	ipHash := auth.HashIP(clientIP, h.cfg.AdminKeySalt) // Reuse admin salt for IP hashing
	userAgent := r.UserAgent()

	// Begin transaction for UPSERT
	tx, err := h.db.Begin()
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer tx.Rollback()

	// Check if ballot already exists
	var existingBallotID string
	err = tx.QueryRow(`
		SELECT id FROM ballot WHERE poll_id = $1 AND voter_token = $2
	`, pollID, voterToken).Scan(&existingBallotID)

	isUpdate := err != sql.ErrNoRows
	var ballotID string

	if isUpdate {
		// Update existing ballot
		ballotID = existingBallotID
		_, err = tx.Exec(`
			UPDATE ballot
			SET submitted_at = $1, ip_hash = $2, user_agent = $3
			WHERE id = $4
		`, time.Now(), ipHash, userAgent, ballotID)

		if err != nil {
			slog.Error("failed to update ballot", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to update ballot")
			return
		}

		// Delete old scores
		_, err = tx.Exec(`DELETE FROM score WHERE ballot_id = $1`, ballotID)
		if err != nil {
			slog.Error("failed to delete old scores", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to update ballot")
			return
		}
	} else {
		// Create new ballot
		ballotID, _ = auth.GenerateID(16)
		_, err = tx.Exec(`
			INSERT INTO ballot (id, poll_id, voter_token, submitted_at, ip_hash, user_agent)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, ballotID, pollID, voterToken, time.Now(), ipHash, userAgent)

		if err != nil {
			slog.Error("failed to insert ballot", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to submit ballot")
			return
		}
	}

	// Insert scores
	for optionID, score := range req.Scores {
		_, err = tx.Exec(`
			INSERT INTO score (ballot_id, option_id, value01)
			VALUES ($1, $2, $3)
		`, ballotID, optionID, score)

		if err != nil {
			slog.Error("failed to insert score", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to save scores")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to submit ballot")
		return
	}

	message := "Ballot submitted successfully"
	if isUpdate {
		message = "Ballot updated successfully"
	}

	slog.Info("ballot submitted", "poll_id", pollID, "ballot_id", ballotID, "is_update", isUpdate)

	middleware.JSONResponse(w, http.StatusCreated, models.SubmitBallotResponse{
		BallotID: ballotID,
		Message:  message,
	})
}
