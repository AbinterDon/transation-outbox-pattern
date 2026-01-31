# Transactional Outbox Pattern Example

[中文版 (Chinese Version)](README_zh.md)

## System Architecture

### 🏗️ System Flow (Sequence Diagram)

```mermaid
sequenceDiagram
    participant Client
    participant API as Order API
    participant DB as Postgres (Orders + Outbox)
    participant Worker as Outbox Processor
    participant Consumer as Inventory Service

    Client->>API: POST /orders (Place Order)
    rect rgb(240, 248, 255)
    Note over API, DB: Transaction START
    API->>DB: INSERT into orders
    API->>DB: INSERT into outbox (OrderCreated Event)
    Note over API, DB: Transaction COMMIT (Atomicity!)
    end
    API-->>Client: 200 OK (Success)

    loop Every Second
        Worker->>DB: SELECT ... SKIP LOCKED (Fetch pending events)
        DB-->>Worker: Return event data
        Worker->>Consumer: Publish OrderCreated Event
        rect rgb(255, 240, 245)
        Note over Consumer, DB: Transaction START
        Consumer->>DB: Check processed_messages (Idempotency)
        Consumer->>DB: Business Logic (Deduct Inventory)
        Consumer->>DB: INSERT into processed_messages
        Note over Consumer, DB: Transaction COMMIT
        end
        Worker->>DB: DELETE from outbox (Handled)
    end
```

---

## 🔍 Core Mechanisms

### 1. Atomic Write
Ensures "Order Creation" and "Event Notification" are bound. Uses a single DB Transaction to write to both `orders` and `outbox` tables, solving the Dual Write problem.

### 2. High Concurrency Background Processing (Worker Pool)
Launches 5 concurrent `OutboxProcessor` (via Goroutines). Utilizes SQL `FOR UPDATE SKIP LOCKED` to allow multiple workers to process messages in parallel without race conditions.

### 3. Idempotency Guarantee
The downstream Consumer (Inventory Service) checks the `processed_messages` table before processing, ensuring business logic executes exactly once even if a message is received multiple times due to network retries.

---

## Quick Start

### 1. Start the Database
```bash
docker-compose up -d
```
*Note: PostgreSQL is mapped to port `5433`.*

### 2. Start the API Server
```bash
go run cmd/server/main.go
```

### 3. Concurrency Stress Test
```bash
go run cmd/stress_test/main.go
```
Sends 50 simultaneous requests. Observe server logs to see how `[Worker-1]` through `[Worker-5]` share the workload.

### 4. Verify Idempotency (Replay)
```bash
go run cmd/replay/main.go
```
Tests duplicate Message IDs and observes "SKIPPING" behavior in the consumer.

---

## Project Structure
- `cmd/server`: Main API & Worker pool entry point.
- `cmd/stress_test`: Concurrency testing tool.
- `internal/usecase`: Core atomic transaction logic.
- `internal/worker`: SKIP LOCKED background polling.
- `internal/consumer`: Idempotency logic.
- `migrations`: SQL Schema definitions.

