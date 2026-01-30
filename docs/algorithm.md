# Balanced Majority Judgment (BMJ)

This document explains the BMJ algorithm used by Quickly Pick to aggregate votes and determine poll winners.

## Overview

Balanced Majority Judgment is a voting algorithm designed for group decision-making that:

1. **Respects intensity** - A slider from hate to love captures how strongly people feel
2. **Avoids divisive options** - Options that many people dislike get penalized
3. **Finds consensus** - Favors options with broad support over polarizing ones

## Why Traditional Voting Fails

**Plurality voting** (pick your favorite):
- Can elect options most people dislike
- Doesn't capture preference intensity
- Encourages strategic voting

**Average/sum scoring**:
- Extreme voters have outsized influence
- Susceptible to tactical voting

**BMJ solves these problems** by using medians (robust to outliers) combined with a soft veto for widely disliked options.

## How It Works

### Step 1: Score Conversion

Voters use a slider from 0 to 1:
- `0.0` = Strong dislike (hate)
- `0.5` = Neutral (meh)
- `1.0` = Strong like (love)

Scores are converted to signed values for statistics:

```
signed_score = 2 × value01 - 1
```

This maps the range:
- `0.0` → `-1.0` (hate)
- `0.5` → `0.0` (neutral)
- `1.0` → `+1.0` (love)

### Step 2: Compute Statistics

For each option, calculate:

| Statistic | Description |
|-----------|-------------|
| **Median** | 50th percentile of signed scores |
| **P10** | 10th percentile (worst-case sentiment) |
| **P90** | 90th percentile (best-case sentiment) |
| **Mean** | Arithmetic average |
| **NegShare** | Fraction of scores below 0 |

### Step 3: Apply Soft Veto Rule

An option is **vetoed** if:

```
NegShare ≥ 33% AND Median ≤ 0
```

This means: if at least a third of voters dislike an option AND the median sentiment is negative or neutral, the option is penalized.

Vetoed options are ranked below all non-vetoed options.

### Step 4: Lexicographic Ranking

Options are sorted by these criteria in order:

1. **Non-vetoed first** - Vetoed options always rank last
2. **Higher median** - Better overall sentiment wins
3. **Higher P10** - Better worst-case (least-misery tiebreaker)
4. **Higher P90** - Better best-case (upside tiebreaker)
5. **Higher mean** - Final numeric tiebreaker
6. **Option ID** - Alphabetical for stable sorting

## Example

### Scenario

Three voters evaluate three restaurant options:

| Voter | Sushi (signed) | Pizza (signed) | Burger (signed) |
|-------|----------------|----------------|-----------------|
| Alice | +0.6 | -0.4 | +0.2 |
| Bob | +0.8 | +0.6 | +0.4 |
| Carol | +0.2 | -0.6 | +0.0 |

### Calculations

**Sushi:**
- Sorted: [+0.2, +0.6, +0.8]
- Median: +0.6
- P10: +0.2
- P90: +0.8
- Mean: +0.53
- NegShare: 0% (no negative scores)
- Veto: No

**Pizza:**
- Sorted: [-0.6, -0.4, +0.6]
- Median: -0.4
- P10: -0.6
- P90: +0.6
- Mean: -0.13
- NegShare: 67% (2 of 3 negative)
- Veto: **Yes** (67% ≥ 33% AND -0.4 ≤ 0)

**Burger:**
- Sorted: [+0.0, +0.2, +0.4]
- Median: +0.2
- P10: +0.0
- P90: +0.4
- Mean: +0.2
- NegShare: 0%
- Veto: No

### Final Ranking

| Rank | Option | Median | Veto | Reason |
|------|--------|--------|------|--------|
| 1 | Sushi | +0.6 | No | Highest median among non-vetoed |
| 2 | Burger | +0.2 | No | Second highest median |
| 3 | Pizza | -0.4 | Yes | Vetoed - too many disliked it |

**Result:** Sushi wins! Even though Bob loved Pizza, too many people disliked it.

## Mathematical Formulas

### Percentile Calculation

For sorted array `S` of length `n`, the `p`-th percentile (where `p ∈ [0,1]`):

```
rank = p × (n - 1)
lower = floor(rank)
upper = lower + 1
weight = rank - lower

percentile = S[lower] × (1 - weight) + S[upper] × weight
```

This uses linear interpolation between adjacent values.

### Negative Share

```
NegShare = count(scores < 0) / total_scores
```

### Veto Condition

```
Veto = (NegShare ≥ 0.33) AND (Median ≤ 0)
```

## Why These Thresholds?

**33% for NegShare:**
- Lower than majority (50%) to catch polarizing options early
- High enough that a small minority can't veto everything
- Represents "significant opposition"

**Median ≤ 0 for veto:**
- An option with positive median has net positive sentiment
- Only penalize options that are neutral-to-negative overall
- Prevents vetoing genuinely popular options

## Comparison to Other Methods

| Method | Handles Intensity | Resists Manipulation | Avoids Divisive |
|--------|-------------------|---------------------|-----------------|
| Plurality | No | No | No |
| Approval | Partial | Partial | No |
| Borda Count | Partial | No | No |
| Mean Score | Yes | No | No |
| Median Score | Yes | Yes | Partial |
| **BMJ** | Yes | Yes | **Yes** |

## Implementation Notes

The BMJ implementation is in `server/handlers/bmj.go`:

- `ComputeBMJRankings()` - Main entry point
- `percentile()` - Calculates percentiles with interpolation
- `mean()` - Arithmetic average
- `negativeShare()` - Fraction of negative scores

Results are stored as a JSON snapshot when the poll closes, ensuring results are immutable and verifiable.
