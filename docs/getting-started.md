# Getting Started with go-crablet

This guide will help you get started with go-crablet, a Go library **exploring** event sourcing concepts with Dynamic Consistency Boundary (DCB) patterns. 

**Note: This is an exploration project for learning and experimenting with DCB concepts, not a production-ready solution.**

## Quick Start

### 1. Installation

```bash
go get github.com/rodolfodpk/go-crablet/pkg/dcb
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BEST PRACTICE: Define event data as structs for type safety and performance
type UserRegisteredData struct {
    Name         string    `json:"name"`
    Email        string    `json:"email"`
    RegisteredAt time.Time `json:"registered_at"`
}

func main() {
    ctx := context.Background()
    
    // Create EventStore
    store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create events with struct-based data (RECOMMENDED)
    events := []dcb.InputEvent{
        dcb.NewEvent("UserRegistered").
            WithTag("user_id", "123").
            WithData(UserRegisteredData{
                Name:         "John Doe",
                Email:        "john@example.com",
                RegisteredAt: time.Now(),
            }).
            Build(),
    }
    
    // Append events
    err = store.Append(ctx, events)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("User registered successfully")
}
```

### 3. DCB Concurrency Control

```go
// Create condition to prevent conflicts using QueryBuilder
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("user_id", "123").
        WithType("UserRegistered").
        Build(),
)

// Append with condition (fails if user already exists)
err = store.AppendIf(ctx, events, condition)
if err != nil {
    if dcb.IsConcurrencyError(err) {
        log.Println("User already exists")
    } else {
        log.Fatal(err)
    }
}
```

### 4. Query Events

```go
// Query events by tags using QueryBuilder
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    Build()

events, err := store.Query(ctx, query, nil)
if err != nil {
    log.Fatal(err)
}

log.Printf("Found %d events for user 123", len(events))
```

### 5. Project State

```go
// BEST PRACTICE: Use typed constants for event types and typed structs for state projection
const (
	EventTypeUserRegistered = "UserRegistered"
	EventTypeCourseScheduled = "CourseScheduled"
	EventTypeStudentEnrolled = "StudentEnrolled"
)

type UserState struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CourseState struct {
	Title            string   `json:"title"`
	EnrolledStudents []string `json:"enrolled_students"`
}

// Define projectors with typed state
projectors := []dcb.StateProjector{
	{
		ID: "UserState",
		Query: dcb.NewQueryBuilder().
			WithTag("user_id", "123").
			Build(),
		InitialState: UserState{
			Name:  "",
			Email: "",
		},
		TransitionFn: func(state any, event dcb.Event) any {
			currentState := state.(UserState)
			
			switch event.GetEventType() {
			case EventTypeUserRegistered:
				var data UserRegisteredData
				if err := json.Unmarshal(event.GetData(), &data); err == nil {
					currentState.Name = data.Name
					currentState.Email = data.Email
				}
			}
			return currentState
		},
	},
	{
		ID: "CourseState",
		Query: dcb.NewQueryBuilder().
			WithTag("course_id", "CS101").
			Build(),
		InitialState: CourseState{
			Title:            "",
			EnrolledStudents: []string{},
		},
		TransitionFn: func(state any, event dcb.Event) any {
			currentState := state.(CourseState)
			
			switch event.GetEventType() {
			case EventTypeCourseScheduled:
				var data CourseScheduledData
				if err := json.Unmarshal(event.GetData(), &data); err == nil {
					currentState.Title = data.Title
				}
			case EventTypeStudentEnrolled:
				var data StudentEnrolledData
				if err := json.Unmarshal(event.GetData(), &data); err == nil {
					currentState.EnrolledStudents = append(currentState.EnrolledStudents, data.StudentID)
				}
			}
			return currentState
		},
	},
}

// Execute projection
finalState, _, err := store.Project(ctx, projectors, nil)
if err != nil {
	log.Fatal(err)
}

// Access typed state
userState := finalState["UserState"].(UserState)
courseState := finalState["CourseState"].(CourseState)

fmt.Printf("User: %s (%s)\n", userState.Name, userState.Email)
fmt.Printf("Course: %s with %d students\n", courseState.Title, len(courseState.EnrolledStudents))
```

## Command Execution

### 1. Create CommandExecutor

```go
commandExecutor := dcb.NewCommandExecutor(store)
```

### 2. Define Command Handler

