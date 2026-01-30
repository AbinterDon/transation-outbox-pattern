package worker

import (
	"context"
	"log"

	"transation-outbox-pattern/internal/consumer"
)

// InMemoryBus simulates a message broker (RabbitMQ/Kafka) by directly calling the consumer.
type InMemoryBus struct {
	consumer *consumer.InventoryConsumer
}

func NewInMemoryBus(consumer *consumer.InventoryConsumer) *InMemoryBus {
	return &InMemoryBus{consumer: consumer}
}

// Publish implements the Publisher interface.
func (b *InMemoryBus) Publish(ctx context.Context, id string, topic string, payload []byte) error {
	log.Printf("[Bus] Relaying message %s directly to Consumer...", id)

	// In a real app, this would route based on topic.
	// For this demo, we assume all messages go to InventoryConsumer.
	return b.consumer.HandleMessage(ctx, id, payload)
}
