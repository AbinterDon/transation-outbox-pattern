package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"transation-outbox-pattern/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderService struct {
	db *pgxpool.Pool
}

func NewOrderService(db *pgxpool.Pool) *OrderService {
	return &OrderService{db: db}
}

// CreateOrder executes the "Order Creation" and "Outbox Event Creation" in a single atomic transaction.
func (s *OrderService) CreateOrder(ctx context.Context, userID string, amount float64) (*model.Order, error) {
	// 1. Start Transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer rollback in case of panic or error (Rollback is a no-op if Committed)
	defer tx.Rollback(ctx)

	// 2. Insert Order
	orderID := uuid.New().String()
	order := &model.Order{
		ID:        orderID,
		UserID:    userID,
		Amount:    amount,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, order.ID, order.UserID, order.Amount, order.Status, order.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	// 3. Create Event Payload
	eventPayload, err := json.Marshal(map[string]interface{}{
		"event_type": "OrderCreated",
		"order_id":   order.ID,
		"user_id":    order.UserID,
		"amount":     order.Amount,
		"timestamp":  time.Now().Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// 4. Insert Outbox Event (Within the SAME transaction)
	outboxID := uuid.New().String()
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox (id, aggregate_id, payload, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, outboxID, order.ID, eventPayload, "PENDING", time.Now())

	if err != nil {
		return nil, fmt.Errorf("failed to insert outbox event: %w", err)
	}

	// 5. Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return order, nil
}

// CreateOrderUnsafe demonstrates the WRONG way (dual write without transaction) for comparison
/*
func (s *OrderService) CreateOrderUnsafe(...) {
    // Insert Order
    // ... basic insert ...

    // IF THIS FAILS, we have a User but no Event!
    // Send Message to Broker / Insert Outbox
}
*/
