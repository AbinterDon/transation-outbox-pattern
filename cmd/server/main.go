package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"transation-outbox-pattern/internal/db"
	"transation-outbox-pattern/internal/usecase"
	"transation-outbox-pattern/internal/worker"
)

type CreateOrderRequest struct {
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
}

func main() {
	ctx := context.Background()

	// 1. Initialize DB
	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	// 2. Initialize Service
	orderService := usecase.NewOrderService(pool)

	// 3. Initialize & Start Outbox Processor
	// RabbitMQ Publisher
	rabbitURL := "amqp://user:password@localhost:5672/"
	msgBus, err := worker.NewRabbitMQPublisher(rabbitURL, "order_events")
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ publisher: %v", err)
	}
	defer msgBus.Close()

	// Start 5 concurrent workers
	for i := 1; i <= 5; i++ {
		processor := worker.NewOutboxProcessor(i, pool, msgBus)
		go processor.Start(ctx)
	}

	// 3. HTTP Handlers
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		order, err := orderService.CreateOrder(r.Context(), req.UserID, req.Amount)
		if err != nil {
			log.Printf("Failed to create order: %v\n", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	})

	// 4. Start Server
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
