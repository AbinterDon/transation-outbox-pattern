# Transactional Outbox Pattern Example

[中文版 (Chinese Version)](README_zh.md)

## Introduction
This project is an example implementation of the **Transactional Outbox Pattern** using Go and PostgreSQL.
It simulates an order system to solve the **Dual Write Problem** in microservices architectures, ensuring:
1.  **Atomicity**: Consistency between order creation and event publishing.
2.  **Reliability (At-least-once Delivery)**: Messages are guaranteed to be delivered eventually, even in the event of failures.
3.  **Idempotency**: The Consumer handles duplicate messages to prevent business logic from being executed multiple times.

## Tech Stack
-   Go (Golang)
-   PostgreSQL (Database & Queue implementation)
-   Docker & Docker Compose

## Quick Start

### 1. Start the Database
```bash
docker-compose up -d
```
This starts PostgreSQL and automatically executes `migrations/init.sql` to create the required tables (`orders`, `outbox`, `processed_messages`).
*Note: The project maps PostgreSQL to host port `5433` to avoid conflicts with default ports.*

### 2. Start the API Server
```bash
go run cmd/server/main.go
```
The server listens on port `:8080`.

### 3. Test Order Creation (Atomic Write)
Send an HTTP POST request to create an order:
```bash
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-d '{"user_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 100.0}'
```
Check the server logs. You will see an order being created and a corresponding event in the `outbox` table. The background `OutboxProcessor` automatically picks it up and processes it.

### 4. Test Idempotency
We provide a Replay script to simulate "receiving the same message twice":
```bash
go run cmd/replay/main.go
```
This script attempts to send the same Message ID to the Consumer twice.
**Expected Result**: The first attempt is processed successfully (`PROCESSING`), and the second attempt is skipped (`SKIPPING duplicate message`).

## Project Structure
-   `cmd/server`: API Server entry point.
-   `cmd/replay`: Tool for testing idempotency.
-   `internal/usecase`: Business logic (`CreateOrder`), containing the core implementation of the Transactional Outbox.
-   `internal/worker`: Background Outbox Processor (Producer), responsible for Polling and Relaying. Uses `SKIP LOCKED` for concurrency safety.
-   `internal/consumer`: Simulated downstream service (Consumer), implementing Idempotency by checking the `processed_messages` table.
-   `migrations`: Database initialization SQL.

## Key Implementation Details
*   **CreateOrder**: Performs `INSERT orders` and `INSERT outbox` within a single DB Transaction.
*   **Outbox Processor**: Uses `SELECT ... FOR UPDATE SKIP LOCKED` to prevent race conditions between multiple processors.
*   **Delivery Guarantee**: Adopts **At-least-once** strategy; Consumers must implement idempotency to handle potential duplicate messages.
