# Simplified API

The `go-crablet` library provides a simplified API for common event sourcing patterns, making it easier to work with events, queries, and projections.

## Table of Contents

1. [QueryBuilder](#querybuilder)
2. [Simplified AppendCondition Constructors](#simplified-appendcondition-constructors)
3. [Projection Helpers](#projection-helpers)
4. [Simplified Tags](#simplified-tags)
5. [EventBuilder](#eventbuilder)
6. [BatchBuilder](#batchbuilder)
7. [Convenience Functions](#convenience-functions)
8. [Complete Example](#complete-example)
9. [Migration Guide](#migration-guide)

## QueryBuilder

The `QueryBuilder` provides a fluent interface for constructing complex queries with proper DCB OR/AND semantics.

### Basic Usage

```go
// Simple query with single item
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserRegistered").
    Build()

// Query with OR conditions (multiple items)
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserRegistered").
    AddItem().
    WithTag("user_id", "456").
    WithType("UserProfileUpdated").
    Build()
```

### Methods

- `WithTag(key, value string)` - Add a tag condition to current item
- `WithTags(kv ...string)` - Add multiple tag conditions to current item
- `WithType(eventType string)` - Add an event type to current item
- `WithTypes(eventTypes ...string)` - Add multiple event types to current item
- `WithTagAndType(key, value, eventType string)` - Add both tag and type to current item
- `WithTagsAndTypes(eventTypes []string, kv ...string)` - Add multiple tags and types to current item
- `AddItem()` - Start a new query item (OR condition)
- `Build()` - Build the final Query

## Simplified AppendCondition Constructors

Quick constructors for common append conditions:

```go
// Fail if any event with the given tag exists
condition := dcb.FailIfExists("user_id", "123")

// Fail if any event with the given type and tag exists
condition := dcb.FailIfEventType("UserRegistered", "user_id", "123")

// Fail if any event with the given types and tag exists
condition := dcb.FailIfEventTypes([]string{"UserRegistered", "UserProfileUpdated"}, "user_id", "123")
```

## Projection Helpers

Simplified constructors for common projection patterns:

```go
// Counter projection
counterProjector := dcb.ProjectCounter("user_actions", "UserAction", "user_id", "123")

// Boolean projection
booleanProjector := dcb.ProjectBoolean("user_active", "UserLogin", "user_id", "123")

// State projection
stateProjector := dcb.ProjectState("user_profile", "UserProfileUpdated", 
    "user_id", "123", 
    UserProfile{}, 
    func(state any, event dcb.Event) any {
        // Transition logic
        return state
    })

// State projection with multiple event types
stateProjector := dcb.ProjectStateWithTypes("user_profile", 
    []string{"UserRegistered", "UserProfileUpdated"}, 
    "user_id", "123", 
    UserProfile{}, 
    func(state any, event dcb.Event) any {
        // Transition logic
        return state
    })

// State projection with multiple tags
stateProjector := dcb.ProjectStateWithTags("user_profile", 
    "UserProfileUpdated", 
    dcb.Tags{"user_id": "123", "tenant": "acme"}, 
    UserProfile{}, 
    func(state any, event dcb.Event) any {
        // Transition logic
        return state
    })
```

## Simplified Tags

Map-based tag construction:

```go
// Create tags from map
tags := dcb.Tags{
    "user_id": "123",
    "tenant":  "acme",
    "version": "1.0",
}

// Convert to []Tag
tagSlice := tags.ToTags()
```

## EventBuilder

Fluent interface for building events:

```go
event := dcb.NewEvent("UserRegistered").
    WithTag("user_id", "123").
    WithTag("email", "user@example.com").
    WithTags(map[string]string{
        "tenant": "acme",
        "version": "1.0",
    }).
    WithData(userData).
    Build()
```

### Methods

- `WithTag(key, value string)` - Add a single tag
- `WithTags(tags map[string]string)` - Add multiple tags
- `WithData(data any)` - Set event data
- `Build()` - Build the final InputEvent

## BatchBuilder

Fluent interface for building event batches:

```go
batch := dcb.NewBatch().
    AddEvent(event1).
    AddEvent(event2).
    AddEvents(event3, event4).
    AddEventFromBuilder(eventBuilder).
    Build()
```

### Methods

- `AddEvent(event InputEvent)` - Add a single event
- `AddEvents(events ...InputEvent)` - Add multiple events
- `AddEventFromBuilder(builder *EventBuilder)` - Add event from builder
- `Build()` - Build the final []InputEvent

## Convenience Functions

Simple functions for common operations:

```go
// Append single event
err := dcb.AppendSingleEvent(ctx, store, "UserLogin", map[string]string{
    "user_id": "123",
    "ip": "192.168.1.1",
}, map[string]string{
    "login_time": time.Now().Format(time.RFC3339),
})

// Append single event with condition
err := dcb.AppendSingleEventIf(ctx, store, "UserProfileUpdated", map[string]string{
    "user_id": "123",
}, userData, condition)

// Append batch from structs
err := dcb.AppendBatchFromStructs(ctx, store,
    struct {
        Type string
        Tags map[string]string
        Data any
    }{
        Type: "UserAction",
        Tags: map[string]string{"user_id": "123", "action": "view_profile"},
        Data: map[string]string{"timestamp": time.Now().Format(time.RFC3339)},
    },
    struct {
        Type string
        Tags map[string]string
        Data any
    }{
        Type: "UserAction",
        Tags: map[string]string{"user_id": "123", "action": "edit_settings"},
        Data: map[string]string{"timestamp": time.Now().Format(time.RFC3339)},
    },
)

// Append batch from structs with condition
err := dcb.AppendBatchFromStructsIf(ctx, store, condition, events...)
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

type UserProfile struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Name     string `json:"name"`
    IsActive bool   `json:"is_active"`
}

func main() {
    ctx := context.Background()
    
    // Create event store
    store, err := dcb.NewEventStore(ctx, pool)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create user registration event
    userEvent := UserProfile{
        UserID: "user_123",
        Email:  "user@example.com",
        Name:   "John Doe",
    }
    
    event := dcb.NewEvent("UserRegistered").
        WithTag("user_id", userEvent.UserID).
        WithTag("email", userEvent.Email).
        WithData(userEvent).
        Build()
    
    // Create append condition (fail if user already exists)
    condition := dcb.FailIfExists("user_id", userEvent.UserID)
    
    // Append with condition
    err = store.AppendIf(ctx, []dcb.InputEvent{event}, condition)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create query to find user events
    query := dcb.NewQueryBuilder().
        WithTag("user_id", userEvent.UserID).
        WithTypes("UserRegistered", "UserProfileUpdated").
        Build()
    
    // Query events
    events, err := store.Query(ctx, query, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create projection for user profile
    projector := dcb.ProjectState("user_profile", "UserProfileUpdated", 
        "user_id", userEvent.UserID, 
        UserProfile{}, 
        func(state any, event dcb.Event) any {
            profile := state.(UserProfile)
            // Update profile based on event
            return profile
        })
    
    // Project state
    result, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User profile: %+v", result[0])
}
```

## Migration Guide

### From Old API to New API

#### Old Query Construction
```go
// Old way
query := dcb.NewQuery([]dcb.Tag{dcb.NewTag("user_id", "123")}, "UserRegistered")
```

#### New Query Construction
```go
// New way
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserRegistered").
    Build()
```

#### Old Tag Construction
```go
// Old way
tags := []dcb.Tag{dcb.NewTag("user_id", "123"), dcb.NewTag("email", "user@example.com")}
```

#### New Tag Construction
```go
// New way
tags := dcb.Tags{
    "user_id": "123",
    "email":   "user@example.com",
}.ToTags()
```

#### Old Event Construction
```go
// Old way
event := dcb.NewInputEvent("UserRegistered", tags, data)
```

#### New Event Construction
```go
// New way
event := dcb.NewEvent("UserRegistered").
    WithTags(map[string]string{
        "user_id": "123",
        "email":   "user@example.com",
    }).
    WithData(data).
    Build()
```

#### Old AppendCondition Construction
```go
// Old way
condition := dcb.NewAppendCondition(dcb.NewQuery([]dcb.Tag{dcb.NewTag("user_id", "123")}, "UserRegistered"))
```

#### New AppendCondition Construction
```go
// New way
condition := dcb.FailIfExists("user_id", "123")
// or
condition := dcb.FailIfEventType("UserRegistered", "user_id", "123")
```

The new API is more intuitive, provides better type safety, and follows Go conventions more closely while maintaining full DCB compliance. 