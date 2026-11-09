# Golingko API Testing & Documentation Guide

This directory consolidates the full API testing and documentation suite for **Golingko - One Platform, Multiple Transit Networks**. Golingko provides complete coverage across three developer tooling formats:

1. **[OpenAPI 3.1.0 Specification](#1-openapi-specification)** (`api/openapi/`) - Contract definition, schema documentation, and interactive Scalar playground.
2. **[Postman Collection & Environment](#2-postman-test-suite)** (`api/postman/`) - Automated end-to-end integration test suites with Newman CLI support.
3. **[Bruno Collection](#3-bruno-git-friendly-test-suite)** (`api/bruno/`) - Open-source, Git-friendly plain-text API testing with Bruno CLI support.

---

## Directory Layout

```
api/
├── README.md                                          # This testing & documentation guide
├── openapi/
│   ├── openapi.yaml                                   # OpenAPI 3.1.0 Specification (YAML)
│   ├── openapi.json                                   # OpenAPI 3.1.0 Specification (JSON)
│   └── index.html                                     # Interactive Scalar API reference UI
├── postman/
│   ├── golingko.postman_collection.json               # Postman Collection v2.1.0
│   └── golingko_local.postman_environment.json        # Postman Local Environment
└── bruno/
    ├── bruno.json                                     # Bruno Collection manifest
    ├── environments/
    │   └── Local.bru                                  # Local environment variables
    ├── 00-health/
    │   └── Health Check.bru
    ├── 01-cards/
    │   ├── 01-Create Card.bru
    │   ├── 02-Get Card.bru
    │   ├── 03-Top Up Card.bru
    │   ├── 04-Top Up Idempotency.bru
    │   └── 05-Get Card Transactions.bru
    ├── 02-stations/
    │   ├── 01-List Stations.bru
    │   ├── 02-Filter MRT.bru
    │   └── 03-Filter TransJakarta.bru
    ├── 03-gate/
    │   ├── 01-Tap In.bru
    │   ├── 02-Tap In Duplicate Conflict.bru
    │   ├── 03-Tap Out.bru
    │   └── 04-Verify Fare Deduction Ledger.bru
    └── 04-errors/
        ├── 01-Card Not Found.bru
        ├── 02-Negative Top-Up.bru
        ├── 03-Provision Empty Card.bru
        ├── 04-Tap In Insufficient Balance.bru
        └── 05-Tap Out Without Journey.bru
```

---

## Quick Reference Summary

| Artifact | Location | Target Tool | Purpose |
| :--- | :--- | :--- | :--- |
| **OpenAPI YAML** | [`api/openapi/openapi.yaml`](file:///F:/__Projects/golingko/api/openapi/openapi.yaml) | Swagger, OpenAPI tools, SDK generators | Complete API contract & schema models |
| **OpenAPI JSON** | [`api/openapi/openapi.json`](file:///F:/__Projects/golingko/api/openapi/openapi.json) | Automated tooling, linters | JSON Schema & OpenAPI 3.1 format |
| **Interactive Docs** | [`api/openapi/index.html`](file:///F:/__Projects/golingko/api/openapi/index.html) or `http://localhost:8080/docs/` | Modern Web Browser | Interactive Scalar API reference & playground |
| **Postman Collection** | [`api/postman/golingko.postman_collection.json`](file:///F:/__Projects/golingko/api/postman/golingko.postman_collection.json) | Postman, Newman | Sequential test runner & environment chaining |
| **Postman Env** | [`api/postman/golingko_local.postman_environment.json`](file:///F:/__Projects/golingko/api/postman/golingko_local.postman_environment.json) | Postman, Newman | Environment variables (`baseUrl: http://localhost:8080`) |
| **Bruno Collection** | [`api/bruno/`](file:///F:/__Projects/golingko/api/bruno) | Bruno App, `@usebruno/cli` | File-based API requests with declarative assertions |

---

## 1. OpenAPI Specification

The OpenAPI specification covers all 8 REST endpoints, headers (such as `Idempotency-Key` and `X-Cache-Lookup`), and comprehensive data models representing Jakarta transit workflows.

### Viewing the Interactive Documentation

When running the Golingko backend server:
```bash
go run ./cmd/golingko
```
Open your browser to:
- **`http://localhost:8080/docs/`**

This serves a high-performance **Scalar API Reference** UI reading directly from `api/openapi/openapi.yaml`. You can explore schemas, review detailed parameter requirements, and execute live API calls directly from the browser.

Alternatively, you can open [`api/openapi/index.html`](file:///F:/__Projects/golingko/api/openapi/index.html) directly with any static server or VS Code Live Server extension.

### Endpoints Documented

| Group | Method & Route | Description |
| :--- | :--- | :--- |
| **Health** | `GET /health` | System liveness & PostgreSQL connection status |
| **Cards** | `POST /api/cards` | Provision 16-digit transit card with initial balance |
| **Cards** | `GET /api/cards/{number}` | Retrieve card metadata, status, and current balance |
| **Cards** | `POST /api/cards/{number}/top-up` | Atomic balance top-up with `Idempotency-Key` header |
| **Cards** | `GET /api/cards/{number}/transactions` | Double-entry audit ledger query with `?limit=` |
| **Stations** | `GET /api/stations` | Multi-modal station directory with optional `?mode=` filter |
| **Gate** | `POST /api/gate/tap-in` | Turnstile entry gate with anti-tailgating validation |
| **Gate** | `POST /api/gate/tap-out` | Turnstile exit with 180m JakLingko integrated tariff clearing |

---

## 2. Postman Test Suite

The Postman collection is organized into a realistic transit lifecycle scenario where variables (`cardNumber`, `topUpIdempotencyKey`) are dynamically extracted from responses and chained to subsequent requests.

### Importing Into Postman

1. Open **Postman**.
2. Click **Import** (top left).
3. Select both files:
   - `api/postman/golingko.postman_collection.json`
   - `api/postman/golingko_local.postman_environment.json`
4. Select the environment **Golingko Local Environment** in the top-right environment selector.

### Running with Newman (CLI)

You can execute the automated test collection in headless CI/CD mode using Newman:

```bash
# Using npm / npx
npx newman run api/postman/golingko.postman_collection.json -e api/postman/golingko_local.postman_environment.json

# Or using the Makefile target
make test-api-postman
```

### Test Folders & Scenarios

- **`00 Health Check`**: Asserts 200 OK and verified health state.
- **`01 Cards & Ledger`**:
  - `01 Create Transit Card`: Asserts 201 Created, sets `{{cardNumber}}` from generated response.
  - `02 Get Card Details`: Asserts 200 OK and validates status is `ACTIVE`.
  - `03 Top Up Card (Atomic & Idempotent)`: Credits Rp 25,000, checks balance = Rp 75,000, generates unique idempotency key.
  - `04 Top Up Card Idempotency Cache Hit`: Replays identical `Idempotency-Key` and asserts header `X-Cache-Lookup: HIT`.
  - `05 Get Card Transactions`: Verifies `TOP_UP` entry in ledger.
- **`02 Stations Directory`**:
  - `01 List All Stations`: Verifies full multi-modal network (`MRT-BHI`, `TJ-KOTA`, etc.).
  - `02 Filter MRT Stations`: Queries `?mode=MRT` and asserts all items have mode `MRT`.
  - `03 Filter TransJakarta Stations`: Queries `?mode=TRANSJAKARTA`.
- **`03 Gate Commute Lifecycle`**:
  - `01 Gate Tap-In (MRT Bundaran HI)`: Turnstile entry, asserts `gate_opened: true` and `status: IN_PROGRESS`.
  - `02 Tap-In Anti-Tailgating Conflict`: Re-taps with active trip, asserts 409 Conflict.
  - `03 Gate Tap-Out & Tariff Clearance (MRT Lebak Bulus)`: Exit gate, asserts fare deducted, status `COMPLETED`.
  - `04 Verify FARE_DEDUCTION in Ledger`: Validates debit record logged in audit ledger.
- **`04 Error Handling & Edge Cases`**:
  - `01 Card Not Found`: Asserts 404 on invalid card number.
  - `02 Negative Top-Up Amount`: Asserts 422 Unprocessable Entity.
  - `03 Provision Zero Balance Card`: Creates card with Rp 0 balance.
  - `04 Tap-In With Insufficient Balance`: Asserts 402 Payment Required (below Rp 5,000 threshold).
  - `05 Tap-Out Without Active Journey`: Asserts 409 Conflict.

---

## 3. Bruno Git-Friendly Test Suite

[Bruno](https://www.usebruno.com/) is an open-source, Git-friendly API client that stores collections directly in the repository as plain-text `.bru` files.

### Opening in Bruno App

1. Launch **Bruno**.
2. Click **Open Collection**.
3. Select the [`api/bruno`](file:///F:/__Projects/golingko/api/bruno) directory in this repository.
4. Select the **Local** environment from the top-right dropdown.
5. Click **Run** on any folder or request.

### Running with Bruno CLI

To run the entire collection from the terminal:

```bash
# Using Bruno CLI via npx
npx @usebruno/cli run api/bruno --env Local

# Or using the Makefile target
make test-api-bruno
```

---

## 4. Transit Business Rules Verified by Tests

The tests systematically assert all core transit domain invariants:

1. **Anti-Tailgating Protocol**: Cards are rejected with `409 Conflict` if passenger attempts a second tap-in while a trip is currently `IN_PROGRESS`.
2. **Minimum Balance Entry Rule**: Turnstiles reject entry with `402 Payment Required` if card balance is less than `Rp 5,000`.
3. **JakLingko Integrated Tariffs**: Multi-operator commutes are capped at `Rp 10,000` within 180 minutes.
4. **Idempotency Guarantee**: Top-ups and gate operations accepting `Idempotency-Key` headers cache 2xx responses for 5 minutes, returning `X-Cache-Lookup: HIT` without double billing.
5. **Double-Entry Financial Auditing**: Every top-up and tap-out writes immutable transaction records to `ledger_transactions` with precise before-and-after balances.
