# Balanced Majority Judgment (BMJ) Algorithm

Quickly Pick uses **Balanced Majority Judgment (BMJ)** to aggregate poll results. BMJ is designed to respect the intensity of preferences while avoiding options that many people dislike.

## Overview

BMJ converts slider positions to signed scores, computes statistical aggregates per option, applies a "soft veto" rule, and ranks options using lexicographic ordering.

## Slider to Score Conversion

The voting interface presents a slider with semantic labels:

```
Hate ←──────── Meh ──────────→ Love
0.0           0.5            1.0
```

### Conversion Formula

Raw slider values (`value01` ∈ [0, 1]) are converted to signed scores:

```
s = 2 × value01 - 1
```

This maps the slider to a signed range:
- `value01 = 0.0` → `s = -1.0` (Hate)
- `value01 = 0.5` → `s = 0.0` (Meh/Neutral)
- `value01 = 1.0` → `s = +1.0` (Love)

## Statistical Aggregates

For each option, BMJ computes five statistics from all signed scores `S = {s₁, s₂, ..., sₙ}`:

### 1. Median (`med`)
The 50th percentile - the middle value when scores are sorted.
```
med = percentile(S, 0.5)
```

### 2. 10th Percentile (`p10`)
The "least-misery" measure - represents the bottom 10% of scores.
```
p10 = percentile(S, 0.1)
```

### 3. 90th Percentile (`p90`)
The "upside" measure - represents the top 10% of scores.
```
p90 = percentile(S, 0.9)
```

### 4. Mean (`mean`)
The arithmetic average of all scores.
```
mean = (s₁ + s₂ + ... + sₙ) / n
```

### 5. Negative Share (`neg_share`)
The fraction of voters who gave negative scores (dislike).
```
neg_share = count(s < 0) / n
```

## Soft Veto Rule

Options can be "soft vetoed" if they have both:
1. **High negativity**: `neg_share ≥ 0.33` (≥33% negative votes)
2. **Non-positive median**: `median ≤ 0` (median is neutral or negative)

```
veto = (neg_share ≥ 0.33) AND (median ≤ 0)
```

Vetoed options are ranked below all non-vetoed options, regardless of other statistics.

## Ranking Algorithm

Options are ranked using **lexicographic ordering** on these criteria:

1. **Veto status** (ascending): Non-vetoed options beat vetoed ones
2. **Median** (descending): Higher median wins
3. **10th percentile** (descending): Higher p10 wins (least-misery)
4. **90th percentile** (descending): Higher p90 wins (upside)
5. **Mean** (descending): Higher mean wins
6. **Option ID** (ascending): Stable tie-breaking

## Example Calculation

Consider a poll with 3 options and 5 voters:

### Raw Votes (value01)
| Voter | Option A | Option B | Option C |
|-------|----------|----------|----------|
| 1     | 0.9      | 0.1      | 0.6      |
| 2     | 0.8      | 0.2      | 0.4      |
| 3     | 0.7      | 0.8      | 0.5      |
| 4     | 0.2      | 0.1      | 0.7      |
| 5     | 0.6      | 0.3      | 0.3      |

### Signed Scores (s = 2×value01 - 1)
| Voter | Option A | Option B | Option C |
|-------|----------|----------|----------|
| 1     | 0.8      | -0.8     | 0.2      |
| 2     | 0.6      | -0.6     | -0.2     |
| 3     | 0.4      | 0.6      | 0.0      |
| 4     | -0.6     | -0.8     | 0.4      |
| 5     | 0.2      | -0.4     | -0.4     |

### Aggregates
| Statistic | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| median    | 0.4      | -0.6     | 0.0      |
| p10       | -0.6     | -0.8     | -0.4     |
| p90       | 0.8      | 0.6      | 0.4      |
| mean      | 0.28     | -0.4     | 0.0      |
| neg_share | 0.2      | 0.8      | 0.4      |
| veto      | false    | true     | false    |

### Final Ranking
1. **Option A** (median=0.4, no veto)
2. **Option C** (median=0.0, no veto) 
3. **Option B** (vetoed due to neg_share=0.8 ≥ 0.33 AND median=-0.6 ≤ 0)

## Implementation in SQL

The BMJ calculation is implemented as a single PostgreSQL query:

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
  veto_rank ASC,     -- Non-vetoed first
  median DESC,       -- Higher median wins
  p10 DESC,          -- Least-misery tiebreaker
  p90 DESC,          -- Upside tiebreaker
  mean DESC,         -- Mean tiebreaker
  option_id ASC;     -- Stable final tiebreaker
```

## Why BMJ?

BMJ addresses common issues with other voting methods:

### vs. Simple Average
- **Problem**: Extreme scores can skew results
- **BMJ Solution**: Median-based ranking is more robust

### vs. Majority Vote
- **Problem**: Ignores intensity of preferences
- **BMJ Solution**: Uses full spectrum of slider values

### vs. Approval Voting
- **Problem**: Binary choice loses nuance
- **BMJ Solution**: Continuous scale captures preference strength

### vs. Ranked Choice
- **Problem**: Complex for voters, doesn't show intensity
- **BMJ Solution**: Simple slider interface with intensity

## Theoretical Properties

BMJ has several desirable properties:

1. **Monotonicity**: Improving an option's scores never hurts its ranking
2. **Clone Independence**: Adding similar options doesn't change relative rankings
3. **Majority Criterion**: If >50% prefer A over B, A ranks higher (when no veto)
4. **Condorcet Loser**: Options disliked by majority are often vetoed
5. **Strategy Resistance**: Honest voting is generally optimal

## Edge Cases

### All Options Vetoed
If all options are vetoed, ranking proceeds normally using median, p10, etc.

### Tied Statistics
The lexicographic ordering ensures deterministic results. Option ID provides final tie-breaking.

### Single Voter
BMJ works with any number of voters ≥1. All percentiles equal the single score.

### No Votes
Options with no votes receive default statistics (implementation-dependent).