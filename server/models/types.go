// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package models

import "time"

// Poll status constants
const (
	StatusDraft  = "draft"
	StatusOpen   = "open"
	StatusClosed = "closed"
)

// Voting method constants
const (
	MethodBMJ = "bmj"
)

// Request types

type CreatePollRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatorName string `json:"creator_name"`
}

type AddOptionRequest struct {
	Label string `json:"label"`
}

type ClaimUsernameRequest struct {
	Username string `json:"username"`
}

// option_id -> value01 (0.0 to 1.0)
type SubmitBallotRequest struct {
	Scores map[string]float64 `json:"scores"`
}

// Response types

type CreatePollResponse struct {
	PollID   string `json:"poll_id"`
	AdminKey string `json:"admin_key"`
}

type AddOptionResponse struct {
	OptionID string `json:"option_id"`
}

type PublishPollResponse struct {
	ShareSlug string `json:"share_slug"`
	ShareURL  string `json:"share_url"`
}

type ClaimUsernameResponse struct {
	VoterToken string `json:"voter_token"`
}

type SubmitBallotResponse struct {
	BallotID string `json:"ballot_id"`
	Message  string `json:"message"`
}

type ClosePollResponse struct {
	ClosedAt time.Time      `json:"closed_at"`
	Snapshot ResultSnapshot `json:"snapshot"`
}

// Domain types

type Poll struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	CreatorName     string     `json:"creator_name"`
	Method          string     `json:"method"`
	Status          string     `json:"status"`
	ShareSlug       *string    `json:"share_slug,omitempty"`
	ClosesAt        *time.Time `json:"closes_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	FinalSnapshotID *string    `json:"final_snapshot_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Option struct {
	ID     string `json:"id"`
	PollID string `json:"poll_id"`
	Label  string `json:"label"`
}

type PollWithOptions struct {
	Poll    Poll     `json:"poll"`
	Options []Option `json:"options"`
}

type Ballot struct {
	ID          string    `json:"id"`
	PollID      string    `json:"poll_id"`
	VoterToken  string    `json:"-"` // Never expose in JSON
	SubmittedAt time.Time `json:"submitted_at"`
	IPHash      *string   `json:"-"` // Never expose in JSON
	UserAgent   *string   `json:"-"` // Never expose in JSON
}

type Score struct {
	BallotID string  `json:"ballot_id"`
	OptionID string  `json:"option_id"`
	Value01  float64 `json:"value01"`
}

// BMJ Result Types

type OptionStats struct {
	OptionID string  `json:"option_id"`
	Label    string  `json:"label"`
	Median   float64 `json:"median"`
	P10      float64 `json:"p10"`
	P90      float64 `json:"p90"`
	Mean     float64 `json:"mean"`
	NegShare float64 `json:"neg_share"`
	Veto     bool    `json:"veto"`
	Rank     int     `json:"rank"` // 1-indexed ranking
}

type ResultSnapshot struct {
	ID         string        `json:"id"`
	PollID     string        `json:"poll_id"`
	Method     string        `json:"method"`
	ComputedAt time.Time     `json:"computed_at"`
	Rankings   []OptionStats `json:"rankings"`
	InputsHash string        `json:"inputs_hash"` // Hash of all ballot IDs for verification
}

// Device role constants
const (
	RoleVoter = "voter"
	RoleAdmin = "admin"
)

// Platform constants
const (
	PlatformIOS     = "ios"
	PlatformMacOS   = "macos"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
)

// Device request/response types

type RegisterDeviceRequest struct {
	Platform string `json:"platform"`
}

type RegisterDeviceResponse struct {
	DeviceID string `json:"device_id"`
	IsNew    bool   `json:"is_new"`
}

type DeviceInfo struct {
	ID         string    `json:"id"`
	DeviceUUID string    `json:"-"` // Never expose in JSON
	Platform   string    `json:"platform"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type DevicePollSummary struct {
	PollID      string    `json:"poll_id"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	ShareSlug   *string   `json:"share_slug,omitempty"`
	Role        string    `json:"role"`
	Username    *string   `json:"username,omitempty"`
	BallotCount int       `json:"ballot_count"`
	LinkedAt    time.Time `json:"linked_at"`
}

type GetMyPollsResponse struct {
	Polls []DevicePollSummary `json:"polls"`
}

type PollPreviewResponse struct {
	Title       string `json:"title"`
	Status      string `json:"status"`
	OptionCount int    `json:"option_count"`
	BallotCount int    `json:"ballot_count"`
}

// Error response

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
