package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"transation-outbox-pattern/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Publisher interface {
	Publish(ctx context.Context, id string, topic string, payload []byte) error
}

// MockPublisher to simulate message broker
type MockPublisher struct{}

func (p *MockPublisher) Publish(ctx context.Context, id string, topic string, payload []byte) error {
	// Simulate network latency
	time.Sleep(100 * time.Millisecond)
	log.Printf("[Publisher] Sent message %s to topic %s: %s", id, topic, string(payload))
	return nil
}

type OutboxProcessor struct {
	id        int
	db        *pgxpool.Pool
	publisher Publisher
}

func NewOutboxProcessor(id int, db *pgxpool.Pool, pub Publisher) *OutboxProcessor {
	return &OutboxProcessor{
		id:        id,
		db:        db,
		publisher: pub,
	}
}

func (p *OutboxProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Printf("[Worker-%d] Started", p.id)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *OutboxProcessor) processBatch(ctx context.Context) {
	// 1. Fetch pending events with lock to avoid concurrency issues
	rows, err := p.db.Query(ctx, `
		SELECT id, aggregate_id, payload, status 
		FROM outbox 
		WHERE status = 'PENDING' 
		ORDER BY created_at ASC 
		LIMIT 10 
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		log.Printf("[Worker-%d] Failed to fetch events: %v", p.id, err)
		return
	}
	defer rows.Close()

	var events []model.OutboxEvent
	for rows.Next() {
		var e model.OutboxEvent
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.Payload, &e.Status); err != nil {
			log.Printf("[Worker-%d] Failed to scan row: %v", p.id, err)
			continue
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		return
	}

	log.Printf("[Worker-%d] Processing batch of %d events", p.id, len(events))

	// 2. Process each event
	for _, event := range events {
		p.processEvent(ctx, event)
	}
}

func (p *OutboxProcessor) processEvent(ctx context.Context, event model.OutboxEvent) {
	// Parse payload to find topic or routing key (simplified here)
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payloadMap); err != nil {
		log.Printf("[Worker-%d] Invalid payload: %v", p.id, err)
		return // Should mark as FAILED in real life
	}

	eventType, _ := payloadMap["event_type"].(string)

	// 3. Publish to Broker
	err := p.publisher.Publish(ctx, event.ID, eventType, event.Payload)
	if err != nil {
		log.Printf("[Worker-%d] Failed to publish event %s: %v", p.id, event.ID, err)
		return // Will be retried next tick
	}

	// 4. Delete processed event (or update status to PROCESSED)
	_, err = p.db.Exec(ctx, "DELETE FROM outbox WHERE id = $1", event.ID)
	if err != nil {
		log.Printf("[Worker-%d] Failed to delete event %s: %v", p.id, event.ID, err)
		// This is where "At-Least-Once" comes in. If we fail here, we republish next time.
	} else {
		log.Printf("[Worker-%d] Successfully processed and deleted event %s", p.id, event.ID)
	}
}
