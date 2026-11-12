# Golingko - One Platform, Multiple Transit Networks

Golingko is a high-throughput backend engine that simulates Greater Jakarta’s public transit network.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Turnstile SLA](https://img.shields.io/badge/Turnstile_SLA-Sub--300ms_PASSED-success?style=flat-square)](docs/RESULT_2026-11-09.md)
[![Peak Throughput](https://img.shields.io/badge/Peak_Throughput-~119k_req%2Fs-blueviolet?style=flat-square)](README.md#benchmark-highlights)
[![API Docs](https://img.shields.io/badge/OpenAPI_3.0-Scalar_UI-orange?style=flat-square)](api/openapi/)
[![Database](https://img.shields.io/badge/PostgreSQL-17-336791?style=flat-square&logo=postgresql)](migrations/)
[![License](https://img.shields.io/badge/License-MIT-lightgrey?style=flat-square)](LICENSE)

### Preface

> "How does a transit card know the exact fare to deduct the moment it taps a turnstile?"

Every day across Greater Jakarta, commuters tap contactless transit cards at station gates without thinking twice. Under the hood, the backend must resolve multi-operator transfer discounts, prevent concurrent double taps (race conditions), and keep ledger balances accurate under heavy rush-hour traffic. Golingko was built to explore, benchmark, and solve these backend challenges from scratch.

### Architectural Decision: BRT Gate Model

TransJakarta BRT operates primarily through enclosed platform stations with turnstiles. Golingko standardizes fare collection around this directional station gate model (`POST /api/gate/tap-in` and `POST /api/gate/tap-out`), mirroring the physical infrastructure of MRT Jakarta, LRT Jakarta, TransJakarta BRT halte, and KRL Commuter Line.

While feeder buses (Mikrotrans) use onboard validators (Tap-On-Bus), the underlying journey tracking, transfer rules, and balance deductions remain identical. Modeling dedicated gates provides clean entry and exit boundaries, prevents duplicate taps (anti-passback protection), and ensures an auditable transaction history.

---

### Benchmark Highlights

Golingko is evaluated against a physical **Sub-300ms Turnstile SLA** to measure throughput, tail latency ($p99$), and saturation limits under commuter load. Benchmarks simulate real Jakarta traffic ratios (90% tap-in/tap-out, 8% balance top-up, 2% card issuance) with in-memory partitioned row-level locking.

- **Line 1 (`net/http` + `zerolog`)**: Top sustained throughput stack, maintaining **106,032 to 119,435 req/s** through 10,000 concurrent workers.
- **Line 2 (`net/http` + `zap`)**: Zero-allocation optimized runner-up, sustaining **111,139 to 119,255 req/s** via `sync.Pool` field slice reuse.
- **Line 3 (`net/http` + `log/slog`)**: Standard library stack, delivering **108,016 to 111,579 req/s** with zero third-party dependencies.
- **Line 4 (Research Baseline - Red Line)**: Real-world baseline of **30,000 req/s** based on Jakarta commuter flow studies at Manggarai Station. All three stacks operate **3.6× to 4.0× above** peak network demand.

```mermaid
xychart-beta
    title "Throughput (req/s): 3 Best Framework Stacks vs Turnstile Baseline"
    x-axis ["500 Workers", "2.5K Rush", "10K Workers", "13K Workers", "20K Workers", "30K Workers"]
    y-axis "Requests per Second (req/s)" 0 --> 140000
    line [117984, 119435, 106032, 106694, 103346, 69013]
    line [119255, 116349, 111139, 109057, 96126, 69873]
    line [111579, 107719, 108016, 110389, 96565, 67872]
    line [30000, 30000, 30000, 30000, 30000, 30000]
```

#### I. Concurrency Saturation & Turnstile SLA Breaking Curve

Evaluates the top three production stacks against the physical **300ms Turnstile SLA** across six operational tiers:

| Concurrency (Workers) | `net/http` + `zerolog` (req/s \| p99) | `net/http` + `zap` (req/s \| p99) | `net/http` + `log/slog` (req/s \| p99) | Turnstile SLA Status | Failure % (SLA Breach) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **500** | **117,984** (18.6 ms) | **119,255** (19.0 ms) | 111,579 (20.5 ms) | **PASSED** | **0.00%** |
| **2,500** *(Rush Hour)* | **119,435** (65.5 ms) | 116,349 (67.2 ms) | 107,719 (73.0 ms) | **PASSED** | **0.00%** |
| **10,000** | 106,032 (297.7 ms) | **111,139** (281.7 ms) | 108,016 (287.2 ms) | **PASSED** | **0.4% – 0.9%** |
| **13,000** *(SLA Cliff)* | 106,694 (385.3 ms) | 109,057 (373.2 ms) | **110,389** (356.4 ms) | **FAILED** *(SLA Cliff)* | **3.0% – 4.3%** |
| **20,000** | **103,346** (580.7 ms) | 96,126 (656.2 ms) | 96,565 (642.4 ms) | **FAILED** | **12.5% – 21.1%** |
| **30,000** *(Saturation Cliff)* | 69,013 (1,091 ms) | **69,873** (992.0 ms) | 67,872 (1,025 ms) | **FAILED** | **59.1% – 63.3%** |

#### II. Router Framework Shootout (`net/http` vs `chi` vs `gin`)

All three router engines were evaluated under identical business logic, route definitions, and partitioned repository row-level locking.

##### A. Router Concurrency Shootout (Zero-Overhead Baseline vs Production Logging)

Compares throughput and tail latency with logging disabled versus standard production JSON logging:

| Router Engine | Logging Mode | 500 Workers (req/s \| p99) | 2,500 Rush Hour (req/s \| p99) | 10,000 Saturation (req/s \| p99) | Turnstile SLA Status | SLA Breach Rate (10k) |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: |
| **`net/http`** *(Go Stdlib)* | **`noop`** *(Zero Overhead)* | **155,433** (15.7 ms) | **156,245** (51.4 ms) | **147,234** (210.1 ms) | **PASSED** | **0.12%** |
| | **`slog`** *(JSON Stdlib)* | **119,482** (20.6 ms) | **125,888** (61.5 ms) | **124,450** (246.5 ms) | **PASSED** | **0.21%** |
| **`chi`** *(v5)* | **`noop`** *(Zero Overhead)* | 131,805 (16.4 ms) | 130,791 (59.9 ms) | 133,249 (233.8 ms) | **PASSED** | 0.15% |
| | **`slog`** *(JSON Stdlib)* | 112,312 (21.2 ms) | 119,815 (68.5 ms) | 120,581 (253.7 ms) | **PASSED** | 0.19% |
| **`gin`** *(Release)* | **`noop`** *(Zero Overhead)* | 129,263 (19.9 ms) | 127,597 (60.3 ms) | 122,033 (257.6 ms) | **PASSED** | 0.16% |
| | **`slog`** *(JSON Stdlib)* | 119,827 (19.3 ms) | 118,608 (65.0 ms) | 120,145 (255.1 ms) | **PASSED** | 0.16% |

##### B. Router Micro-Benchmark Allocations (`go test -bench=BenchmarkRouter_`)

Measures per-operation latency and heap allocation overhead across router dispatch:

| Route Operation | `net/http` (Stdlib) | `chi` (v5) | `gin` (Release) | Key Architectural Observation |
| :--- | :---: | :---: | :---: | :--- |
| **Station List** (`GET /api/stations`) | **34.1 µs** (7,486 B, 20 allocs) | 37.3 µs (7,864 B, 22 allocs) | **32.3 µs** (7,493 B, 20 allocs) | `net/http` matches Gin; Chi incurs 2 extra allocations |

#### III. Structured Logging Framework Shootout (Mounted on `net/http`)

Evaluates the exact observability overhead (throughput penalty, tail latency inflation, and GC pressure) across drivers:

| Logging Driver | 500 Workers (req/s \| p99) | 2,500 Rush Hour (req/s \| p99) | 10,000 Saturation (req/s \| p99) | Turnstile SLA Status | SLA Breach Rate (10k) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **`noop`** *(Zero Overhead Baseline)* | 118,785 (18.0 ms) | 121,693 (63.1 ms) | 113,448 (275.5 ms) | Reference Baseline | 0.56% |
| **`zerolog`** *(JSON)* | **113,929** (19.4 ms) | **120,897** (64.9 ms) | **121,404** (256.8 ms) | **PASSED** | **0.14%** |
| **`slog`** *(JSON Stdlib)* | 114,362 (21.9 ms) | 115,270 (68.7 ms) | 114,350 (269.7 ms) | **PASSED** | 0.20% |
| **`zap`** *(JSON - Pooled Zero-Alloc)* | 110,218 (19.3 ms) | 110,886 (71.4 ms) | 107,860 (291.6 ms) | **PASSED** | 0.77% |

##### A. Driver Allocation Micro-Benchmarks (`go test -bench=Benchmark -benchmem ./pkg/logger`)

Measures isolated driver latency and memory allocations per log call:

| Logging Driver | Operation Latency | Heap Allocation | Allocations / Op | Efficiency Profile |
| :--- | :---: | :---: | :---: | :--- |
| **`noop`** | 14.9 ns/op | **0 B/op** | **0 allocs/op** | Pure CPU no-op execution |
| **`slog`** *(JSON Stdlib)* | **637 ns/op** | **0 B/op** | **0 allocs/op** | Zero third-party dependency standard library |
| **`zerolog`** *(JSON)* | 1,248 ns/op | **0 B/op** | **0 allocs/op** | In-place byte buffer serialization |
| **`zap`** *(JSON Pooled)* | 1,627 ns/op | **0 B/op** | **0 allocs/op** | **Optimized via `sync.Pool` slice reuse** |

#### IV. Direct Architectural Conclusions & Code Facts

##### A. Why `net/http` is Faster than `chi` and `gin`
Go 1.22 introduced enhanced routing pattern matching directly into `http.ServeMux` ([`internal/api/router_nethttp.go`](file:///f:/__Projects/golingko/internal/api/router_nethttp.go)). Because route patterns and wildcard paths like `/api/cards/{number}` resolve natively without intermediate context allocations, `net/http` operates with zero framework overhead:
- In the router shootout, `net/http` delivers up to **156,245 req/s** (+19.5% over Chi, +22.4% over Gin) with lowest tail latency ($p99 = 15.72\text{ ms}$).
- **`chi`** relies on `chi.RouteContext` ([`internal/api/router_chi.go`](file:///f:/__Projects/golingko/internal/api/router_chi.go)), which dynamically allocates URL parameter structures on the heap, adding 2 extra allocations and 378 bytes per request.
- **`gin`** requires adapting `gin.Context` into standard `http.HandlerFunc` via [`adaptGin`](file:///f:/__Projects/golingko/internal/api/router_gin.go), introducing wrapper function indirection that reduces throughput under high concurrency.

##### B. Why `zap` Was Slower and How We Made It Faster (The `sync.Pool` Zero-Alloc Optimization)
Uber Zap is known for zero-allocation performance when used natively with typed fields (`zap.String()`, `zap.Int()`). However, Golingko standardizes all loggers under Go's universal `*slog.Logger` interface. This created an initial performance bottleneck:
- **The Initial Bridge Overhead**: Zap was initially mounted via `go.uber.org/zap/exp/zapslog`. On every logged request, `zapslog` converted `slog.Attr` into a dynamically allocated slice of `zapcore.Field` on the heap (`make([]zapcore.Field, 0, r.NumAttrs())`). This caused temporary garbage and GC pressure that limited throughput.
- **The Zero-Allocation Fix**: We replaced `zapslog` with a custom `zapHandler` ([`pkg/logger/zap.go`](file:///f:/__Projects/golingko/pkg/logger/zap.go)) backed by a `sync.Pool` of pre-allocated `[]zapcore.Field` slices (`zapFieldsPool`). Each request borrows a slice from the pool, populates it without heap allocation, writes to Zap core, resets the slice, and returns it to the pool.
- **The Performance Result**:
  - Heap allocations dropped from **1 alloc/op (320 B/op)** to **0 allocs/op (0 B/op)**.
  - Concurrency throughput reached **119,255 req/s**, maintaining full parity with Zerolog and standard library `slog`.
- **The Zerolog Baseline**: `zerolog` uses a custom adapter ([`pkg/logger/zerolog.go`](file:///f:/__Projects/golingko/pkg/logger/zerolog.go)) writing directly into a reusable byte buffer with **0 B/op and 0 allocations**, sustaining up to 121,404 req/s at 10,000 workers.
- **The Standard `slog` Baseline**: Go's built-in `log/slog` also writes JSON directly to internal reusable buffers with **0 B/op and 0 allocations**, maintaining sub-millisecond execution and lowest latency (637 ns/op) without third-party dependencies.

---

### Key Architectural Capabilities

- **Multi-Modal Transit Integration**: Combines MRT Jakarta, LRT Jakarta, TransJakarta BRT, and Mikrotrans routes under a single ticketing lifecycle.
- **180-Minute Integrated Fare Policy**: Applies transfer discounts across multiple transit operators within a rolling 3-hour window, capped at Rp 10,000.
- **Accidental Tap Grace Period**: Free immediate exit within 3 minutes if tapped by mistake; assesses a standard Rp 3,000 platform turnaround fee after 15 minutes.
- **Directional Gate Protocol**: Validates directional flow (enter before exit), enforces minimum card balance (Rp 5,000), and auto-settles abandoned journeys after 4 hours.
- **Immutable Double-Entry Ledger**: Tracks all top-ups and trip fare deductions with row-level locks, ensuring balanced audit trails without balance drift.
- **Distributed Mutation Idempotency**: Caches mutation responses for 5 minutes via the `Idempotency-Key` header to prevent duplicate charges from scanner retries.
- **Pluggable Router Architectures**: Supports hot-swapping between Go stdlib `net/http`, Chi, and Gin without modifying business logic.
- **Pluggable Structured Logging**: Standardized on `log/slog` with swappable drivers for standard Slog, Uber Zap, Zerolog, or a zero-overhead Noop driver.

---

```
golingko/
├── cmd/
│   ├── golingko/                     # Application entrypoint & dependency wiring
│   ├── seed/                         # Transit network and test data seeder
│   └── stress/                       # High-concurrency transit stress CLI
├── internal/
│   ├── api/                          # HTTP layer, handlers, router engines, and server lifecycle
│   │   ├── handler.go                # REST request handlers
│   │   ├── router.go                 # Common router engine registry & interface
│   │   ├── router_nethttp.go         # Go stdlib net/http router implementation
│   │   ├── router_chi.go             # Chi v5 router implementation
│   │   ├── router_gin.go             # Gin router implementation
│   │   └── server.go                 # HTTP server lifecycle & graceful shutdown
│   ├── transit/                      # Core transit business domain
│   │   ├── card.go                   # Card management & provisioning service
│   │   ├── fare.go                   # Integrated tariff & calculation rules
│   │   ├── journey.go                # Turnstile tap-in/tap-out journey orchestrator
│   │   ├── ledger.go                 # Financial transaction ledger & audit models
│   │   ├── repository.go             # Transit PostgreSQL data access contracts
│   │   ├── route.go                  # Transit route models & connectivity checks
│   │   ├── slug.go                   # Transit slug normalization utilities
│   │   └── station.go                # Multi-modal station directory & queries
│   ├── config/
│   └── worker/
├── test/
│   ├── api_bench_test.go             # API endpoint benchmarks
│   ├── concurrency_test.go           # Concurrency & race condition test suite
│   ├── mock_repo.go                  # Thread-safe in-memory partitioned repository
│   └── router_bench_test.go          # Pluggable router comparison benchmarks
├── pkg/
├── api/
├── migrations/
├── seeds/
├── scripts/
├── docker-compose.yml
└── Makefile
```

---

### Technology Stack

| Category | Technology | Details |
|---|---|---|
| **Backend Runtime** | [Go](https://go.dev/) | 1.25+ |
| **Database** | [PostgreSQL](https://www.postgresql.org/) | 16+ with connection pooling and transaction isolation |
| **HTTP Routing** | `net/http`, [Chi](https://github.com/go-chi/chi), [Gin](https://github.com/gin-gonic/gin) | Pluggable routing adapters |
| **Logging Drivers** | `log/slog`, [Uber Zap](https://github.com/uber-go/zap), [Zerolog](https://github.com/rs/zerolog) | Pluggable logging backends |
| **API Documentation** | [OpenAPI](https://www.openapis.org/) | Version 3.1.0, rendered via [Scalar](https://scalar.com/) |
| **API Automated Testing** | [Bruno](https://www.usebruno.com/), [Postman](https://www.postman.com/) | API testing and automation via Newman |
| **Infrastructure** | [Docker](https://www.docker.com/), Docker Compose | Containerization and service orchestration |

---

### Quick Start

Launch Golingko using either `make` shortcuts or standard Go and Docker commands.

#### Running with Make

```bash
# 1. Start the PostgreSQL container
make up

# 2. Run migrations and load transit seeds
make migrate-up
make seed

# 3. Start the application server (binds to http://localhost:8080)
make dev
```

#### Running with Traditional Tooling

```bash
# 1. Start the PostgreSQL container
docker compose up -d

# 2. Run migrations and load transit seeds
.\scripts\migrate.ps1 up
.\scripts\seed.ps1 up

# 3. Start the application server (binds to http://localhost:8080)
go run ./cmd/golingko
```

---

### Quick Actions

Development, database, testing, and benchmark commands available via `make`:

| Command | Traditional Equivalent | Description |
| :--- | :--- | :--- |
| `make up` | `docker compose up -d` | Boots up the PostgreSQL container in the background |
| `make down` | `docker compose down` | Stops and removes PostgreSQL containers |
| `make dev` | `go run ./cmd/golingko` | Starts the application server on `http://localhost:8080` |
| `make build` | `go build -o bin/golingko ./cmd/golingko` | Compiles the standalone application binary into `bin/` |
| `make clean` | `rm -rf bin/` | Removes compiled binaries and build artifacts |
| `make migrate-up` | `.\scripts\migrate.ps1 up` | Applies all pending database migrations |
| `make migrate-down` | `.\scripts\migrate.ps1 down 1` | Rolls back the latest applied migration |
| `make migrate-status` | `.\scripts\migrate.ps1 version` | Checks the current database migration version |
| `make migrate-create name=...` | `.\scripts\migrate.ps1 create <name>` | Scaffolds a new pair of SQL migration files |
| `make seed` | `.\scripts\seed.ps1 up` | Seeds realistic stations, routes, cards, journeys, and ledger |
| `make seed-down` | `.\scripts\seed.ps1 down` | Purges seeded test records from the database |
| `make seed-reset` | `.\scripts\seed.ps1 reset` | Resets and re-applies fresh seeds |
| `make test` | `go test -v -race ./...` | Runs unit tests with Go race condition detection |
| `make bench` | `go test -bench Benchmark -benchmem -run none ./test ./pkg/logger` | Runs memory allocation and CPU benchmark suites |
| `make bench-loggers` | `.\scripts\benchmark_loggers.ps1` | Runs structured logging driver micro-benchmarks |
| `make stress` | `go run ./cmd/stress -workers 200 -duration 5s` | Runs high-concurrency turnstile load simulation (200 workers) |
| `make stress-matrix` | `.\scripts\stress_matrix.ps1` | Runs multi-driver stress matrix & SLA breaking point profiler |
| `make test-api-bruno` | `npx @usebruno/cli run api/bruno --env Local` | Runs automated Bruno API test collection |
| `make test-api-postman` | `npx newman run api/postman/...` | Runs automated Postman test collection via Newman |

---

### Configuration Reference

Configure the platform via environment variables or a `.env` file:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `SERVER_ADDR` | `127.0.0.1:8080` | Server listen address (defaults to local loopback; accepts `PORT` alias) |
| `DATABASE_URL` | `postgres://golingko:golingko123@localhost:5432/golingko_db?sslmode=disable` | PostgreSQL connection string |
| `HTTP_ROUTER` | `nethttp` | HTTP router engine (`nethttp`, `chi`, `gin`) |
| `LOG_DRIVER` | `slog` | Structured logging driver (`slog`, `zap`, `zerolog`, `noop`) |
| `LOG_VERBOSITY` | `info` | Minimum log verbosity threshold (`debug`, `info`, `warn`, `error`; accepts alias `LOG_LEVEL`) |
| `MIN_TAP_BALANCE` | `5000` | Minimum balance in IDR required to tap in at turnstiles |
| `JOURNEY_EXPIRY_HOURS` | `4` | Maximum journey duration before sweeper penalty assessment |

---

### API Documentation & Tooling

- **Interactive API Documentation**: Explore and execute requests via Scalar UI at `http://localhost:8080/docs/`.
- **[API Testing & Tooling Guide](file:///f:/__Projects/golingko/api/README.md)**: Details on Bruno, Postman, and Scalar test workflows.

---

### License

This project is licensed under the [MIT License](LICENSE).
