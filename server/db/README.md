# Database Package

Database schema management and operations for the Quickly Pick polling service.

## Overview

This package handles database schema creation and provides the foundation for all database operations. It defines the complete PostgreSQL schema with proper constraints, indexes, and relationships.

## Functions

### `CreateSchema(db *sql.DB) error`

Creates all required tables and indexes for the application. Safe to call multiple times as it uses `CREATE TABLE IF NOT EXISTS`.

**Parameters:**
- `db *sql.DB` - Open database connection

**Returns:**
- `error` - Schema creation error, if any

**Usage:**
```go
import (
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/danielhkuo/quickly-pick/db"
)

func main() {
    dbConn, err := sql.Open("postgres", databaseURL)
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    defer dbConn.Close()

    // Create schema
    if err := db.CreateSchema(dbConn); err != nil {
        log.Fatal("Schema creation failed:", err)
    }
    
    log.Println("Database schema ready")
}
```

## Database Schema

The schema consists of 6 main tables with proper relationships and constraints:

### Tables Overview

```
poll (1) ──── (N) option
  │
  ├── (1) ──── (N) username_claim
  │
  ├── (1) ──── (N) ballot ──── (N) score ──── (1) option
  │
  └── (1) ──── (N) result_snapshot
```

### poll
Main poll metadata and state management.

```sql
CREATE TABLE poll (
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
```

**Key Features:**
- Status constraint ensures valid poll states
- Unique share_slug for public access
- Timestamps for lifecycle tracking

### option
Poll choices/options.

```sql
CREATE TABLE option (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    label TEXT NOT NULL
);
```

**Key Features:**
- Cascade delete when poll is removed
- Simple label-based options

### username_claim
Maps usernames to voter tokens within polls.

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

**Key Features:**
- Composite primary key prevents duplicate tokens
- Unique constraint prevents username conflicts
- One username per poll enforcement

### ballot
Submitted votes with metadata.

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

**Key Features:**
- One ballot per voter per poll
- IP hash for abuse detection (privacy-preserving)
- User agent tracking for analytics

### score
Individual option scores within ballots.

```sql
CREATE TABLE score (
    ballot_id TEXT NOT NULL REFERENCES ballot(id) ON DELETE CASCADE,
    option_id TEXT NOT NULL REFERENCES option(id) ON DELETE CASCADE,
    value01 REAL NOT NULL CHECK (value01 >= 0 AND value01 <= 1),
    PRIMARY KEY (ballot_id, option_id)
);
```

**Key Features:**
- Composite primary key (one score per option per ballot)
- Value constraint ensures valid slider range [0,1]
- Cascade delete maintains referential integrity

### result_snapshot
Computed poll results storage.

```sql
CREATE TABLE result_snapshot (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL REFERENCES poll(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL
);
```

**Key Features:**
- JSONB payload for flexible result storage
- Method field for algorithm versioning
- Immutable snapshots for audit trail

## Indexes

Performance indexes are automatically created:

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

## Data Flow Examples

### Poll Creation Flow

```go
// 1. Insert poll
pollID := auth.GenerateID(16)
_, err := db.Exec(`
    INSERT INTO poll (id, title, description, creator_name) 
    VALUES ($1, $2, $3, $4)
`, pollID, title, description, creatorName)

// 2. Add options
for _, label := range options {
    optionID := auth.GenerateID(12)
    _, err := db.Exec(`
        INSERT INTO option (id, poll_id, label) 
        VALUES ($1, $2, $3)
    `, optionID, pollID, label)
}

// 3. Publish poll
shareSlug := auth.GenerateShareSlug(pollID, salt)
_, err = db.Exec(`
    UPDATE poll 
    SET status = 'open', share_slug = $1 
    WHERE id = $2
`, shareSlug, pollID)
```

### Voting Flow

```go
// 1. Claim username (if new)
voterToken := auth.GenerateVoterToken()
_, err := db.Exec(`
    INSERT INTO username_claim (poll_id, username, voter_token) 
    VALUES ($1, $2, $3) 
    ON CONFLICT (poll_id, username) DO NOTHING
`, pollID, username, voterToken)

// 2. Upsert ballot
ballotID := auth.GenerateID(16)
_, err = db.Exec(`
    INSERT INTO ballot (id, poll_id, voter_token, ip_hash, user_agent) 
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (poll_id, voter_token) 
    DO UPDATE SET submitted_at = NOW(), ip_hash = $4, user_agent = $5
`, ballotID, pollID, voterToken, ipHash, userAgent)

// 3. Replace scores
tx, _ := db.Begin()
tx.Exec(`DELETE FROM score WHERE ballot_id = $1`, ballotID)
for _, score := range scores {
    tx.Exec(`
        INSERT INTO score (ballot_id, option_id, value01) 
        VALUES ($1, $2, $3)
    `, ballotID, score.OptionID, score.Value01)
}
tx.Commit()
```

### Results Computation

```go
// 1. Close poll
_, err := db.Exec(`
    UPDATE poll 
    SET status = 'closed', closed_at = NOW() 
    WHERE id = $1
