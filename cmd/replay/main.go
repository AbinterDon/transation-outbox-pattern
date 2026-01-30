package main

import (
	"context"
	"log"

	"transation-outbox-pattern/internal/consumer"
	"transation-outbox-pattern/internal/db"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	svc := consumer.NewInventoryConsumer(pool)

	// Simulate a message ID that created by the previous run
	// You might need to change this ID to what you see in the logs or DB
	// For this test, let's create a NEW message, process it twice.

	msgID := "550e8400-e29b-41d4-a716-446655449999"
	payload := []byte(`{"order_id": "order-123", "event_type": "OrderCreated"}`)

	log.Println("1st Attempt:")
	if err := svc.HandleMessage(ctx, msgID, payload); err != nil {
		log.Fatalf("First attempt failed: %v", err)
	}

	log.Println("2nd Attempt (Should be skipped):")
	if err := svc.HandleMessage(ctx, msgID, payload); err != nil {
		log.Fatalf("Second attempt failed: %v", err)
	}

	log.Println("Done")
}
