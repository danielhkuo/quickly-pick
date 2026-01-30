# Database Schema

Quickly Pick uses PostgreSQL for data storage. This document describes the database schema, relationships, and design decisions.

## Tables Overview

| Table | Purpose |
|-------|---------|
| `poll` | Poll metadata and lifecycle state |
| `option` | Voting options for each poll |
| `username_claim` | Maps usernames to voter tokens per poll |
| `ballot` | One ballot per voter per poll |
| `score` | Individual scores per option per ballot |
| `result_snapshot` | Immutable BMJ results when poll closes |
| `device` | Registered devices (iOS/macOS/Android/web) |
| `device_poll` | Links devices to polls with role |

## Entity Relationship Diagram

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   device    │       │    poll     │       │   option    │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ id (PK)     │       │ id (PK)     │◄──────│ id (PK)     │
│ device_uuid │       │ title       │       │ poll_id(FK) │
│ platform    │       │ description │       │ label       │
│ created_at  │       │ creator_name│       └─────────────┘
│ last_seen_at│       │ method      │
└──────┬──────┘       │ status      │       ┌─────────────┐
       │              │ share_slug  │       │   ballot    │
       │              │ closes_at   │       ├─────────────┤
       │              │ closed_at   │◄──────│ id (PK)     │
       │              │ final_snap  │       │ poll_id(FK) │
       │              │ created_at  │       │ voter_token │
       │              └──────┬──────┘       │ submitted_at│
       │                     │              │ ip_hash     │
       │              ┌──────┴──────┐       │ user_agent  │
       │              │             │       └──────┬──────┘
       │              ▼             │              │
       │       ┌─────────────┐     │       ┌──────┴──────┐
       │       │username_claim│     │       │   score     │
       │       ├─────────────┤     │       ├─────────────┤
       │       │poll_id (PK) │     │       │ballot_id(PK)│
       │       │voter_token(P│     │       │option_id(PK)│
       │       │username     │     │       │ value01     │
       │       │created_at   │     │       └─────────────┘
       │       └─────────────┘     │
       │                           │       ┌─────────────┐
       │                           │       │result_snap  │
       ▼                           │       ├─────────────┤
┌─────────────┐                    └──────►│ id (PK)     │
│ device_poll │                            │ poll_id(FK) │
├─────────────┤                            │ method      │
│device_id(PK)│                            │ computed_at │
│poll_id (PK) │                            │ payload     │
│voter_token  │                            └─────────────┘
│ role        │
│ linked_at   │
└─────────────┘
```

## Table Definitions

### poll

Stores poll metadata and tracks lifecycle state.

```sql
CREATE TABLE poll (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    creator_name TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'bmj',
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'open', 'closed')),
    share_slug TEXT UNIQUE,
    closes_at TIMESTAMP,
    closed_at TIMESTAMP,
    final_snapshot_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | 32-char hex random ID |
| `title` | TEXT | Poll question |
| `description` | TEXT | Optional details |
| `creator_name` | TEXT | Display name of creator |
| `method` | TEXT | Voting algorithm (always `bmj`) |
| `status` | TEXT | Lifecycle: `draft` → `open` → `closed` |
| `share_slug` | TEXT | URL-friendly sharing code (base62) |
| `closes_at` | TIMESTAMP | Optional scheduled close time |
| `closed_at` | TIMESTAMP | Actual close timestamp |
| `final_snapshot_id` | TEXT | Reference to result snapshot |
| `created_at` | TIMESTAMP | Creation timestamp |

**Indexes:**
- `idx_poll_share_slug` on `share_slug`
- `idx_poll_status` on `status`

---

### option

Voting options belonging to a poll.

