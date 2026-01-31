package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

func main() {
	start := time.Now()
	var wg sync.WaitGroup

	totalRequests := 50

	fmt.Printf("Starting stress test with %d requests...\n", totalRequests)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Generate a random UUID for user_id to satisfy DB constraint
			userID := uuid.New().String()
			jsonBody := []byte(fmt.Sprintf(`{"user_id": "%s", "amount": 100.0}`, userID))

			resp, err := http.Post("http://localhost:8080/orders", "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				fmt.Printf("Request %d failed: %v\n", id, err)
				return
			}
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("Finished %d requests in %v\n", totalRequests, duration)
}