`, pollID)

// 2. Compute BMJ ranking
rows, err := db.Query(`
    WITH signed_scores AS (
        SELECT option_id, (value01 * 2.0 - 1.0) AS s
        FROM score s
        JOIN ballot b ON s.ballot_id = b.id
        WHERE b.poll_id = $1
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
        option_id, median, p10, p90, mean, neg_share,
        (CASE WHEN neg_share >= 0.33 AND median <= 0 THEN 1 ELSE 0 END) AS veto_rank
    FROM aggregates
    ORDER BY veto_rank ASC, median DESC, p10 DESC, p90 DESC, mean DESC, option_id ASC
`, pollID)

// 3. Store snapshot
snapshotID := auth.GenerateID(16)
payload := buildResultPayload(ranking)
_, err = db.Exec(`
    INSERT INTO result_snapshot (id, poll_id, method, payload) 
    VALUES ($1, $2, 'bmj', $3)
`, snapshotID, pollID, payload)

// 4. Link snapshot to poll
_, err = db.Exec(`
    UPDATE poll 
    SET final_snapshot_id = $1 
    WHERE id = $2
`, snapshotID, pollID)
```

## Schema Validation

The schema includes several validation constraints:

### Data Integrity
- Foreign key constraints maintain referential integrity
- Unique constraints prevent duplicates
- Check constraints validate data ranges

### Business Logic
- Poll status must be 'draft', 'open', or 'closed'
- Score values must be between 0 and 1
- One ballot per voter per poll
- One username per poll

## Migration Strategy

For production deployments, consider using a migration tool:

### golang-migrate Example

```sql
-- 001_initial_schema.up.sql
CREATE TABLE poll (
    id TEXT PRIMARY KEY,
    -- ... rest of schema
);

-- 001_initial_schema.down.sql
DROP TABLE IF EXISTS result_snapshot;
DROP TABLE IF EXISTS score;
DROP TABLE IF EXISTS ballot;
DROP TABLE IF EXISTS username_claim;
DROP TABLE IF EXISTS option;
DROP TABLE IF EXISTS poll;
```

### Schema Versioning

Track schema version in database:

```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO schema_version (version) VALUES (1);
```

## Performance Considerations

### Query Optimization
- Use prepared statements for repeated queries
- Leverage indexes for common query patterns
- Consider query complexity for large datasets

### Connection Management
- Use connection pooling in production
- Set appropriate connection limits
- Monitor connection usage

### Storage Optimization
- Consider partitioning for very large datasets
- Archive old closed polls periodically
- Monitor disk usage growth

## Testing

### Schema Tests

```go
func TestCreateSchema(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    err := CreateSchema(db)
    if err != nil {
        t.Fatalf("CreateSchema failed: %v", err)
    }
    
    // Verify tables exist
    tables := []string{"poll", "option", "username_claim", "ballot", "score", "result_snapshot"}
    for _, table := range tables {
        var exists bool
        err := db.QueryRow(`
            SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_name = $1
            )
        `, table).Scan(&exists)
        
        if err != nil || !exists {
            t.Errorf("Table %s does not exist", table)
        }
    }
}
```

### Constraint Tests

```go
func TestConstraints(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Test poll status constraint
    _, err := db.Exec(`
        INSERT INTO poll (id, title, creator_name, status) 
        VALUES ('test', 'Test', 'Alice', 'invalid')
    `)
    if err == nil {
        t.Error("Expected error for invalid poll status")
    }
    
    // Test score value constraint
    _, err = db.Exec(`
        INSERT INTO score (ballot_id, option_id, value01) 
        VALUES ('ballot1', 'option1', 1.5)
    `)
    if err == nil {
        t.Error("Expected error for invalid score value")
    }
}
```

## Troubleshooting

### Common Issues

**"relation does not exist"**
- Run `CreateSchema()` before other database operations
- Check database connection and permissions

**"duplicate key value violates unique constraint"**
- Check for duplicate usernames in same poll
- Verify ballot upsert logic for voter tokens

**"foreign key constraint violation"**
- Ensure referenced records exist before insertion
- Check cascade delete behavior

### Debug Queries

```sql
-- Check poll states
SELECT status, count(*) FROM poll GROUP BY status;

-- Find polls without options
SELECT p.id, p.title FROM poll p 
LEFT JOIN option o ON p.id = o.poll_id 
WHERE o.id IS NULL;

-- Check ballot/score consistency
SELECT b.id, count(s.option_id) as score_count 
FROM ballot b 
LEFT JOIN score s ON b.id = s.ballot_id 
GROUP BY b.id 
HAVING count(s.option_id) = 0;
```

## Dependencies

- `database/sql` - Standard database interface
- `github.com/lib/pq` - PostgreSQL driver
- PostgreSQL 12+ - Database server

## Best Practices

1. **Always use transactions** for multi-table operations
2. **Handle foreign key constraints** gracefully
3. **Use prepared statements** for repeated queries
4. **Monitor query performance** with EXPLAIN ANALYZE
5. **Backup database** before schema changes