package setup

import (
	"fmt"
	"time"
)

// BenchmarkEvent represents a pre-generated event for benchmarks
type BenchmarkEvent struct {
	ID        string
	Type      string
	Tags      map[string]string
	Data      []byte
	Timestamp time.Time
}

// BenchmarkDataConfig defines the size and types of benchmark data
type BenchmarkDataConfig struct {
	SingleEvents    int
	RealisticEvents int
	AppendIfEvents  int
	MixedEvents     int
}

// Default benchmark data configurations
var BenchmarkDataSizes = map[string]BenchmarkDataConfig{
	"tiny": {
		SingleEvents:    100,    // 100 single event operations
		RealisticEvents: 100,    // 100 realistic batch operations (1-12 events)
		AppendIfEvents:  100,    // 100 conditional append operations
		MixedEvents:     50,     // 50 mixed operation sequences
	},
	"small": {
		SingleEvents:    1000,   // 1000 single event operations
		RealisticEvents: 1000,   // 1000 realistic batch operations (1-12 events)
		AppendIfEvents:  1000,   // 1000 conditional append operations
		MixedEvents:     500,    // 500 mixed operation sequences
	},
}

// Realistic batch sizes for real-world usage
var RealisticBatchSizes = []int{
	1,   // Single event (most common)
	2,   // Two related events
	3,   // Small transaction
	5,   // Typical business operation
	8,   // Medium batch
	12,  // Larger business operation
	20,  // Uncommon but possible
	50,  // Rare, bulk operations
	100, // Very rare, data migration
}

// GenerateBenchmarkData creates pre-generated events for benchmarks
func GenerateBenchmarkData(config BenchmarkDataConfig) map[string][]BenchmarkEvent {
	data := make(map[string][]BenchmarkEvent)
	
	// Generate single events
	data["single"] = generateSingleEvents(config.SingleEvents)
	
	// Generate realistic batch events (most common scenarios)
	data["realistic"] = generateRealisticBatchEvents(config.RealisticEvents)
	
	// Generate AppendIf events
	data["appendif"] = generateAppendIfEvents(config.AppendIfEvents)
	
	// Generate mixed event types
	data["mixed"] = generateMixedEvents(config.MixedEvents)
	
	return data
}

// generateSingleEvents creates unique single events for benchmarks
func generateSingleEvents(count int) []BenchmarkEvent {
	events := make([]BenchmarkEvent, count)
	
	for i := 0; i < count; i++ {
		uniqueID := fmt.Sprintf("single_%d", i)
		events[i] = BenchmarkEvent{
			ID:   uniqueID,
			Type: "TestEvent",
			Tags: map[string]string{
				"test":      "single",
				"unique_id": uniqueID,
				"sequence":  fmt.Sprintf("%d", i),
			},
			Data:      []byte(fmt.Sprintf(`{"value":"test","unique_id":"%s","sequence":%d}`, uniqueID, i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
	}
	
	return events
}

// generateBatchEvents creates batch events for different batch sizes
func generateBatchEvents(count, batchSize int) []BenchmarkEvent {
	events := make([]BenchmarkEvent, count)
	
	for i := 0; i < count; i++ {
		batchID := fmt.Sprintf("batch_%d_%d", batchSize, i)
		events[i] = BenchmarkEvent{
			ID:   batchID,
			Type: "TestEvent",
			Tags: map[string]string{
				"test":      "batch",
				"batch_id":  batchID,
				"batch_size": fmt.Sprintf("%d", batchSize),
				"sequence":  fmt.Sprintf("%d", i),
			},
			Data:      []byte(fmt.Sprintf(`{"batch_id":"%s","batch_size":%d,"sequence":%d}`, batchID, batchSize, i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
	}
	
	return events
}

// generateRealisticBatchEvents creates events with realistic batch sizes
func generateRealisticBatchEvents(count int) []BenchmarkEvent {
	events := make([]BenchmarkEvent, count)
	
	for i := 0; i < count; i++ {
		// Use realistic batch sizes (1-12 most common)
		batchSize := RealisticBatchSizes[i%len(RealisticBatchSizes)]
		batchID := fmt.Sprintf("realistic_%d_%d", batchSize, i)
		
		events[i] = BenchmarkEvent{
			ID:   batchID,
			Type: "RealisticEvent",
			Tags: map[string]string{
				"test":       "realistic",
				"batch_id":   batchID,
				"batch_size": fmt.Sprintf("%d", batchSize),
				"sequence":   fmt.Sprintf("%d", i),
				"scenario":   getRealisticScenario(batchSize),
			},
			Data:      []byte(fmt.Sprintf(`{"batch_id":"%s","batch_size":%d,"scenario":"%s","sequence":%d}`, batchID, batchSize, getRealisticScenario(batchSize), i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
	}
	
	return events
}

// getRealisticScenario returns a realistic business scenario for the batch size
func getRealisticScenario(batchSize int) string {
	switch {
	case batchSize == 1:
		return "single_event"
	case batchSize <= 3:
		return "small_transaction"
	case batchSize <= 8:
		return "business_operation"
	case batchSize <= 12:
		return "complex_workflow"
	case batchSize <= 20:
		return "bulk_operation"
	default:
		return "data_migration"
	}
}

// generateAppendIfEvents creates events for conditional append benchmarks
func generateAppendIfEvents(count int) []BenchmarkEvent {
	events := make([]BenchmarkEvent, count)
	
	for i := 0; i < count; i++ {
		uniqueID := fmt.Sprintf("appendif_%d", i)
		events[i] = BenchmarkEvent{
			ID:   uniqueID,
			Type: "TestEvent",
			Tags: map[string]string{
				"test":      "appendif",
				"unique_id": uniqueID,
				"sequence":  fmt.Sprintf("%d", i),
			},
			Data:      []byte(fmt.Sprintf(`{"value":"test","unique_id":"%s","sequence":%d}`, uniqueID, i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
	}
	
	return events
}

// generateMixedEvents creates events with different types for mixed operation benchmarks
func generateMixedEvents(count int) []BenchmarkEvent {
	events := make([]BenchmarkEvent, count)
	
	eventTypes := []string{"DataUpdate", "UserAction", "SystemEvent", "BusinessRule"}
	
	for i := 0; i < count; i++ {
		eventType := eventTypes[i%len(eventTypes)]
		uniqueID := fmt.Sprintf("mixed_%s_%d", eventType, i)
		
		events[i] = BenchmarkEvent{
			ID:   uniqueID,
			Type: eventType,
			Tags: map[string]string{
				"test":      "mixed",
				"event_type": eventType,
				"unique_id": uniqueID,
				"sequence":  fmt.Sprintf("%d", i),
			},
			Data:      []byte(fmt.Sprintf(`{"event_type":"%s","unique_id":"%s","sequence":%d}`, eventType, uniqueID, i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
	}
	
	return events
}




