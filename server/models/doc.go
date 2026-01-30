// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package models defines request, response, and domain types for the API.

# Request Types

Types for parsing incoming JSON:

  - CreatePollRequest: title, description, creator_name
  - AddOptionRequest: label
  - ClaimUsernameRequest: username
  - SubmitBallotRequest: scores (map[string]float64)
  - RegisterDeviceRequest: platform

# Response Types

Types for JSON responses:

  - CreatePollResponse: poll_id, admin_key
  - AddOptionResponse: option_id
  - PublishPollResponse: share_slug, share_url
  - ClaimUsernameResponse: voter_token
  - SubmitBallotResponse: ballot_id, message
  - ClosePollResponse: closed_at, snapshot
  - ErrorResponse: error, message

# Domain Types

Internal data structures:

  - Poll: poll metadata and lifecycle state
  - Option: voting option with label
  - Ballot: voter submission metadata
  - Score: individual option score (0-1)
  - OptionStats: BMJ statistics for an option
  - ResultSnapshot: immutable result record

# Constants

Status values:

	StatusDraft  = "draft"
	StatusOpen   = "open"
	StatusClosed = "closed"

Voting method:

	MethodBMJ = "bmj"

Device roles:

	RoleVoter = "voter"
	RoleAdmin = "admin"

Platforms:

	PlatformIOS     = "ios"
	PlatformMacOS   = "macos"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
*/
package models
