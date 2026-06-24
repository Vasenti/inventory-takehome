# Decisions

## Delivery strategy

I treated the 4-hour limit as part of the problem. I prioritized a correct and explainable backend first, then added the smallest frontend that proves the API contract works.

The implemented order was:

1. Design the PostgreSQL schema around correctness and query patterns.
2. Implement migrations so the backend can bootstrap the database.
3. Implement ingestion with idempotency, validation, bounded database connections, and graceful cancellation.
4. Expose the stock and movement-history API with Fiber.
5. Add a minimal React + TypeScript UI after the backend path was working.

## Backend design

The backend lives under `backend/` as its own Go module. The frontend lives under `web/`, and the sample data remains at the repository root under `data/`.

I use a lightweight layered structure with `application`, `domain`, and `infrastructure` packages:

- `domain` contains inventory concepts and validation rules that do not depend on files, Fiber, GORM, or PostgreSQL.
- `application` contains use cases such as ingestion and inventory queries. It depends on small interfaces instead of concrete infrastructure.
- `infrastructure` contains adapters for PostgreSQL, file readers, migrations, and Fiber HTTP handlers.

This is intentionally smaller than a full enterprise Clean Architecture setup, but it gives the code clear boundaries and keeps the most important SOLID principle in place: application code depends on abstractions, while infrastructure implements the details.

The ingestion process reads all event files under `data/events/` concurrently, while the GORM-backed PostgreSQL connection pool is capped at 10 open connections to respect the exercise constraint.

Each valid movement is stored once using `event_id` as the idempotency key. Duplicate deliveries do not change stock more than once, whether they appear in the same file or across different files.

Stock is materialized per product instead of recalculated from the full movement table on every request. That makes the current-stock endpoint fast and predictable as the movement table grows to millions of rows.

The product movement history is indexed by product and movement time because it is expected to be queried constantly.

The API uses Fiber and exposes:

- `GET /products/stock`
- `GET /products/:sku/movements`

## Data validation

Invalid lines are skipped and recorded without aborting the whole ingestion run. That includes malformed JSON, unknown SKUs, invalid movement types, non-positive quantities, and invalid timestamps.

Validation errors are operationally useful, but they should not block valid inventory events from being processed.

## Use cases and test scenarios

The main use cases are ingestion, current stock queries, and movement history queries. Unit tests should cover behavior at the domain and application layers first because those layers express the business rules without requiring PostgreSQL, Fiber, or real files.

Domain movement validation scenarios:

- accepts a valid `IN` movement for a known SKU;
- accepts a valid `OUT` movement for a known SKU;
- rejects missing `event_id`;
- rejects unknown SKU;
- rejects unknown movement type;
- rejects zero or negative quantity;
- rejects invalid `occurred_at`;
- parses valid RFC3339 timestamps into `time.Time`.

Ingestion application scenarios:

- loads products before processing events;
- upserts each product and initializes stock through the repository port;
- processes multiple event files as one ingest run;
- records malformed JSON lines as ingest errors and continues;
- records domain validation errors and continues;
- stores valid movements through the repository port;
- counts inserted movements, duplicate deliveries, invalid lines, loaded products, and processed files;
- treats `StoreMovement(...)=false` as a duplicate and does not count it as inserted;
- returns partial summary when an operational error interrupts processing;
- cancels sibling file workers after the first operational error;
- propagates product reader, event listing, event reading, product upsert, movement store, and error-recording failures.

Inventory query scenarios:

- returns current stock from the repository port;
- returns product movement history when the SKU exists;
- returns `exists=false` without querying movements when the SKU does not exist;
- propagates repository errors when checking product existence;
- propagates repository errors when listing movements.

Infrastructure scenarios that are worth integration tests or focused adapter tests:

- PostgreSQL movement storage inserts a new event and updates materialized stock in the same transaction;
- duplicate `event_id` does not update stock again;
- invalid lines are persisted in `ingest_errors`;
- product history query remains ordered by `occurred_at DESC, event_id`;
- migrations are idempotent through `schema_migrations`;
- file readers preserve source file, line number, raw line, and parse errors.

## Trade-offs

I optimized for correctness, observability, and simple operational behavior over maximum throughput.

A batch-oriented ingestion path could be faster for very large datasets, but this version stays easy to reason about: validate, deduplicate, insert the movement, and update materialized stock in one database transaction.

I used explicit SQL for the critical ingestion and query paths even though the project uses GORM for the database connection. That keeps idempotency and stock updates easy to inspect.

The frontend is intentionally small. The exercise says visual polish is not evaluated, so frontend work proves the API integration rather than consuming time better spent on ingestion correctness.

## AI usage

I used AI as a pair-programming assistant to review the exercise, shape the implementation plan, implement focused slices, and challenge trade-offs. I kept commit messages and final design decisions explicit instead of relying on generated summaries.

I accepted help for structuring the work and identifying risks such as idempotency, bounded database concurrency, cancellation, and indexes for history queries.

I would reject AI output that hides important decisions behind generic abstractions, ignores the pool-size constraint, treats duplicate events as harmless inserts, or skips documenting what was left out.

## Known limits

The current solution covers migrations, ingestion, API, and a minimal frontend.

Nice-to-have items left out of scope are advanced ingestion metrics, pagination for very large movement histories, containerized backend/frontend services, authentication, richer API error types, and elaborate UI styling.
