package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "go-crablet/internal/grpc-app/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEventStoreServiceClient(conn)

	// Benchmark parameters
	duration := 5 * time.Second
	concurrentUsers := 10
	requestsPerSecond := 100

	fmt.Printf("Starting gRPC benchmark for %v\n", duration)
	fmt.Printf("Concurrent users: %d\n", concurrentUsers)
	fmt.Printf("Target RPS: %d\n", requestsPerSecond)

	// Create a channel to coordinate goroutines
	done := make(chan bool)
	results := make(chan time.Duration, 10000)

	// Start worker goroutines
	for i := 0; i < concurrentUsers; i++ {
		go func(workerID int) {
			ticker := time.NewTicker(time.Second / time.Duration(requestsPerSecond/concurrentUsers))
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					start := time.Now()

					// Make gRPC call
					_, err := client.Read(context.Background(), &pb.ReadRequest{
						Query: &pb.Query{
							Items: []*pb.QueryItem{
								{
									Types: []string{"TestEvent"},
									Tags:  []string{"test:value"},
								},
							},
						},
					})

					duration := time.Since(start)
					results <- duration

					if err != nil {
						log.Printf("Worker %d: Error: %v", workerID, err)
					}
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(duration)
	close(done)

	// Collect results
	var totalRequests int
	var totalDuration time.Duration
	var minDuration time.Duration = time.Hour
	var maxDuration time.Duration

	for {
		select {
		case duration := <-results:
			totalRequests++
			totalDuration += duration
			if duration < minDuration {
				minDuration = duration
			}
			if duration > maxDuration {
				maxDuration = duration
			}
		default:
			goto done
		}
	}
done:

	// Print results
	fmt.Printf("\nBenchmark Results:\n")
	fmt.Printf("Total requests: %d\n", totalRequests)
	fmt.Printf("Average response time: %v\n", totalDuration/time.Duration(totalRequests))
	fmt.Printf("Min response time: %v\n", minDuration)
	fmt.Printf("Max response time: %v\n", maxDuration)
	fmt.Printf("Requests per second: %.2f\n", float64(totalRequests)/duration.Seconds())
}
