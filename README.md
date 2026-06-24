# Inventory Take-home

Inventory movement ingestion and query service built with Go, PostgreSQL, Fiber, GORM, React, and TypeScript.

The original exercise prompt is preserved in [ASSIGNMENT.md](ASSIGNMENT.md). Design notes and trade-offs are documented in [DECISIONS.md](DECISIONS.md).

## Project Structure

```text
backend/   Go API, ingestion command, migrations, and data generator
web/       React + TypeScript frontend
data/      product catalog and NDJSON event files
```

## Backend Architecture

The backend uses a lightweight layered structure:

```text
backend/internal/domain          Inventory types and validation rules
backend/internal/application     Ingestion and query use cases
backend/internal/infrastructure  PostgreSQL, file readers, migrations, and Fiber handlers
```

The application layer depends on small interfaces. Infrastructure packages implement those interfaces with PostgreSQL, NDJSON/CSV readers, and Fiber HTTP handlers.

## Backend

Start PostgreSQL:

```bash
docker compose up -d
```

Run ingestion:

```bash
cd backend
go run ./cmd/ingest
```

Expected sample output:

```text
products loaded: 8
files processed: 4
events inserted: 2000
duplicates skipped: 188
invalid lines: 81
```

Run the API:

```bash
cd backend
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default.

Available endpoints:

```text
GET /healthz
GET /products/stock
GET /products/:sku/movements
```

Run backend verification:

```bash
cd backend
go test ./...
```

## Frontend

```bash
cd web
npm install
npm run dev
```

Open `http://localhost:5173`.

To point the frontend to a different API URL:

```bash
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

Run frontend verification:

```bash
cd web
npm run build
```

## Regenerate Data

```bash
cd backend
go run ./tools/gen
go run ./tools/gen -n 2000000 -files 20
```

The generator writes to `../data` by default from the `backend/` directory.
