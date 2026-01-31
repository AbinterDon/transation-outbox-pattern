package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"transation-outbox-pattern/internal/consumer"
	"transation-outbox-pattern/internal/db"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize DB
	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 2. Initialize Consumer Logic
	inventoryConsumer := consumer.NewInventoryConsumer(pool)

	// 3. Connect to RabbitMQ
	rabbitURL := "amqp://user:password@localhost:5672/"
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	queueName := "order_events"
	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack (IMPORTANT: we want manual ack)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// 4. Handle Messages
	go func() {
		for d := range msgs {
			log.Printf("[Worker-Consumer] Received message: %s", d.MessageId)

			// Call the core idempotency/business logic
			err := inventoryConsumer.HandleMessage(ctx, d.MessageId, d.Body)
			if err != nil {
				log.Printf("[Worker-Consumer] Failed to process message: %v", err)
				// Nack and requeue
				d.Nack(false, true)
				continue
			}

			// Acknowledge the message
			d.Ack(false)
		}
	}()

	log.Println("[Worker-Consumer] Waiting for messages. To exit press CTRL+C")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("[Worker-Consumer] Shutting down...")
}