```sql
CREATE TABLE option (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    label TEXT NOT NULL
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | 24-char hex random ID |
| `poll_id` | TEXT | FK to poll |
| `label` | TEXT | Display text for option |

**Indexes:**
- `idx_option_poll_id` on `poll_id`

---

### username_claim

Maps display names to voter tokens for a specific poll.

```sql
CREATE TABLE username_claim (
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    voter_token TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (poll_id, voter_token),
    UNIQUE (poll_id, username)
);
```

| Column | Type | Description |
|--------|------|-------------|
| `poll_id` | TEXT | FK to poll |
| `username` | TEXT | Display name (unique per poll) |
| `voter_token` | TEXT | Secret token for ballot submission |
| `created_at` | TIMESTAMP | Claim timestamp |

**Constraints:**
- Composite PK: `(poll_id, voter_token)`
- Unique: `(poll_id, username)` - one name per poll

---

### ballot

One ballot per voter per poll. Can be updated while poll is open.

```sql
CREATE TABLE ballot (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    voter_token TEXT NOT NULL,
    submitted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_hash TEXT,
    user_agent TEXT,
    UNIQUE (poll_id, voter_token)
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | 32-char hex random ID |
| `poll_id` | TEXT | FK to poll |
| `voter_token` | TEXT | Identifies voter (not exposed) |
| `submitted_at` | TIMESTAMP | Last update timestamp |
| `ip_hash` | TEXT | Hashed IP for fraud detection |
| `user_agent` | TEXT | Browser/app info |

**Indexes:**
- `idx_ballot_poll_id` on `poll_id`
- `idx_ballot_voter_token` on `(poll_id, voter_token)`

---

### score

Individual scores for each option in a ballot.

```sql
CREATE TABLE score (
    ballot_id TEXT NOT NULL REFERENCES ballot(id) ON DELETE CASCADE,
    option_id TEXT NOT NULL REFERENCES option(id) ON DELETE CASCADE,
    value01 REAL NOT NULL CHECK (value01 >= 0 AND value01 <= 1),
    PRIMARY KEY (ballot_id, option_id)
);
```

| Column | Type | Description |
|--------|------|-------------|
| `ballot_id` | TEXT | FK to ballot |
| `option_id` | TEXT | FK to option |
| `value01` | REAL | Score from 0.0 (hate) to 1.0 (love) |

**Constraints:**
- Composite PK: `(ballot_id, option_id)`
- Check: `value01` between 0 and 1

---

### result_snapshot

Immutable results computed when poll closes.

```sql
CREATE TABLE result_snapshot (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | 32-char hex random ID |
| `poll_id` | TEXT | FK to poll |
| `method` | TEXT | Algorithm used (`bmj`) |
| `computed_at` | TIMESTAMP | Computation timestamp |
| `payload` | JSONB | Full BMJ results as JSON |

**Payload structure:**
```json
{
  "rankings": [
    {
      "option_id": "...",
      "label": "...",
      "median": 0.6,
      "p10": 0.2,
      "p90": 0.9,
      "mean": 0.55,
      "neg_share": 0.1,
      "veto": false,
      "rank": 1
    }
  ],
  "inputs_hash": "5-ballots"
}
```

---

### device

Registered devices for cross-session poll tracking.

```sql
CREATE TABLE device (
    id TEXT PRIMARY KEY,
    device_uuid TEXT NOT NULL UNIQUE,
    platform TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | Internal device ID |
| `device_uuid` | TEXT | Client-provided UUID (from X-Device-UUID header) |
| `platform` | TEXT | `ios`, `macos`, `android`, or `web` |
| `created_at` | TIMESTAMP | First registration |
| `last_seen_at` | TIMESTAMP | Last API call |

---

### device_poll

Links devices to polls with their role.

```sql
CREATE TABLE device_poll (
    device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    voter_token TEXT,
    role TEXT NOT NULL DEFAULT 'voter',
    linked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, poll_id)
);
```

| Column | Type | Description |
|--------|------|-------------|
| `device_id` | TEXT | FK to device |
| `poll_id` | TEXT | FK to poll |
| `voter_token` | TEXT | If voter, their token (for username lookup) |
| `role` | TEXT | `admin` or `voter` |
| `linked_at` | TIMESTAMP | Association timestamp |

## Cascade Delete Behavior

When a poll is deleted:
- All options are deleted (CASCADE)
- All username claims are deleted (CASCADE)
- All ballots are deleted (CASCADE)
- All scores are deleted (CASCADE via ballot)
- All result snapshots are deleted (CASCADE)
- All device_poll links are deleted (CASCADE)

When a device is deleted:
- All device_poll links are deleted (CASCADE)
- Polls themselves remain intact

## Design Decisions

### Why TEXT for IDs?

- Hex-encoded random bytes (16 bytes = 32 chars)
- URL-safe without encoding
- Easy to generate without database round-trip
- No sequential prediction (security)

### Why separate ballot and score tables?

- Allows atomic ballot updates (delete old scores, insert new)
- Tracks ballot metadata separately from scores
- Efficient queries for BMJ computation (join option → score)

### Why JSONB for result snapshots?

- Flexible schema for algorithm results
- Immutable record of computation
- Easy to add fields without migrations
- Efficient PostgreSQL storage

### Why soft delete with status instead of DELETE?

Polls are never deleted, only closed. This preserves:
- Audit trail
- Historical results
- Device poll history
