# Tx Processor

`tx-processor` processes newline-delimited JSON transaction data, aggregates per-user analytics concurrently in Go, stores the results in PostgreSQL, and exposes an HTTP API for querying the processed analytics.

It is designed around worker pools, worker-local aggregation, bulk database writes, and optional Redis-backed read caching.

## Table of Contents

- Overview
- Approach
- Trade-offs
- Project Structure
- Prerequisites
- Running the Application
- API Endpoints
- Testing

## Overview

The processor reads an NDJSON file of transactions, fans the work out across multiple workers, aggregates analytics in memory, and flushes the final result to PostgreSQL in bulk.

The API layer exposes endpoints for:

- total orders per user
- total spending per user
- top users by order volume
- anomaly detection

Redis can be enabled to speed up repeated single-user reads.

## Approach

### Transaction Format

Transactions are stored as **NDJSON**: one JSON object per line.

This allows the processor to stream records incrementally instead of loading the entire file into memory. Compared with a JSON array, this keeps memory usage low and makes the approach suitable for large datasets.

### Processing Model

The processor uses a buffered channel and a pool of worker goroutines.

- One goroutine reads lines from the file.
- Lines are pushed onto a buffered channel.
- Multiple workers consume lines concurrently.
- Each worker aggregates into its own private in-memory map.
- After all workers finish, the worker-local maps are merged into one final snapshot.
- The merged result is flushed to PostgreSQL in bulk.

This avoids lock contention during the hot path because workers do not share analytics state while processing.

### Database Interaction

Analytics are stored in PostgreSQL using `sqlx`.

- migrations are managed with `goose`
- per-user analytics are inserted or updated using `ON CONFLICT`
- bulk upserts reduce database round-trips significantly compared with one write per transaction

### Caching

Redis is used as an optional cache for repeated single-user analytics lookups.

Cached reads are used for:

- `GET /total_orders?user_id={user_id}`
- `GET /total_spendings?user_id={user_id}`

After a successful bulk write, affected user cache entries are invalidated so repeated reads do not continue serving stale values.

## Trade-offs

- Worker-local aggregation reduces lock contention, but requires a merge step after processing.
- Bulk database writes improve throughput substantially, but failures affect larger chunks of work than one-row-at-a-time inserts.
- Redis improves repeated read latency, but adds operational complexity.
- The anomaly query is simple and effective for surfacing outliers, but it is still a heuristic rather than a full anomaly-detection system.

## Project Structure

```bash
tx-processor/
├── cmd/
│   └── tx-processor/
│       ├── cli/                    # File processor
│       └── main.go                 # API server
├── internal/
│   ├── cache/                      # Cache interface
│   │   └── redis/                  # Redis implementation
│   ├── config/                     # Environment config
│   ├── db/                         # Database setup
│   │   └── migrations/             # SQL migrations
│   ├── handler/                    # HTTP handlers
│   ├── middleware/                 # HTTP middleware
│   ├── models/                     # Shared types
│   ├── processor/                  # Processing engine
│   ├── repository/                 # Database queries
│   ├── server/                     # HTTP server
│   └── service/                    # Business logic
├── scripts/                        # Test data generator
└── docker-compose.yaml
```

## Prerequisites

Before you begin, make sure you have:

- Go `1.25.7+`
- Docker
- Docker Compose

## Running the Application

### Clone the repository

```bash
git clone https://github.com/iamkaroko/tx-processor.git
cd tx-processor
```

### Start infrastructure

```bash
docker compose up -d
```

This starts:

- PostgreSQL
- Redis

Optional tools:

```bash
docker compose --profile tools up -d
```

This also starts:

- pgAdmin at `http://localhost:5050`
- Redis Commander at `http://localhost:8081`

### Install dependencies

```bash
go mod tidy
```

### Generate test data

```bash
go run scripts/generate_test_data.go 1000000 sample_transactions.json
```

### Run the processor

```bash
go run cmd/tx-processor/cli/main.go -file=sample_transactions.json -workers=10 -batch=500
```

Flags:

- `-file` path to the NDJSON file
- `-workers` number of worker goroutines
- `-batch` batch size per worker

Example:

```bash
go run cmd/tx-processor/cli/main.go -file=sample_transactions.json -workers=5 -batch=250
```

### Run the API

```bash
go run cmd/tx-processor/main.go
```

### Enable Redis-backed reads

Redis-backed reads are enabled by setting:

```bash
REDIS_ENABLED=true go run cmd/tx-processor/main.go
```

## API Endpoints

### Total orders for a user

```bash
curl "http://localhost:8080/total_orders?user_id=user_1"
```

### Total spending for a user

```bash
curl "http://localhost:8080/total_spendings?user_id=user_1"
```

### Top users by order volume

```bash
curl "http://localhost:8080/top_users?limit=10"
```

### Users with anomalous activity

```bash
curl "http://localhost:8080/anomalies"
```

## Testing

Run the test suite with:

```bash
go test ./...
```
