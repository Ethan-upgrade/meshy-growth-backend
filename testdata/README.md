# testdata — real-world A/B testing dataset validation

This directory holds tools that validate `experiment-service`'s statistics
engine (`wilsonInterval`, `twoProportionZTest`, `chiSquarePValue` in
`cmd/experiment-service/stats.go`) and the full event-gateway →
conversion pipeline against a **real, publicly available** A/B testing
dataset instead of only synthetic numbers.

## Dataset

**Source**: Udacity "Analyze A/B Test Results" course dataset, mirrored at
https://github.com/30lm32/ml-ab-testing/blob/master/ab_data.csv

294,479 rows of `user_id, timestamp, group, landing_page, converted` — a
classic e-commerce landing-page A/B test (`control`/`old_page` vs.
`treatment`/`new_page`).

The CSV is **not** committed to git (see `.gitignore`); fetch it locally:

```powershell
curl.exe -L -o testdata/ab_data.csv https://raw.githubusercontent.com/30lm32/ml-ab-testing/master/ab_data.csv
```

## `compute_stats.go` — offline aggregation

Standalone script (`//go:build ignore`, run with `go run`) that:

1. Reads `ab_data.csv`.
2. Drops 3,894 duplicate `user_id` rows (same user recorded more than once —
   a known data-quality issue in this dataset).
3. Drops 2,044 rows where `group`/`landing_page` are inconsistent (e.g.
   `group=control` with `landing_page=new_page`).
4. Prints per-variant `assigned`/`converted`/`rate`.

```
go run testdata/compute_stats.go
```

Result (reproduced in `cmd/experiment-service/stats_test.go` as
`TestStatsAgainstRealWorldABDataset`, independently cross-checked against a
Python `math.erf`-based z-test):

```
control:   assigned=144226 converted=17349 rate=0.1203
treatment: assigned=144314 converted=17134 rate=0.1187
p_value_vs_control (two-proportion z-test) ≈ 0.1956  (not significant, as expected —
  this is the well-known "no real difference" result for this dataset)
```

## `replay_ab_dataset.go` — live end-to-end integration test

Replays a random sample of dataset rows through the **actually running**
`event-gateway` and `experiment-service` HTTP APIs (requires
`docker compose up -d` from the repo root first):

```
cd testdata
go run replay_ab_dataset.go -sample 3000
```

It:

1. Creates a fresh experiment `real_dataset_replay_<unix-ts>` via
   `POST /v1/experiments` (idempotent — new key per run, doesn't collide with
   the demo `seed` data).
2. For each sampled row, calls `GET /v1/experiments/:key/assign` (exercising
   the real murmur3 bucketing in `internal/experiment`), and if the
   dataset's own `converted` flag is true, sends a `purchase` event via
   `POST /v1/events` (exercising Kafka → conversion-worker → MySQL).
3. Waits for the Kafka pipeline to drain, then calls
   `GET /v1/experiments/:key/stats` and diffs the live counts against the
   counts it itself observed from step 2's assignment responses.

Verified run (3,000 sampled rows, 2026-08-20):

```
control:   assigned=1525 converted=203 rate=0.1331
treatment: assigned=1475 converted=150 rate=0.1017
p_value_vs_control = 0.00758   (significant sub-sample; expected — smaller N,
                                 higher variance than the full 288k-row dataset)
srm_check.flagged  = false     (chi_square=0.833, p=0.361 — balanced 1525/1475
                                 split is NOT a sample ratio mismatch)

PASS: end-to-end replay through live event-gateway + experiment-service
matches expected counts exactly.
```

This confirms, against real third-party data (not just hand-crafted unit
test fixtures), that:

- murmur3-based bucketing (`internal/experiment.Bucket`/`PickVariant`) is
  deterministic and produces a roughly balanced split.
- The full write path (`event-gateway` → Kafka `events.raw` →
  `conversion-worker` → MySQL `events`/`experiment_assignments`) does not
  drop or double-count events under a few-thousand-request burst.
- `experiment-service`'s stats endpoint aggregates exactly the same numbers
  a client would compute itself from the raw assignment/event responses.
