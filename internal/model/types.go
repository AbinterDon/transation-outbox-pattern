package model

import (
	"time"
)

type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type OutboxEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	Payload     []byte    `json:"payload"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
