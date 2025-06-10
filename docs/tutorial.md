# Getting Started with go-crablet

This tutorial will guide you through the basics of using go-crablet as a library in your Go application. We'll create a simple todo list application to demonstrate the core concepts.

## Prerequisites

- Go 1.24 or later
- PostgreSQL 12 or later
- Basic understanding of Go and event sourcing concepts

## Step 1: Installation

First, add go-crablet to your project:

```bash
go get github.com/rodolfodpk/go-crablet
```

## Step 2: Database Setup

Create a PostgreSQL database and set up the required schema:

```sql
-- Create the database
CREATE DATABASE todo_app;

-- Connect to the database
\c todo_app

-- Create the events table
CREATE TABLE events (
    id UUID PRIMARY KEY,
    type TEXT NOT NULL,
    tags JSONB NOT NULL,
    data JSONB NOT NULL,
    position BIGSERIAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    causation_id UUID NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED,
    correlation_id UUID NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED
);

-- Create indexes for efficient querying
CREATE INDEX idx_events_position ON events (position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);
CREATE INDEX idx_events_causation_id ON events (causation_id);
CREATE INDEX idx_events_correlation_id ON events (correlation_id);
```

## Step 3: Create a Simple Todo Application

Let's create a simple todo application that demonstrates the core features of go-crablet. Create a new file `main.go`:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet"
)

// TodoState represents the current state of a todo
type TodoState struct {
    ID          string
    Title       string
    Completed   bool
    CreatedAt   string
}

// TodoAdded event data
type TodoAdded struct {
    Title     string `json:"title"`
    CreatedAt string `json:"created_at"`
}

// TodoCompleted event data
type TodoCompleted struct {
    CompletedAt string `json:"completed_at"`
}

func main() {
    // Create a context that will be used throughout the application
    ctx := context.Background()

    // Get database URL from environment variable
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        dbURL = "postgres://postgres:postgres@localhost:5432/todo_app?sslmode=disable"
    }

    // Create a PostgreSQL connection pool
    pool, err := pgxpool.New(ctx, dbURL)
    if err != nil {
        log.Fatalf("Unable to connect to database: %v", err)
    }
    defer pool.Close()

    // Create a new event store
    store, err := dcb.NewEventStore(ctx, pool)
    if err != nil {
        log.Fatalf("Unable to create event store: %v", err)
    }

    // Create a new todo
    todoID := "todo-1"
    todoTags := dcb.NewTags("todo_id", todoID)
    
    // Create the todo creation event
    todoAdded := TodoAdded{
        Title:     "Learn go-crablet",
        CreatedAt: "2024-03-20T10:00:00Z",
    }

    // Marshal the event data to JSON
    data, err := json.Marshal(todoAdded)
    if err != nil {
        log.Fatalf("Unable to marshal event data: %v", err)
    }

    // Create the event using NewInputEvent with pre-marshaled JSON data
    event := dcb.NewInputEvent(
        "TodoAdded",
        todoTags,
        data, // Pre-marshaled JSON data
    )

    // Get current stream position
    query := dcb.NewQuery(todoTags)
    position, err := store.GetCurrentPosition(ctx, query)
    if err != nil {
        log.Fatalf("Unable to get current position: %v", err)
    }

    // Append the event
    newPosition, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, query, position)
    if err != nil {
        log.Fatalf("Unable to append event: %v", err)
    }
    fmt.Printf("Todo created at position %d\n", newPosition)

    // Create a projector for todo state
    todoProjector := dcb.StateProjector{
        Query: query,
        InitialState: &TodoState{
            ID: todoID,
        },
        TransitionFn: func(state any, event dcb.Event) any {
            todo := state.(*TodoState)
            switch event.Type {
            case "TodoAdded":
                var data TodoAdded
                if err := json.Unmarshal(event.Data, &data); err != nil {
                    panic(err)
                }
                todo.Title = data.Title
                todo.CreatedAt = data.CreatedAt
                todo.Completed = false
            case "TodoCompleted":
                var data TodoCompleted
                if err := json.Unmarshal(event.Data, &data); err != nil {
                    panic(err)
                }
                todo.Completed = true
            }
            return todo
        },
    }

    // Project the current state
    _, todoState, err := store.ProjectState(ctx, todoProjector)
    if err != nil {
        log.Fatalf("Unable to project state: %v", err)
    }

    // Print the todo state
    todo := todoState.(*TodoState)
    fmt.Printf("Todo: %s\n", todo.Title)
    fmt.Printf("Created at: %s\n", todo.CreatedAt)
    fmt.Printf("Completed: %v\n", todo.Completed)

    // Mark the todo as completed
    todoCompleted := TodoCompleted{
        CompletedAt: "2024-03-20T11:00:00Z",
    }

    // Marshal the event data to JSON
    data, err = json.Marshal(todoCompleted)
    if err != nil {
        log.Fatalf("Unable to marshal event data: %v", err)
    }

    // Create the completion event with pre-marshaled JSON data
    completionEvent := dcb.NewInputEvent(
        "TodoCompleted",
        todoTags,
        data, // Pre-marshaled JSON data
    )

    // Get updated position
    position, err = store.GetCurrentPosition(ctx, query)
    if err != nil {
        log.Fatalf("Unable to get current position: %v", err)
    }

    // Append the completion event
    newPosition, err = store.AppendEvents(ctx, []dcb.InputEvent{completionEvent}, query, position)
    if err != nil {
        log.Fatalf("Unable to append event: %v", err)
    }
    fmt.Printf("Todo completed at position %d\n", newPosition)

    // Project the final state
    _, todoState, err = store.ProjectState(ctx, todoProjector)
    if err != nil {
        log.Fatalf("Unable to project state: %v", err)
    }

    // Print the final todo state
    todo = todoState.(*TodoState)
    fmt.Printf("\nFinal Todo State:\n")
    fmt.Printf("Todo: %s\n", todo.Title)
    fmt.Printf("Created at: %s\n", todo.CreatedAt)
    fmt.Printf("Completed: %v\n", todo.Completed)
}
```

## Step 4: Run the Application

1. Make sure PostgreSQL is running and the database is created
2. Set the DATABASE_URL environment variable if needed
3. Run the application:

```bash
go run main.go
```

You should see output similar to:

```
Todo created at position 1
Todo: Learn go-crablet
Created at: 2024-03-20T10:00:00Z
Completed: false
Todo completed at position 2

Final Todo State:
Todo: Learn go-crablet
Created at: 2024-03-20T10:00:00Z
Completed: true
```

## What We've Learned

This tutorial demonstrated several key concepts of go-crablet:

1. **Event Store Setup**: Creating a connection to PostgreSQL and initializing the event store
2. **Event Creation**: Defining and creating events with proper tags and data
3. **Stream Position Handling**: Using current stream positions for optimistic concurrency control
4. **State Projection**: Creating a projector to build the current state from events
5. **Event Appending**: Appending events to the store with proper position handling and UUID-based IDs

## Next Steps

Now that you understand the basics, you can:

1. Add more event types to your todo application
2. Implement querying and filtering of todos
3. Add more complex state projections
4. Implement error handling and retry logic
5. Add validation and business rules

Check out the other documentation files for more advanced features and examples:

- [Overview](overview.md): High-level overview of go-crablet
- [State Projection](state-projection.md): Detailed guide on state projection
- [Examples](examples.md): More complex examples and use cases 