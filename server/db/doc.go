// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package db handles database schema creation.

# Schema Creation

CreateSchema initializes all required tables:

	if err := db.CreateSchema(conn); err != nil {
		log.Fatal(err)
	}

Safe to call multiple times - uses IF NOT EXISTS for all tables and indexes.

# Tables

The schema includes:

  - poll: Poll metadata and lifecycle state
  - option: Voting options per poll
  - username_claim: Maps usernames to voter tokens
  - ballot: One ballot per voter per poll
  - score: Individual option scores (0-1)
  - result_snapshot: Immutable BMJ results
  - device: Registered devices
  - device_poll: Links devices to polls

# Relationships

	poll 1──* option
	poll 1──* username_claim
	poll 1──* ballot
	ballot 1──* score
	poll 1──* result_snapshot
	device *──* poll (via device_poll)

All foreign keys use ON DELETE CASCADE.

# Indexes

Performance indexes on:

  - poll.share_slug (unique)
  - poll.status
  - option.poll_id
  - ballot.poll_id
  - ballot.(poll_id, voter_token)
  - score.option_id
  - device.device_uuid (unique)
*/
package db
