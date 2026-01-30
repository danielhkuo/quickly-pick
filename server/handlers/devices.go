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

type DeviceHandler struct {
	db  *sql.DB
	cfg cliparse.Config
}

func NewDeviceHandler(db *sql.DB, cfg cliparse.Config) *DeviceHandler {
	return &DeviceHandler{db: db, cfg: cfg}
}

// Register handles POST /devices/register
// Registers a device and returns its device_id (or finds existing)
func (h *DeviceHandler) Register(w http.ResponseWriter, r *http.Request) {
	deviceUUID := r.Header.Get("X-Device-UUID")
	if deviceUUID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "X-Device-UUID header required")
		return
	}

	var req models.RegisterDeviceRequest
	if err := middleware.ParseJSONBody(r, &req); err != nil {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate platform
	if !isValidPlatform(req.Platform) {
		middleware.ErrorResponse(w, http.StatusBadRequest, "platform must be one of: ios, macos, android, web")
		return
	}

	// Check if device already exists
	var existingID string
	err := h.db.QueryRow(`
		SELECT id FROM device WHERE device_uuid = $1
	`, deviceUUID).Scan(&existingID)

	if err == nil {
		// Device exists, update last_seen_at
		_, err = h.db.Exec(`
			UPDATE device SET last_seen_at = $1 WHERE id = $2
		`, time.Now(), existingID)
		if err != nil {
			slog.Error("failed to update device last_seen_at", "error", err)
		}

		slog.Info("device registered (existing)", "device_id", existingID)
		middleware.JSONResponse(w, http.StatusOK, models.RegisterDeviceResponse{
			DeviceID: existingID,
			IsNew:    false,
		})
		return
	}

	if err != sql.ErrNoRows {
		slog.Error("failed to query device", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Create new device
	deviceID, err := auth.GenerateID(16)
	if err != nil {
		slog.Error("failed to generate device ID", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to register device")
		return
	}

	now := time.Now()
	_, err = h.db.Exec(`
		INSERT INTO device (id, device_uuid, platform, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5)
	`, deviceID, deviceUUID, req.Platform, now, now)

	if err != nil {
		slog.Error("failed to insert device", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Failed to register device")
		return
	}

	slog.Info("device registered (new)", "device_id", deviceID, "platform", req.Platform)

	middleware.JSONResponse(w, http.StatusCreated, models.RegisterDeviceResponse{
		DeviceID: deviceID,
		IsNew:    true,
	})
}

// GetMe handles GET /devices/me
// Returns current device info
func (h *DeviceHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	deviceUUID := r.Header.Get("X-Device-UUID")
	if deviceUUID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "X-Device-UUID header required")
		return
	}

	var device models.DeviceInfo
	err := h.db.QueryRow(`
		SELECT id, platform, created_at, last_seen_at
		FROM device
		WHERE device_uuid = $1
	`, deviceUUID).Scan(&device.ID, &device.Platform, &device.CreatedAt, &device.LastSeenAt)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Device not registered")
		return
	}
	if err != nil {
		slog.Error("failed to query device", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Update last_seen_at
	_, err = h.db.Exec(`
		UPDATE device SET last_seen_at = $1 WHERE id = $2
	`, time.Now(), device.ID)
	if err != nil {
		slog.Error("failed to update device last_seen_at", "error", err)
	}

	middleware.JSONResponse(w, http.StatusOK, device)
}

// GetMyPolls handles GET /devices/my-polls
// Returns polls where this device is admin or voter
func (h *DeviceHandler) GetMyPolls(w http.ResponseWriter, r *http.Request) {
	deviceUUID := r.Header.Get("X-Device-UUID")
	if deviceUUID == "" {
		middleware.ErrorResponse(w, http.StatusBadRequest, "X-Device-UUID header required")
		return
	}

	// Get device ID
	var deviceID string
	err := h.db.QueryRow(`
		SELECT id FROM device WHERE device_uuid = $1
	`, deviceUUID).Scan(&deviceID)

	if err == sql.ErrNoRows {
		middleware.ErrorResponse(w, http.StatusNotFound, "Device not registered")
		return
	}
	if err != nil {
		slog.Error("failed to query device", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Update last_seen_at
	_, err = h.db.Exec(`
		UPDATE device SET last_seen_at = $1 WHERE id = $2
	`, time.Now(), deviceID)
	if err != nil {
		slog.Error("failed to update device last_seen_at", "error", err)
	}

	// Get polls linked to this device with metadata
	rows, err := h.db.Query(`
		SELECT
			p.id,
			p.title,
			p.status,
			p.share_slug,
			dp.role,
			dp.voter_token,
			dp.linked_at,
			(SELECT COUNT(*) FROM ballot b WHERE b.poll_id = p.id) as ballot_count
		FROM device_poll dp
		JOIN poll p ON dp.poll_id = p.id
		WHERE dp.device_id = $1
		ORDER BY dp.linked_at DESC
	`, deviceID)

	if err != nil {
		slog.Error("failed to query device polls", "error", err)
		middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	polls := []models.DevicePollSummary{}
	for rows.Next() {
		var summary models.DevicePollSummary
		var voterToken sql.NullString

		if err := rows.Scan(
			&summary.PollID,
			&summary.Title,
			&summary.Status,
			&summary.ShareSlug,
			&summary.Role,
			&voterToken,
			&summary.LinkedAt,
			&summary.BallotCount,
		); err != nil {
			slog.Error("failed to scan poll", "error", err)
			middleware.ErrorResponse(w, http.StatusInternalServerError, "Database error")
			return
		}

		// Get username if voter_token is present
		if voterToken.Valid {
			var username string
			err := h.db.QueryRow(`
				SELECT username FROM username_claim
				WHERE poll_id = $1 AND voter_token = $2
			`, summary.PollID, voterToken.String).Scan(&username)
			if err == nil {
				summary.Username = &username
			}
		}

		polls = append(polls, summary)
	}

	middleware.JSONResponse(w, http.StatusOK, models.GetMyPollsResponse{
		Polls: polls,
	})
}

// GetOrCreateDevice looks up or creates a device record from the X-Device-UUID header.
// Returns device ID and whether it was newly created. Returns empty string if no header.
func GetOrCreateDevice(db *sql.DB, r *http.Request) (string, error) {
	deviceUUID := r.Header.Get("X-Device-UUID")
	if deviceUUID == "" {
		return "", nil
	}

	// Check if device exists
	var deviceID string
	err := db.QueryRow(`
		SELECT id FROM device WHERE device_uuid = $1
	`, deviceUUID).Scan(&deviceID)

	if err == nil {
		// Update last_seen_at
		_, _ = db.Exec(`UPDATE device SET last_seen_at = $1 WHERE id = $2`, time.Now(), deviceID)
		return deviceID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// Create new device with 'web' as default platform
	// (actual platform is set via /devices/register)
	deviceID, err = auth.GenerateID(16)
	if err != nil {
		return "", err
	}

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO device (id, device_uuid, platform, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5)
	`, deviceID, deviceUUID, models.PlatformWeb, now, now)

	if err != nil {
		return "", err
	}

	return deviceID, nil
}

// LinkDeviceToPoll creates an association between a device and a poll
func LinkDeviceToPoll(db *sql.DB, deviceID, pollID, role string, voterToken *string) error {
	if deviceID == "" {
		return nil
	}

	var vt sql.NullString
	if voterToken != nil {
		vt = sql.NullString{String: *voterToken, Valid: true}
	}

	// Use INSERT ... ON CONFLICT to handle re-linking (e.g., voter becomes admin)
	_, err := db.Exec(`
		INSERT INTO device_poll (device_id, poll_id, voter_token, role, linked_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (device_id, poll_id) DO UPDATE SET
			role = CASE WHEN device_poll.role = 'admin' THEN 'admin' ELSE EXCLUDED.role END,
			voter_token = COALESCE(device_poll.voter_token, EXCLUDED.voter_token)
	`, deviceID, pollID, vt, role, time.Now())

	return err
}

func isValidPlatform(platform string) bool {
	switch platform {
	case models.PlatformIOS, models.PlatformMacOS, models.PlatformAndroid, models.PlatformWeb:
		return true
	}
	return false
}
