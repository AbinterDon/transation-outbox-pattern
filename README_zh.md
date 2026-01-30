# Transactional Outbox Pattern Example

[English Version](README.md)

## 簡介
這是一個使用 Go 和 PostgreSQL 實作 **Transactional Outbox Pattern** 的範例專案。
模擬了一個訂單系統，解決了微服務架構中常見的 **Dual Write Problem**，並確保了：
1. **原子性 (Atomicity)**: 訂單建立與事件發送的一致性。
2. **可靠性 (At-least-once Delivery)**: 即使中間發生故障，訊息最終也能成功送達。
3. **冪等性 (Idempotency)**: Consumer 端能處理重複訊息，避免業務邏輯重複執行。

## 技術棧
- Go (Golang)
- PostgreSQL (Database & Queue implementation)
- Docker & Docker Compose

## 快速開始

### 1. 啟動資料庫
```bash
docker-compose up -d
```
這會啟動 PostgreSQL 並自動執行 `migrations/init.sql` 建立所需的 Tables (`orders`, `outbox`, `processed_messages`)。
*注意：本專案設定 PostgreSQL 映射到 host 的 `5433` port 以避免與預設 port 衝突。*

### 2. 啟動 API Server
```bash
go run cmd/server/main.go
```
Server 會監聽 `:8080` port。

### 3. 測試建立訂單 (Atomic Write)
發送 HTTP POST 請求來建立訂單：
```bash
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-d '{"user_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 100.0}'
```
觀察 Server log，你會建立一個訂單，並且可以在 `outbox` table 中看到對應的事件。背景的 `OutboxProcessor` 會自動撈取並處理它。

### 4. 測試冪等性 (Idempotency)
我們提供了一個 Replay script 來模擬「重複收到同一則訊息」的情況：
```bash
go run cmd/replay/main.go
```
此腳本會嘗試發送兩次相同的 Message ID 給 Consumer。
**預期結果**：第一次成功處理 (PROCESSING)，第二次顯示 "SKIPPING duplicate message"。

## 專案結構
- `cmd/server`: API Server 進入點。
- `cmd/replay`: 用於測試冪等性的工具。
- `internal/usecase`: 業務邏輯 (CreateOrder)，包含 Transactional Outbox 的核心實作。
- `internal/worker`: 背景 Outbox Processor (Producer)，負責 Polling 與 Relay。使用 `SKIP LOCKED` 確保併發安全。
- `internal/consumer`: 模擬下游服務 (Consumer)，透過檢查 `processed_messages` table 資料來實作 Idempotency。
- `migrations`: 資料庫初始化 SQL。

## 關鍵實作細節
*   **CreateOrder**: 在單一 DB Transaction 中同時 `INSERT orders` 與 `INSERT outbox`。
*   **Outbox Processor**: 使用 `SELECT ... FOR UPDATE SKIP LOCKED` 來避免多個 Processor 搶同一筆資料。
*   **Delivery Guarantee**: 採用 **At-least-once** 策略，Consumer 必須實作冪等性來處理可能的重複發送。
