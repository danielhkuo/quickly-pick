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

type PollHandler struct {
	db  *sql.DB
	cfg cliparse.Config
}

func NewPollHandler(db *sql.DB, cfg cliparse.Config) *PollHandler {
	return &PollHandler{db: db, cfg: cfg}
}

// CreatePoll handles POST /polls
func (h *PollHandler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePollRequest
	if err := middleware.ParseJSONBody(r, &req); err != nil {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate input
	if req.Title == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.CreatorName == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "creator_name is required")
		return
	}

	// Generate poll ID
	pollID, err := auth.GenerateID(16)
	if err != nil {
		slog.Error("failed to generate poll ID", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to create poll")
		return
	}

	// Generate admin key
	adminKey := auth.GenerateAdminKey(pollID, h.cfg.AdminKeySalt)

	// Insert poll into database
	_, err = h.db.Exec(`
		INSERT INTO poll (id, title, description, creator_name, method, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, pollID, req.Title, req.Description, req.CreatorName, models.MethodBMJ, models.StatusDraft, time.Now())

	if err != nil {
		slog.Error("failed to insert poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to create poll")
		return
	}

	slog.Info("poll created", "poll_id", pollID, "creator", req.CreatorName)

	// Return response
	middleware.JSONResponse(w, http.StatusCreated, models.CreatePollResponse{
		PollID:   pollID,
		AdminKey: adminKey,
	})
}

// AddOption handles POST /polls/:id/options
func (h *PollHandler) AddOption(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	if pollID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "poll_id is required")
		return
	}

	// Validate admin key
	adminKey := r.Header.Get("X-Admin-Key")
	if err := auth.ValidateAdminKey(pollID, adminKey, h.cfg.AdminKeySalt); err != nil {
		middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid admin key")
		return
	}

	// Parse request
	var req models.AddOptionRequest
	if err := middleware.ParseJSONBody(r, &req); err != nil {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Label == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "label is required")
		return
	}

	// Check poll exists and is in draft status
	var status string
	err := h.db.QueryRow("SELECT status FROM poll WHERE id = $1", pollID).Scan(&status)
	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	if status != models.StatusDraft {
		middleware.ErrorResponse(w, http.StatusConflict, "Cannot add options to non-draft poll")
		return
	}

	// Generate option ID
	optionID, err := auth.GenerateID(12)
	if err != nil {
		slog.Error("failed to generate option ID", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to create option")
		return
	}

	// Insert option
	_, err = h.db.Exec(`
		INSERT INTO option (id, poll_id, label)
		VALUES ($1, $2, $3)
	`, optionID, pollID, req.Label)

	if err != nil {
		slog.Error("failed to insert option", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to create option")
		return
	}

	slog.Info("option added", "poll_id", pollID, "option_id", optionID)

	middleware.JSONResponse(w, http.StatusCreated, models.AddOptionResponse{
		OptionID: optionID,
	})
}

// PublishPoll handles POST /polls/:id/publish
func (h *PollHandler) PublishPoll(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	if pollID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "poll_id is required")
		return
	}

	// Validate admin key
	adminKey := r.Header.Get("X-Admin-Key")
	if err := auth.ValidateAdminKey(pollID, adminKey, h.cfg.AdminKeySalt); err != nil {
		middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid admin key")
		return
	}

	// Check poll exists and is in draft status
	var status string
	var optionCount int
	err := h.db.QueryRow(`
		SELECT p.status, COUNT(o.id)
		FROM poll p
		LEFT JOIN option o ON p.id = o.poll_id
		WHERE p.id = $1
		GROUP BY p.status
	`, pollID).Scan(&status, &optionCount)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	if status != models.StatusDraft {
		middleware.ErrorResponse(w, http.StatusConflict, "Poll is not in draft status")
		return
	}

	if optionCount < 2 {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Poll must have at least 2 options")
		return
	}

	// Generate share slug
	shareSlug := auth.GenerateShareSlug(pollID, h.cfg.PollSlugSalt)

	// Update poll to open status
	_, err = h.db.Exec(`
		UPDATE poll
		SET status = $1, share_slug = $2
		WHERE id = $3
	`, models.StatusOpen, shareSlug, pollID)

	if err != nil {
		slog.Error("failed to publish poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to publish poll")
		return
	}

	slog.Info("poll published", "poll_id", pollID, "share_slug", shareSlug)

	// Build share URL (could be configurable)
	baseURL := "https://quickly-pick.com" // TODO: Make this configurable
	shareURL := baseURL + "/polls/" + shareSlug

	middleware.JSONResponse(w, http.StatusOK, models.PublishPollResponse{
		ShareSlug: shareSlug,
		ShareURL:  shareURL,
	})
}

// GetPollAdmin handles GET /polls/:id/admin
// Returns poll details for admin access using poll ID and admin key
func (h *PollHandler) GetPollAdmin(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	if pollID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "poll_id is required")
		return
	}

	// Validate admin key
	adminKey := r.Header.Get("X-Admin-Key")
	if err := auth.ValidateAdminKey(pollID, adminKey, h.cfg.AdminKeySalt); err != nil {
		middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid admin key")
		return
	}

	// Get poll by ID
	var poll models.Poll
	err := h.db.QueryRow(`
		SELECT id, title, description, creator_name, method, status, 
		       share_slug, closes_at, closed_at, final_snapshot_id, created_at
		FROM poll
		WHERE id = $1
	`, pollID).Scan(
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

// ClosePoll handles POST /polls/:id/close
func (h *PollHandler) ClosePoll(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	if pollID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "poll_id is required")
		return
	}

	// Validate admin key
	adminKey := r.Header.Get("X-Admin-Key")
	if err := auth.ValidateAdminKey(pollID, adminKey, h.cfg.AdminKeySalt); err != nil {
		middleware.ErrorResponse(w, http.StatusUnauthorized, "Invalid admin key")
		return
	}

	// Check poll exists and is open
	var status string
	err := h.db.QueryRow("SELECT status FROM poll WHERE id = $1", pollID).Scan(&status)
	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Poll not found")
		return
	}
	if err != nil {
		slog.Error("failed to query poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	if status != models.StatusOpen {
		middleware.ErrorResponse(w, http.StatusConflict, "Poll is not open")
		return
	}

	// Compute BMJ results
	snapshotID, _ := auth.GenerateID(16)
	closedAt := time.Now()

	// Begin transaction
	tx, err := h.db.Begin()
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer tx.Rollback()

	// Update poll to closed
	_, err = tx.Exec(`
		UPDATE poll
		SET status = $1, closed_at = $2, final_snapshot_id = $3
		WHERE id = $4
	`, models.StatusClosed, closedAt, snapshotID, pollID)

	if err != nil {
		slog.Error("failed to close poll", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to close poll")
		return
	}

	// Insert placeholder snapshot (will be replaced with actual BMJ computation)
	_, err = tx.Exec(`
		INSERT INTO result_snapshot (id, poll_id, method, computed_at, payload)
		VALUES ($1, $2, $3, $4, $5)
	`, snapshotID, pollID, models.MethodBMJ, closedAt, `{"rankings":[],"inputs_hash":"placeholder"}`)

	if err != nil {
		slog.Error("failed to insert snapshot", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to save results")
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to close poll")
		return
	}

	slog.Info("poll closed", "poll_id", pollID, "snapshot_id", snapshotID)

	// Return response with placeholder snapshot
	middleware.JSONResponse(w, http.StatusOK, models.ClosePollResponse{
		ClosedAt: closedAt,
		Snapshot: models.ResultSnapshot{
			ID:         snapshotID,
			PollID:     pollID,
			Method:     models.MethodBMJ,
			ComputedAt: closedAt,
			Rankings:   []models.OptionStats{},
			InputsHash: "placeholder",
		},
	})
}
