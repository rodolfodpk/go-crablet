package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "go-crablet/internal/grpc-app/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BenchmarkResult struct {
	Operation    string
	Duration     time.Duration
	Success      bool
	Error        string
	ResponseSize int
}

type BenchmarkStats struct {
	Operation     string
	TotalRequests int
	SuccessCount  int
	ErrorCount    int
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	AvgDuration   time.Duration
	P95Duration   time.Duration
	P99Duration   time.Duration
}

func main() {
	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEventStoreServiceClient(conn)

	// Test health endpoint first
	ctx := context.Background()
	healthResp, err := client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	log.Printf("Health check passed: %s", healthResp.Status)

	// Run benchmarks
	results := runBenchmarks(client)

	// Calculate and display statistics
	stats := calculateStats(results)
	displayResults(stats)
}

func runBenchmarks(client pb.EventStoreServiceClient) []BenchmarkResult {
	var results []BenchmarkResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Benchmark parameters - quick 3-second test
	numWorkers := 5
	requestsPerWorker := 20
	totalRequests := numWorkers * requestsPerWorker

	log.Printf("Starting 3-second benchmark with %d workers, %d requests each (%d total)",
		numWorkers, requestsPerWorker, totalRequests)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerResults := runWorker(client, workerID, requestsPerWorker)

			mu.Lock()
			results = append(results, workerResults...)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	return results
}

func runWorker(client pb.EventStoreServiceClient, workerID, numRequests int) []BenchmarkResult {
	var results []BenchmarkResult
	ctx := context.Background()

	for i := 0; i < numRequests; i++ {
		// Health check
		start := time.Now()
		_, err := client.Health(ctx, &pb.HealthRequest{})
		duration := time.Since(start)

		results = append(results, BenchmarkResult{
			Operation: "Health",
			Duration:  duration,
			Success:   err == nil,
			Error:     getErrorString(err),
		})

		// Append event
		start = time.Now()
		_, err = client.Append(ctx, &pb.AppendRequest{
			Events: []*pb.InputEvent{
				{
					Type: "TestEvent",
					Tags: []string{
						fmt.Sprintf("worker:%d", workerID),
						fmt.Sprintf("iteration:%d", i),
						"benchmark:grpc",
					},
					Data: fmt.Sprintf(`{"message": "test event %d from worker %d", "timestamp": "%s"}`,
						i, workerID, time.Now().Format(time.RFC3339)),
				},
			},
		})
		duration = time.Since(start)

		if err != nil {
			log.Printf("Append error: %v", err)
		}

		results = append(results, BenchmarkResult{
			Operation: "Append",
			Duration:  duration,
			Success:   err == nil,
			Error:     getErrorString(err),
		})

		// Read events
		start = time.Now()
		readResp, err := client.Read(ctx, &pb.ReadRequest{
			Query: &pb.Query{
				Items: []*pb.QueryItem{
					{
						Types: []string{"TestEvent"},
						Tags:  []string{"benchmark:grpc"},
					},
				},
			},
		})
		duration = time.Since(start)

		if err != nil {
			log.Printf("Read error: %v", err)
		}

		responseSize := 0
		if err == nil && readResp != nil {
			responseSize = len(readResp.Events)
		}

		results = append(results, BenchmarkResult{
			Operation:    "Read",
			Duration:     duration,
			Success:      err == nil,
			Error:        getErrorString(err),
			ResponseSize: responseSize,
		})

		// Small delay to avoid overwhelming the server
		time.Sleep(10 * time.Millisecond)
	}

	return results
}

func getErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func calculateStats(results []BenchmarkResult) map[string]BenchmarkStats {
	statsByOp := make(map[string][]BenchmarkResult)

	// Group results by operation
	for _, result := range results {
		statsByOp[result.Operation] = append(statsByOp[result.Operation], result)
	}

	// Calculate statistics for each operation
	stats := make(map[string]BenchmarkStats)

	for op, opResults := range statsByOp {
		if len(opResults) == 0 {
			continue
		}

		var durations []time.Duration
		successCount := 0
		totalDuration := time.Duration(0)
		minDuration := opResults[0].Duration
		maxDuration := opResults[0].Duration

		for _, result := range opResults {
			durations = append(durations, result.Duration)
			totalDuration += result.Duration

			if result.Success {
				successCount++
			}

			if result.Duration < minDuration {
				minDuration = result.Duration
			}
			if result.Duration > maxDuration {
				maxDuration = result.Duration
			}
		}

		// Sort durations for percentile calculation
		for i := 0; i < len(durations); i++ {
			for j := i + 1; j < len(durations); j++ {
				if durations[i] > durations[j] {
					durations[i], durations[j] = durations[j], durations[i]
				}
			}
		}

		avgDuration := totalDuration / time.Duration(len(opResults))
		p95Index := int(float64(len(durations)) * 0.95)
		p99Index := int(float64(len(durations)) * 0.99)

		if p95Index >= len(durations) {
			p95Index = len(durations) - 1
		}
		if p99Index >= len(durations) {
			p99Index = len(durations) - 1
		}

		stats[op] = BenchmarkStats{
			Operation:     op,
			TotalRequests: len(opResults),
			SuccessCount:  successCount,
			ErrorCount:    len(opResults) - successCount,
			TotalDuration: totalDuration,
			MinDuration:   minDuration,
			MaxDuration:   maxDuration,
			AvgDuration:   avgDuration,
			P95Duration:   durations[p95Index],
			P99Duration:   durations[p99Index],
		}
	}

	return stats
}

func displayResults(stats map[string]BenchmarkStats) {
	fmt.Println("\n=== gRPC Benchmark Results ===")
	fmt.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Println()

	for op, stat := range stats {
		fmt.Printf("Operation: %s\n", op)
		fmt.Printf("  Total Requests: %d\n", stat.TotalRequests)
		fmt.Printf("  Success Rate: %.2f%% (%d/%d)\n",
			float64(stat.SuccessCount)/float64(stat.TotalRequests)*100,
			stat.SuccessCount, stat.TotalRequests)
		fmt.Printf("  Error Rate: %.2f%% (%d/%d)\n",
			float64(stat.ErrorCount)/float64(stat.TotalRequests)*100,
			stat.ErrorCount, stat.TotalRequests)
		fmt.Printf("  Duration Statistics:\n")
		fmt.Printf("    Min: %v\n", stat.MinDuration)
		fmt.Printf("    Max: %v\n", stat.MaxDuration)
		fmt.Printf("    Avg: %v\n", stat.AvgDuration)
		fmt.Printf("    P95: %v\n", stat.P95Duration)
		fmt.Printf("    P99: %v\n", stat.P99Duration)
		fmt.Printf("  Throughput: %.2f req/sec\n",
			float64(stat.TotalRequests)/stat.TotalDuration.Seconds())
		fmt.Println()
	}

	// Save results to file
	saveResultsToFile(stats)
}

func saveResultsToFile(stats map[string]BenchmarkStats) {
	filename := fmt.Sprintf("grpc-benchmark-%s.json", time.Now().Format("2006-01-02-15-04-05"))

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Failed to create results file: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(stats); err != nil {
		log.Printf("Failed to write results: %v", err)
		return
	}

	fmt.Printf("Results saved to: %s\n", filename)
}
