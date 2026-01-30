// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package handlers contains HTTP request handlers for the Quickly Pick API.

# Handler Types

Each handler is a struct with database and config dependencies:

  - PollHandler: Poll lifecycle (create, publish, close)
  - VotingHandler: Username claims and ballot submission
  - ResultsHandler: Poll info and results retrieval
  - DeviceHandler: Device registration and poll history

Handlers are created via constructor functions that accept *sql.DB and Config:

	pollHandler := handlers.NewPollHandler(db, cfg)

# Poll Lifecycle

Polls progress through three states: draft → open → closed

	POST /polls           → CreatePoll (returns admin_key)
	POST /polls/{id}/options → AddOption (draft only)
	POST /polls/{id}/publish → PublishPoll (generates share_slug)
	POST /polls/{id}/close   → ClosePoll (computes BMJ results)

Admin operations require the X-Admin-Key header.

# Voting Flow

Voters interact via the share slug:

	POST /polls/{slug}/claim-username → ClaimUsername (returns voter_token)
	POST /polls/{slug}/ballots        → SubmitBallot (create or update)

Voter operations require the X-Voter-Token header.

# BMJ Algorithm

The Balanced Majority Judgment algorithm is implemented in bmj.go:

	rankings, err := ComputeBMJRankings(db, pollID)

This computes median, P10, P90, mean, negative share, and veto status
for each option, then ranks them lexicographically.

# Device Tracking

Optional device tracking for native apps:

	POST /devices/register → Register
	GET /devices/me        → GetMe
	GET /devices/my-polls  → GetMyPolls

Device operations require the X-Device-UUID header.
*/
package handlers
