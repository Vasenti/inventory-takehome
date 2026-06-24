# Decisions

## Delivery strategy

I treated the 4-hour limit as part of the problem. My priority is to deliver a correct and explainable backend first, then add the smallest frontend that proves the API contract works.

The intended order is:

1. Design the PostgreSQL schema around correctness and query patterns.
2. Implement ingestion with idempotency, validation, bounded concurrency, and graceful cancellation.
3. Expose the stock and movement-history API.
4. Add a minimal React + TypeScript UI only after the backend path is solid.

## Backend design

The ingestion process should read all event files under `data/events/` concurrently, but database writes must stay within the configured connection pool limit. I plan to keep the pool size at 10 and use worker limits so file concurrency does not become unbounded database concurrency.

Each valid movement is stored once using `event_id` as the idempotency key. Duplicate deliveries should not change stock more than once, whether they appear in the same file or across different files.

Stock should be materialized per product instead of recalculated from the full movement table on every request. That makes the current-stock endpoint fast and predictable as the movement table grows to millions of rows.

The product movement history should be indexed by product and movement time because it is expected to be queried constantly.

## Data validation

Invalid lines should be skipped and recorded without aborting the whole ingestion run. That includes malformed JSON, unknown SKUs, invalid movement types, non-positive quantities, and invalid timestamps.

Validation errors are operationally useful, but they should not block valid inventory events from being processed.

## Trade-offs

I am optimizing for correctness, observability, and simple operational behavior over maximum throughput.

A batch-oriented ingestion path could be faster for very large datasets, but the first version should stay easy to reason about: validate, deduplicate, insert the movement, and update materialized stock in one database transaction.

I will keep the frontend intentionally small. The exercise says visual polish is not evaluated, so frontend work should prove the API integration rather than consume time better spent on ingestion correctness.

## AI usage

I used AI as a pair-programming assistant to review the exercise, shape the implementation plan, and challenge trade-offs. I kept the commit messages and final design decisions explicit instead of relying on generated summaries.

I accepted help for structuring the work and identifying risks such as idempotency, bounded database concurrency, cancellation, and indexes for history queries.

I would reject AI output that hides important decisions behind generic abstractions, ignores the pool-size constraint, treats duplicate events as harmless inserts, or skips documenting what was left out.

## Known limits

If time gets tight, I will prefer a backend with migrations, ingestion, API, and clear documentation over a polished frontend.

Nice-to-have items that may stay out of scope are advanced ingestion metrics, pagination beyond the minimal history endpoint, containerized app services, authentication, and elaborate UI styling.