```go
func handleRegisterUser(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    var data map[string]any
    json.Unmarshal(cmd.GetData(), &data)
    
    // Business logic validation
    if data["email"] == "" {
        return nil, errors.New("email required")
    }
    
    // Create event
    event := dcb.NewEvent("UserRegistered").
        WithTag("user_id", data["user_id"].(string)).
        WithData(data).
        Build()
    
    return []dcb.InputEvent{event}, nil
}
```

### 3. Execute Command

```go
// Create command
command := dcb.NewCommand("RegisterUser", dcb.ToJSON(map[string]any{
    "user_id": "123",
    "name": "John Doe",
    "email": "john@example.com",
}), nil)

// Execute command
events, err := commandExecutor.ExecuteCommand(ctx, command, handleRegisterUser, nil)
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### EventStore Configuration

```go
config := dcb.EventStoreConfig{
    MaxBatchSize:           1000,
    StreamBuffer:           1000,
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    QueryTimeout:           15000, // 15 seconds
    AppendTimeout:          10000, // 10 seconds
}

store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

### Connection Pool Configuration

```go
pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/crablet")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Configure pool
pool.Config().MaxConns = 20
pool.Config().MinConns = 5
```

## Examples

The `internal/examples/` directory contains complete, runnable examples:

- **`internal/examples/transfer/`** - Money transfer system with DCB concurrency control
- **`internal/examples/ticket_booking/`** - Concert ticket booking demonstrating DCB concurrency control
- **`internal/examples/decision_model/`** - Complex decision model with multiple projectors
- **`internal/examples/batch/`** - Batch event processing examples

### Running Examples

**Prerequisite: Database must be running!**

```bash
# 1. Start database (if not already running)
docker-compose up -d

# 2. Run any example
go run internal/examples/[example-name]/main.go

# 3. Or use Makefile targets
make example-transfer
make example-enrollment
make example-concurrency  # runs ticket_booking
make example-batch
make example-streaming
make example-decision
```

## Testing

### Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test package
go test -v ./pkg/dcb/tests/...
```

### Run Benchmarks

```bash
# Run Go library benchmarks
make benchmark-go

# Run enhanced business scenario benchmarks
make benchmark-go-enhanced

# Run all Go benchmarks (comprehensive)
make benchmark-go-all

# Run web app benchmarks
make benchmark-web-app

# Run all benchmarks
make benchmark-all
```

### Generate Benchmark Data

```bash
# Generate realistic benchmark data for fast access
make generate-benchmark-data

# Generate all data (datasets + benchmark data)
make generate-all-data

# Generate only test datasets
make generate-datasets
```

**🚀 New: Realistic Benchmark Scenarios**
- **Common batch sizes**: 1, 2, 3, 5, 8, 12 events (most real-world usage)
- **Runtime data generation**: Clean, simple benchmark execution
- **Real-world validation**: Performance reflects actual business patterns
- **~2,200 ops/sec**: Single events with 1.1-1.2ms latency

## Next Steps

1. **Read the Documentation**:
   - [Overview](./overview.md): Core concepts and architecture
   - [EventStore Flow](./eventstore-flow.md): Direct event operations
   - [Command Execution Flow](./command-execution-flow.md): High-level command pattern
   - [Examples](./examples.md): Complete usage examples

2. **Explore Examples**:
   - Start with `internal/examples/transfer/` for basic usage
   - Try `internal/examples/ticket_booking/` for DCB concurrency control
   - Check `internal/examples/decision_model/` for complex scenarios

3. **Run Benchmarks**:
   - Use `make benchmark-go` to test performance
   - Check `./benchmarks.md` for detailed results

4. **Production Setup**:
   - Configure connection pooling
   - Set up monitoring and alerting
   - Implement proper error handling
   - Consider backup and recovery strategies

## Troubleshooting

### Common Issues

1. **Database Connection**:
   ```bash
   # Check if PostgreSQL is running
   docker-compose ps
   
   # Check connection
   psql -h localhost -p 5432 -U postgres -d crablet
   ```

2. **Schema Issues**:
   ```bash
   # Recreate database
   docker-compose down
   docker-compose up -d
   ```

3. **Test Failures**:
   ```bash
   # Clean and rebuild
   go clean -cache
   go test ./...
   ```

### Getting Help

- **Issues**: Create an issue on GitHub
- **Discussions**: Use GitHub Discussions
- **Documentation**: Check the docs/ directory

This getting started guide provides the foundation for using go-crablet. Explore the examples and documentation to learn more about advanced features and best practices.
