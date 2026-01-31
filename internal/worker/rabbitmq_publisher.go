package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

func NewRabbitMQPublisher(url string, queueName string) (*RabbitMQPublisher, error) {
	var conn *amqp.Connection
	var err error

	// Retry connection because RabbitMQ takes time to start in Docker
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ, retrying in 2s... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, id string, topic string, payload []byte) error {
	err := p.channel.PublishWithContext(ctx,
		"",      // exchange
		p.queue, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			MessageId:    id,
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent, // Ensure message is persisted to disk
		})

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("[RabbitMQ] Published message %s to queue %s", id, p.queue)
	return nil
}

func (p *RabbitMQPublisher) Close() {
	p.channel.Close()
	p.conn.Close()
}
