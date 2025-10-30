# Database Schema

The Quickly Pick database uses PostgreSQL with a normalized schema designed for polling operations.

## Entity Relationship Diagram

```
poll (1) ──── (N) option
  │
  └── (1) ──── (N) username_claim
  │
  └── (1) ──── (N) ballot ──── (N) score ──── (1) option
  │
  └── (1) ──── (N) result_snapshot
```

## Tables

### poll

Stores poll metadata and state.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | Unique poll identifier |
| `title` | TEXT NOT NULL | Poll title |
| `description` | TEXT | Optional poll description |
| `creator_name` | TEXT NOT NULL | Name of poll creator |
| `method` | TEXT NOT NULL DEFAULT 'bmj' | Voting method (always BMJ) |
| `status` | TEXT NOT NULL DEFAULT 'draft' | Poll state: draft/open/closed |
| `share_slug` | TEXT UNIQUE | Public URL slug for sharing |
| `closes_at` | TIMESTAMP | Optional auto-close time |
| `closed_at` | TIMESTAMP | Actual close timestamp |
| `final_snapshot_id` | TEXT | Reference to final results |
| `created_at` | TIMESTAMP NOT NULL DEFAULT NOW() | Creation timestamp |

**Constraints:**
- `status` must be one of: 'draft', 'open', 'closed'
- `share_slug` is unique across all polls

### option

Stores poll options/choices.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | Unique option identifier |
| `poll_id` | TEXT NOT NULL | Foreign key to poll |
| `label` | TEXT NOT NULL | Option display text |

**Foreign Keys:**
- `poll_id` REFERENCES `poll(id)` ON DELETE CASCADE

### username_claim

Maps usernames to voter tokens within a poll.

| Column | Type | Description |
|--------|------|-------------|
| `poll_id` | TEXT NOT NULL | Foreign key to poll |
| `username` | TEXT NOT NULL | Claimed username |
| `voter_token` | TEXT NOT NULL | Unique voter identifier |
| `created_at` | TIMESTAMP NOT NULL DEFAULT NOW() | Claim timestamp |

**Primary Key:** `(poll_id, voter_token)`

**Unique Constraints:**
- `(poll_id, username)` - One username per poll

**Foreign Keys:**
- `poll_id` REFERENCES `poll(id)` ON DELETE CASCADE

### ballot

Stores submitted votes.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | Unique ballot identifier |
| `poll_id` | TEXT NOT NULL | Foreign key to poll |
| `voter_token` | TEXT NOT NULL | Voter identifier |
| `submitted_at` | TIMESTAMP NOT NULL DEFAULT NOW() | Submission timestamp |
| `ip_hash` | TEXT | Hashed IP for abuse detection |
| `user_agent` | TEXT | Browser user agent |

**Unique Constraints:**
- `(poll_id, voter_token)` - One ballot per voter per poll

**Foreign Keys:**
- `poll_id` REFERENCES `poll(id)` ON DELETE CASCADE

### score

Stores individual option scores within ballots.

| Column | Type | Description |
|--------|------|-------------|
| `ballot_id` | TEXT NOT NULL | Foreign key to ballot |
| `option_id` | TEXT NOT NULL | Foreign key to option |
| `value01` | REAL NOT NULL | Score value between 0 and 1 |

**Primary Key:** `(ballot_id, option_id)`

**Constraints:**
- `value01` must be between 0 and 1 (inclusive)

**Foreign Keys:**
- `ballot_id` REFERENCES `ballot(id)` ON DELETE CASCADE
- `option_id` REFERENCES `option(id)` ON DELETE CASCADE

### result_snapshot

Stores computed poll results.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | Unique snapshot identifier |
| `poll_id` | TEXT NOT NULL | Foreign key to poll |
| `method` | TEXT NOT NULL | Algorithm used (always 'bmj') |
| `computed_at` | TIMESTAMP NOT NULL DEFAULT NOW() | Computation timestamp |
| `payload` | JSONB NOT NULL | Results data and metadata |

**Foreign Keys:**
- `poll_id` REFERENCES `poll(id)` ON DELETE CASCADE

## Indexes

Performance indexes are created for common query patterns:

```sql
-- Poll lookups
CREATE INDEX idx_poll_share_slug ON poll(share_slug);
CREATE INDEX idx_poll_status ON poll(status);

-- Option queries
CREATE INDEX idx_option_poll_id ON option(poll_id);

-- Username claims
CREATE INDEX idx_username_claim_poll_id ON username_claim(poll_id);

-- Ballot queries
CREATE INDEX idx_ballot_poll_id ON ballot(poll_id);
CREATE INDEX idx_ballot_voter_token ON ballot(poll_id, voter_token);

-- Score aggregation
CREATE INDEX idx_score_option_id ON score(option_id);

-- Result snapshots
CREATE INDEX idx_result_snapshot_poll_id ON result_snapshot(poll_id);
```

## Data Flow

### Poll Creation Flow

1. `INSERT INTO poll` (status='draft')
2. `INSERT INTO option` (multiple)
3. `UPDATE poll SET status='open', share_slug=?`

### Voting Flow

1. `INSERT INTO username_claim` (if new username)
2. `INSERT INTO ballot ON CONFLICT UPDATE` (upsert)
3. `DELETE FROM score WHERE ballot_id=?`
4. `INSERT INTO score` (multiple)

### Results Flow

1. `UPDATE poll SET status='closed', closed_at=NOW()`
2. Compute BMJ ranking from scores
3. `INSERT INTO result_snapshot`
4. `UPDATE poll SET final_snapshot_id=?`

## BMJ Calculation Query

The core BMJ ranking query aggregates scores per option:

```sql
WITH signed_scores AS (
  SELECT 
    option_id,
    (value01 * 2.0 - 1.0) AS s
  FROM score 
  WHERE option_id IN (SELECT id FROM option WHERE poll_id = $1)
),
aggregates AS (
  SELECT
    option_id,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY s) AS median,
    percentile_cont(0.10) WITHIN GROUP (ORDER BY s) AS p10,
    percentile_cont(0.90) WITHIN GROUP (ORDER BY s) AS p90,
    avg(s) AS mean,
    avg(CASE WHEN s < 0 THEN 1.0 ELSE 0.0 END) AS neg_share
  FROM signed_scores 
  GROUP BY option_id
)
SELECT 
  option_id,
  median, p10, p90, mean, neg_share,
  (CASE WHEN neg_share >= 0.33 AND median <= 0 THEN 1 ELSE 0 END) AS veto_rank
FROM aggregates
ORDER BY 
  veto_rank ASC,     -- Non-vetoed options first
  median DESC,       -- Higher median wins
  p10 DESC,          -- Higher 10th percentile wins
  p90 DESC,          -- Higher 90th percentile wins  
  mean DESC,         -- Higher mean wins
  option_id ASC;     -- Stable tie-breaking
```

## Migration Strategy

The schema is designed to be created idempotently using `CREATE TABLE IF NOT EXISTS`. For production deployments, consider using a proper migration tool like:

- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- [atlas](https://atlasgo.io/)

## Backup Considerations

Key data to backup:
- All tables (full schema)
- Pay special attention to `result_snapshot` for audit trails
- Consider archiving closed polls after a retention period