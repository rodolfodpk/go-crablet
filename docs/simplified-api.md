# Simplified API Guide

This document describes the new simplified API constructs that provide a better developer experience with 50% less boilerplate for common operations. The API is **fully DCB compliant** according to the [DCB specification](https://dcb.events/specification/).

## Overview

The simplified API introduces several new constructs that make common operations more intuitive and reduce boilerplate code:

- **QueryBuilder**: Fluent interface for building DCB-compliant queries
- **Simplified AppendCondition**: Direct constructors for common conditions
- **Projection Helpers**: Pre-built projectors for common patterns
- **Simplified Tags**: Map-based tag construction

## QueryBuilder Pattern (DCB Compliant)

The QueryBuilder provides a fluent interface for constructing queries that are **fully compliant with the DCB specification**. It properly implements OR/AND semantics:

- **QueryItems are combined with OR** (as per DCB specification)
- **Conditions within a QueryItem are combined with AND**
- **Supports complex query patterns** with multiple event types and tags

### DCB Compliance

The QueryBuilder follows the [DCB specification](https://dcb.events/specification/) which states:

> All Query Items are effectively combined with an **OR**, e.g. adding an extra Query Item will likely result in more Events being returned

### Basic Usage

```go
// Old way - verbose and error-prone
query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserRegistered")

// New way - fluent and DCB compliant
query := dcb.NewQueryBuilder().WithTagAndType("user_id", "123", "UserRegistered").Build()
```

### DCB OR/AND Semantics

#### Single QueryItem (AND conditions)
```go
// This creates a single QueryItem with AND conditions
query := dcb.NewQueryBuilder().
    WithTypes("EventA", "EventB").                    // OR between event types
    WithTags("key1", "value1", "key2", "value2").     // AND with event types
    Build()

// Matches: (EventA OR EventB) AND (key1=value1 AND key2=value2)
```

#### Multiple QueryItems (OR conditions)
```go
// This creates multiple QueryItems combined with OR
query := dcb.NewQueryBuilder().
    AddItem().WithType("EventA").WithTag("key1", "value1").
    AddItem().WithType("EventB").WithTag("key2", "value2").
    Build()

// Matches: (EventA AND key1=value1) OR (EventB AND key2=value2)
```

### Available Methods

#### `WithTag(key, value string)`
Adds a single tag condition to the current QueryItem (AND).

```go
query := dcb.NewQueryBuilder().WithTag("user_id", "123").Build()
```

#### `WithTags(kv ...string)`
Adds multiple tag conditions to the current QueryItem (AND).

```go
query := dcb.NewQueryBuilder().WithTags("user_id", "123", "status", "active").Build()
```

#### `WithType(eventType string)`
Adds a single event type condition to the current QueryItem (OR with existing types).

```go
query := dcb.NewQueryBuilder().WithType("UserRegistered").Build()
```

#### `WithTypes(eventTypes ...string)`
Adds multiple event type conditions to the current QueryItem (OR with existing types).

```go
query := dcb.NewQueryBuilder().WithTypes("UserRegistered", "UserProfileUpdated").Build()
```

#### `WithTagAndType(key, value, eventType string)`
Adds both tag and event type conditions to the current QueryItem.

```go
query := dcb.NewQueryBuilder().WithTagAndType("user_id", "123", "UserRegistered").Build()
```

#### `WithTagsAndTypes(eventTypes []string, kv ...string)`
Adds both tags and event types conditions to the current QueryItem.

```go
query := dcb.NewQueryBuilder().WithTagsAndTypes(
    []string{"UserRegistered", "UserProfileUpdated"}, 
    "user_id", "123", "status", "active",
).Build()
```

#### `AddItem()`
Starts a new QueryItem for OR conditions. This creates a new QueryItem that will be combined with OR.

```go
query := dcb.NewQueryBuilder().
    AddItem().WithType("EventA").WithTag("key1", "value1").
    AddItem().WithType("EventB").WithTag("key2", "value2").
    Build()
```

### Complex DCB Patterns

The QueryBuilder supports complex patterns that match the DCB specification example:

```go
// DCB specification example:
// Matches Events that are either:
// - of type EventType1 OR EventType2
// - tagged tag1 AND tag2  
// - of type EventType2 OR EventType3 AND tagged tag1 AND tag3

query := dcb.NewQueryBuilder().
    AddItem().WithTypes("EventType1", "EventType2").
    AddItem().WithTags("tag1", "tag2").
    AddItem().WithTypes("EventType2", "EventType3").WithTags("tag1", "tag3").
    Build()
```

## Simplified AppendCondition

The simplified AppendCondition constructors eliminate the 3-step process for common conditions.

### Available Constructors

#### `FailIfExists(key, value string)`
Creates a condition that fails if any events match the given tag.

```go
// Old way - 3-step process
item := dcb.NewQueryItem([]string{"UserRegistered"}, []dcb.Tag{dcb.NewTag("user_id", "123")})
query := dcb.NewQueryFromItems(item)
condition := dcb.NewAppendCondition(query)

// New way - direct constructor
condition := dcb.FailIfExists("user_id", "123")
```

#### `FailIfEventType(eventType, key, value string)`
Creates a condition that fails if events of the given type exist with the specified tag.

```go
condition := dcb.FailIfEventType("UserRegistered", "user_id", "123")
```

#### `FailIfEventTypes(eventTypes []string, key, value string)`
Creates a condition that fails if events of any of the given types exist with the specified tag.

```go
condition := dcb.FailIfEventTypes([]string{"UserRegistered", "UserProfileUpdated"}, "user_id", "123")
```

## Projection Helpers

Projection helpers provide pre-built projectors for common patterns, eliminating boilerplate code.

### Available Helpers

#### `ProjectCounter(id, eventType, key, value string)`
Creates a projector that counts events.

```go
// Old way - manual setup
projector := dcb.StateProjector{
    ID: "user_count",
    Query: dcb.NewQuery(dcb.NewTags("status", "active"), "UserRegistered"),
    InitialState: 0,
    TransitionFn: func(state any, event dcb.Event) any {
        return state.(int) + 1
    },
}

// New way - helper function
projector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
```

#### `ProjectBoolean(id, eventType, key, value string)`
Creates a projector that tracks if events exist.

```go
projector := dcb.ProjectBoolean("user_exists", "UserRegistered", "user_id", "123")
```

#### `ProjectState(id, eventType, key, value string, initialState any, transitionFn func(any, dcb.Event) any)`
Creates a projector with custom initial state and transition function.

```go
projector := dcb.ProjectState("user", "UserRegistered", "user_id", "123", UserState{}, func(state any, event dcb.Event) any {
    // Custom transition logic
    return state
})
```

#### `ProjectStateWithTypes(id string, eventTypes []string, key, value string, initialState any, transitionFn func(any, dcb.Event) any)`
Creates a projector for multiple event types.

```go
projector := dcb.ProjectStateWithTypes("user", []string{"UserRegistered", "UserProfileUpdated"}, "user_id", "123", UserState{}, transitionFn)
```

#### `ProjectStateWithTags(id string, eventType string, tags Tags, initialState any, transitionFn func(any, dcb.Event) any)`
Creates a projector with multiple tag conditions.

```go
projector := dcb.ProjectStateWithTags("user", "UserRegistered", dcb.Tags{"user_id": "123", "status": "active"}, UserState{}, transitionFn)
```

## Simplified Tags

The `Tags` type provides a map-based approach to tag construction, making it more readable and less error-prone.

### Usage

```go
// Old way - awkward with many tags
tags := dcb.NewTags("user_id", "123", "email", "user@example.com", "status", "active")

// New way - map-based and readable
tags := dcb.Tags{
    "user_id": "123",
    "email":   "user@example.com", 
    "status":  "active",
}.ToTags()
```

### Creating Events with Simplified Tags

```go
event := dcb.NewInputEvent("UserRegistered", dcb.Tags{
    "user_id": "123",
    "email":   "user@example.com",
}.ToTags(), dcb.ToJSON(userData))
```

## Complete Example

Here's a complete example showing how the simplified API reduces boilerplate and supports complex DCB patterns:

```go
// Create a user with simplified API
userEvent := UserRegistered{
    UserID:    "user_123",
    Email:     "user@example.com",
    Username:  "johndoe",
    CreatedAt: time.Now(),
}

// Use simplified tags
event := dcb.NewInputEvent("UserRegistered", dcb.Tags{
    "user_id": "user_123",
    "email":   "user@example.com",
}.ToTags(), dcb.ToJSON(userEvent))

// Append without condition
err := store.Append(ctx, []dcb.InputEvent{event})

// Query with simplified query
query := dcb.NewQueryBuilder().WithTagAndType("user_id", "user_123", "UserRegistered").Build()
events, err := store.Query(ctx, query, nil)

// Complex query with multiple event types and tags
complexQuery := dcb.NewQueryBuilder().
    WithTypes("UserRegistered", "UserProfileUpdated").
    WithTags("user_id", "user_123", "status", "active").
    Build()

// OR conditions with multiple QueryItems
orQuery := dcb.NewQueryBuilder().
    AddItem().WithType("UserRegistered").WithTag("user_id", "user_123").
    AddItem().WithType("UserProfileUpdated").WithTag("status", "active").
    Build()

// Update with DCB concurrency control
userProjector := dcb.ProjectState("user", "UserRegistered", "user_id", "user_123", UserState{}, transitionFn)
projectedStates, appendCondition, err := store.Project(ctx, []dcb.StateProjector{userProjector}, nil)

updateEvent := dcb.NewInputEvent("UserProfileUpdated", dcb.Tags{
    "user_id": "user_123",
}.ToTags(), dcb.ToJSON(profileUpdate))

err = store.AppendIf(ctx, []dcb.InputEvent{updateEvent}, appendCondition)
```

## Benefits

The simplified API provides several key benefits:

1. **50% less boilerplate** for common operations
2. **More intuitive query construction** with fluent interfaces
3. **Fewer errors** with type-safe helpers
4. **Better readability** with map-based tag construction
5. **Clearer intent** with descriptive method names
6. **Full DCB compliance** according to the specification
7. **Support for complex patterns** with proper OR/AND semantics
8. **Backward compatibility** - all existing code continues to work

## Migration Guide

The simplified API is **additive** - all existing code continues to work unchanged. You can gradually migrate to the new constructs:

1. **Start with QueryBuilder** for new queries
2. **Use simplified AppendCondition** constructors for new conditions
3. **Adopt projection helpers** for common patterns
4. **Switch to Tags type** for better readability
5. **Leverage complex DCB patterns** for advanced use cases

## Demo

Run the API demo to see all features in action:

```bash
go run ./internal/examples/api_demo
```

This demonstrates all the simplified API constructs with real examples and shows the reduction in boilerplate code, including complex DCB patterns. 