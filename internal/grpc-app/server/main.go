package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "go-crablet/internal/grpc-app/proto"
)

type server struct {
	pb.UnimplementedEventStoreServiceServer
	store dcb.EventStore
}

func (s *server) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok"}, nil
}

func (s *server) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	log.Printf("[gRPC] Read request: %+v", req)
	start := time.Now()

	// Convert proto query to DCB query
	query := convertProtoQuery(req.Query)
	options := convertProtoReadOptions(req.Options)

	// Execute read
	result, err := s.store.Read(ctx, query, options)
	if err != nil {
		log.Printf("[gRPC] Read error: %v", err)
		return nil, err
	}

	duration := time.Since(start)

	// Convert DCB events to proto events
	events := make([]*pb.Event, len(result.Events))
	for i, event := range result.Events {
		events[i] = convertDCBEvent(event)
	}

	var checkpointEventID string
	if len(events) > 0 {
		checkpointEventID = events[len(events)-1].Id
	}

	return &pb.ReadResponse{
		Events:                 events,
		CheckpointEventId:      checkpointEventID,
		DurationInMicroseconds: duration.Microseconds(),
		NumberOfMatchingEvents: int32(len(result.Events)),
	}, nil
}

func (s *server) Append(ctx context.Context, req *pb.AppendRequest) (*pb.AppendResponse, error) {
	log.Printf("[gRPC] Append request: %+v", req)
	start := time.Now()

	// Convert proto events to DCB events
	events := make([]dcb.InputEvent, len(req.Events))
	for i, event := range req.Events {
		events[i] = convertProtoInputEvent(event)
	}

	// Convert proto condition to DCB condition
	var condition *dcb.AppendCondition
	if req.Condition != nil {
		condition = convertProtoAppendCondition(req.Condition)
	}

	// Execute append
	_, err := s.store.Append(ctx, events, condition)
	if err != nil {
		log.Printf("[gRPC] Append error: %v", err)
		// Check if it's a concurrency error
		if _, ok := err.(*dcb.ConcurrencyError); ok {
			return &pb.AppendResponse{
				DurationInMicroseconds: time.Since(start).Microseconds(),
				AppendConditionFailed:  true,
			}, nil
		}
		return nil, err
	}

	return &pb.AppendResponse{
		DurationInMicroseconds: time.Since(start).Microseconds(),
		AppendConditionFailed:  false,
	}, nil
}

// Conversion functions
func convertProtoQuery(query *pb.Query) dcb.Query {
	if query == nil {
		return dcb.Query{Items: []dcb.QueryItem{}}
	}

	items := make([]dcb.QueryItem, len(query.Items))
	for i, item := range query.Items {
		items[i] = dcb.QueryItem{
			EventTypes: item.Types,
			Tags:       convertProtoTags(item.Tags),
		}
	}
	return dcb.Query{Items: items}
}

func convertProtoReadOptions(options *pb.ReadOptions) *dcb.ReadOptions {
	if options == nil {
		return nil
	}

	var fromPosition *int64
	if options.From != nil {
		if pos, err := parseEventID(*options.From); err == nil {
			fromPosition = &pos
		}
	}

	// Backwards and batch size are not used in DCB core, but can be added if needed
	return &dcb.ReadOptions{
		FromPosition: fromPosition,
		Limit:        nil,
		BatchSize:    nil,
	}
}

func convertProtoTags(tags []string) []dcb.Tag {
	result := make([]dcb.Tag, len(tags))
	for i, tag := range tags {
		// Parse "key:value" format
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			result[i] = dcb.Tag{
				Key:   parts[0],
				Value: parts[1],
			}
		} else {
			// Fallback: treat as key with empty value
			result[i] = dcb.Tag{
				Key:   tag,
				Value: "",
			}
		}
	}
	return result
}

func convertDCBEvent(event dcb.Event) *pb.Event {
	tags := make([]string, len(event.Tags))
	for i, tag := range event.Tags {
		if tag.Value != "" {
			tags[i] = fmt.Sprintf("%s:%s", tag.Key, tag.Value)
		} else {
			tags[i] = tag.Key
		}
	}

	return &pb.Event{
		Id:   fmt.Sprintf("%d", event.Position), // Use position as ID
		Type: event.Type,
		Tags: tags,
		Data: string(event.Data), // Data as JSON string
	}
}

func convertProtoInputEvent(event *pb.InputEvent) dcb.InputEvent {
	return dcb.InputEvent{
		Type: event.Type,
		Tags: convertProtoTags(event.Tags),
		Data: []byte(event.Data),
	}
}

func convertProtoAppendCondition(condition *pb.AppendCondition) *dcb.AppendCondition {
	if condition == nil {
		return nil
	}

	var failIfEventsMatch *dcb.Query
	if condition.FailIfEventsMatch != nil {
		query := convertProtoQuery(condition.FailIfEventsMatch)
		failIfEventsMatch = &query
	}

	var after *int64
	if condition.After != nil {
		if pos, err := parseEventID(*condition.After); err == nil {
			after = &pos
		}
	}

	return &dcb.AppendCondition{
		FailIfEventsMatch: failIfEventsMatch,
		After:             after,
	}
}

func parseEventID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

func main() {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@postgres:5432/dcb_app?sslmode=disable"
	}

	// Configure connection pool for performance
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}

	// Optimize connection pool for high throughput
	maxConns := 100
	minConns := 20

	if maxConnsEnv := os.Getenv("DB_MAX_CONNS"); maxConnsEnv != "" {
		if parsed, err := strconv.Atoi(maxConnsEnv); err == nil && parsed > 0 {
			maxConns = parsed
		}
	}

	if minConnsEnv := os.Getenv("DB_MIN_CONNS"); minConnsEnv != "" {
		if parsed, err := strconv.Atoi(minConnsEnv); err == nil && parsed > 0 && parsed <= maxConns {
			minConns = parsed
		}
	}

	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	config.MaxConnLifetime = 10 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 60 * time.Second

	// Connect to database with retry logic
	var pool *pgxpool.Pool
	maxRetries := 30
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to database (attempt %d/%d)...", i+1, maxRetries)

		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			log.Printf("Successfully connected to database")
			break
		}

		log.Printf("Failed to connect to database: %v", err)
		if i < maxRetries-1 {
			log.Printf("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(context.Background(), pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterEventStoreServiceServer(grpcServer, &server{store: store})

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "9091"
	}

	// Start listening
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Starting DCB gRPC server on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
