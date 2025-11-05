// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package db

import (
	"database/sql"
	"fmt"
)

// CreateSchema creates all tables needed for the application.
// Safe to call multiple times - uses IF NOT EXISTS.
func CreateSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

const schema = `
-- Polls
CREATE TABLE IF NOT EXISTS poll (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    creator_name TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'bmj',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
    share_slug TEXT UNIQUE,
    closes_at TIMESTAMP,
    closed_at TIMESTAMP,
    final_snapshot_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_poll_share_slug ON poll(share_slug);
CREATE INDEX IF NOT EXISTS idx_poll_status ON poll(status);

-- Options
CREATE TABLE IF NOT EXISTS option (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    label TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_option_poll_id ON option(poll_id);

-- Username Claims
CREATE TABLE IF NOT EXISTS username_claim (
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    voter_token TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (poll_id, voter_token),
    UNIQUE (poll_id, username)
);

CREATE INDEX IF NOT EXISTS idx_username_claim_poll_id ON username_claim(poll_id);

-- Ballots
CREATE TABLE IF NOT EXISTS ballot (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    voter_token TEXT NOT NULL,
    submitted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_hash TEXT,
    user_agent TEXT,
    UNIQUE (poll_id, voter_token)
);

CREATE INDEX IF NOT EXISTS idx_ballot_poll_id ON ballot(poll_id);
CREATE INDEX IF NOT EXISTS idx_ballot_voter_token ON ballot(poll_id, voter_token);

-- Scores
CREATE TABLE IF NOT EXISTS score (
    ballot_id TEXT NOT NULL REFERENCES ballot(id) ON DELETE CASCADE,
    option_id TEXT NOT NULL REFERENCES option(id) ON DELETE CASCADE,
    value01 REAL NOT NULL CHECK (value01 >= 0 AND value01 <= 1),
    PRIMARY KEY (ballot_id, option_id)
);

CREATE INDEX IF NOT EXISTS idx_score_option_id ON score(option_id);

-- Result Snapshots
CREATE TABLE IF NOT EXISTS result_snapshot (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_result_snapshot_poll_id ON result_snapshot(poll_id);
`
