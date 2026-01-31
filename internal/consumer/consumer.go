package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryConsumer struct {
	db *pgxpool.Pool
}

func NewInventoryConsumer(db *pgxpool.Pool) *InventoryConsumer {
	return &InventoryConsumer{db: db}
}

type EventPayload struct {
	ID        string `json:"order_id"` // Using OrderID as the unique key for this example event
	EventType string `json:"event_type"`
}

// HandleMessage processes the event with Idempotency enforcement
func (c *InventoryConsumer) HandleMessage(ctx context.Context, messageID string, payload []byte) error {
	// 1. Start Transaction
	tx, err := c.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Check Idempotency (Have we processed this MessageID before?)
	// Note: We use the messageID (UUID from Outbox) as the key.
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM processed_messages WHERE message_id = $1)", messageID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if exists {
		log.Printf("[Consumer] SKIPPING duplicate message %s", messageID)
		return nil // Return nil so we acknowledge the message broker
	}

	// 3. Process Business Logic (Simulate deducting inventory)
	// In a real app, this would update inventory tables, etc.
	var event EventPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("[Consumer] PROCESSING order %s (Deducting Inventory...)", event.ID)
	// Simulate work
	time.Sleep(50 * time.Millisecond)

	// UPDATE Order status to COMPLETED
	_, err = tx.Exec(ctx, "UPDATE orders SET status = 'COMPLETED' WHERE id = $1", event.ID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// 4. Record Processed Message
	_, err = tx.Exec(ctx, "INSERT INTO processed_messages (message_id, processed_at) VALUES ($1, $2)", messageID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record processed message: %w", err)
	}

	// 5. Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[Consumer] COMPLETED message %s", messageID)
	return nil
}
