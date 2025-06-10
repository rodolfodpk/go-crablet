# Appending Events

This guide explains how to append events to the event store using go-crablet, including best practices for handling concurrency, validation, and consistency.

## Basic Event Appending

The simplest way to append events is using the `AppendEvents` method:

```go
// Create tags for the event
tags := dcb.NewTags(
    "account_id", "acc123",
    "user_id", "user456",
)

// Create a new event
event := dcb.NewInputEvent(
    "AccountBalanceUpdated", 
    tags, 
    []byte(`{"balance": 1000, "currency": "USD"}`),
)

// Define the consistency boundary
query := dcb.NewQuery(tags, "AccountBalanceUpdated")

// Get current stream position
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

// Append the event using the current position
newPosition, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, query, position)
if err != nil {
    return err
}
```

## Stream Position and Concurrency Control

go-crablet uses optimistic concurrency control to ensure consistency. This means:

1. Always get the current stream position before appending events
2. Use that position when appending
3. Handle concurrency errors appropriately

```go
// Get current stream position
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

// Append events using the current position
newPosition, err := store.AppendEvents(ctx, events, query, position)
if err != nil {
    if _, ok := err.(*dcb.ConcurrencyError); ok {
        // Handle concurrent modification
        // You might want to retry with a new position
        return fmt.Errorf("concurrent modification detected: %w", err)
    }
    return err
}
```

## Batch Operations

You can append multiple events in a single operation using the `NewEventBatch` helper. This is useful for maintaining consistency across related events:

```go
// Create multiple events using NewEventBatch helper
events := dcb.NewEventBatch(
    dcb.NewInputEvent(
        "OrderPlaced",
        dcb.NewTags("order_id", "order123"),
        []byte(`{"amount": 100, "currency": "USD"}`),
    ),
    dcb.NewInputEvent(
        "PaymentProcessed",
        dcb.NewTags("order_id", "order123"),
        []byte(`{"amount": 100, "payment_id": "pay123"}`),
    ),
)

// Define a query that includes all relevant event types
query := dcb.NewQuery(
    dcb.NewTags("order_id", "order123"),
    "OrderPlaced",
    "PaymentProcessed",
)

// Get current position and append
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

newPosition, err := store.AppendEvents(ctx, events, query, position)
```

## Event Validation

go-crablet performs several validations when appending events:

1. Event type must not be empty
2. Event data must be valid JSON
3. Tags must be properly formatted

```go
// This will fail validation - empty event type
event := dcb.NewInputEvent("", tags, []byte(`{"foo":"bar"}`))

// This will fail validation - invalid JSON
event := dcb.NewInputEvent("OrderPlaced", tags, []byte(`invalid json`))

// This will pass validation
event := dcb.NewInputEvent(
    "OrderPlaced",
    dcb.NewTags("order_id", "order123"),
    []byte(`{"amount": 100}`),
)
```

## Consistency Boundaries

The `Query` parameter in `AppendEvents` defines the consistency boundary for your events. This ensures that:

1. Events within the same boundary are processed atomically
2. Concurrent modifications to the same boundary are detected
3. Event ordering is maintained within the boundary

```go
// Define a consistency boundary for an order
query := dcb.NewQuery(
    dcb.NewTags("order_id", "order123"),
    "OrderPlaced",
    "OrderStatusChanged",
)

// All events in this boundary will be processed together
events := []dcb.InputEvent{
    dcb.NewInputEvent("OrderPlaced", tags, orderData),
    dcb.NewInputEvent("OrderStatusChanged", tags, updateData),
}

// If another process tries to modify the same order concurrently,
// one of the operations will fail with a concurrency error
newPosition, err := store.AppendEvents(ctx, events, query, position)
```

## Best Practices

1. **Always Use Current Position**
   ```go
   // ❌ Don't use a fixed position
   store.AppendEvents(ctx, events, query, 0)
   
   // ✅ Always get the current position first
   position, err := store.GetCurrentPosition(ctx, query)
   store.AppendEvents(ctx, events, query, position)
   ```

2. **Define Clear Consistency Boundaries**
   ```go
   // ❌ Too broad - might cause unnecessary conflicts
   query := dcb.NewQuery(nil, "OrderPlaced")
   
   // ✅ Specific to the entity
   query := dcb.NewQuery(
       dcb.NewTags("order_id", "order123"),
       "OrderPlaced",
       "OrderStatusChanged",
   )
   ```

3. **Handle Concurrency Errors**
   ```go
   // ❌ Ignoring concurrency errors
   newPosition, err := store.AppendEvents(ctx, events, query, position)
   if err != nil {
       return err
   }
   
   // ✅ Proper error handling with retry logic
   newPosition, err := store.AppendEvents(ctx, events, query, position)
   if err != nil {
       if _, ok := err.(*dcb.ConcurrencyError); ok {
           // Implement retry logic or notify user
           return fmt.Errorf("concurrent modification: %w", err)
       }
       return err
   }
   ```

4. **Validate Event Data Before Appending**
   ```go
   // ❌ Relying only on go-crablet validation
   event := dcb.NewInputEvent("OrderPlaced", tags, []byte(`{"amount": "invalid"}`))
   
   // ✅ Validate data before appending
   type OrderData struct {
       Amount float64 `json:"amount"`
   }
   data := OrderData{Amount: 100}
   jsonData, err := json.Marshal(data)
   if err != nil {
       return err
   }
   event := dcb.NewInputEvent("OrderPlaced", tags, jsonData)
   ```

5. **Use Batch Operations for Related Events**
   ```go
   // ❌ Appending related events separately
   store.AppendEvents(ctx, []dcb.InputEvent{orderPlaced}, query, pos1)
   store.AppendEvents(ctx, []dcb.InputEvent{paymentProcessed}, query, pos2)
   
   // ✅ Appending related events in a batch using NewEventBatch
   events := dcb.NewEventBatch(orderPlaced, paymentProcessed)
   store.AppendEvents(ctx, events, query, position)
   ```

## Related Documentation

- [State Projection](docs/state-projection.md): Learn how to project state from events
- [Course Subscription Example](docs/course-subscription.md): See a complete example of event appending in a real application
- [Examples](docs/examples.md): More examples of using go-crablet 

## Example event types

"AccountBalanceUpdated",
"DepositMade",
"WithdrawalProcessed",
"OrderPlaced",
"OrderStatusChanged",
"PaymentProcessed", 