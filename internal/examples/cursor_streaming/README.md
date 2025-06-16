# Cursor-Based Streaming Example

This example demonstrates true PostgreSQL cursor-based streaming in go-crablet.

## Features Demonstrated

1. **True Streaming**: Uses PostgreSQL server-side cursors to process large datasets without loading everything into memory
2. **Batch Processing**: Processes events in configurable batches (default 1000, configurable via `BatchSize`)
3. **ProjectDecisionModel with Cursors**: Shows how to use cursor-based streaming for decision model projection
4. **ReadStream with Cursors**: Demonstrates direct event streaming with cursors

## How It Works

### Cursor-Based Streaming vs Regular Query

**Regular Query (without cursors):**
```go
// Loads ALL events into memory at once
rows, err := pool.Query(ctx, sqlQuery, args...)
for rows.Next() {
    // Process each row
}
```

**Cursor-Based Streaming:**
```go
// Declares a server-side cursor
tx.Exec(ctx, "DECLARE cursor_name CURSOR FOR " + sqlQuery)

// Fetches events in batches
for {
    rows, err := tx.Query(ctx, "FETCH 100 FROM cursor_name")
    // Process batch
    if rows.RowsAffected() < 100 {
        break // No more data
    }
}
```

### Benefits

1. **Memory Efficiency**: Only batch size in memory at any time
2. **Scalability**: Can handle millions of events
3. **Real-time Processing**: Start processing immediately without waiting for all data
4. **Consistent Snapshots**: Maintains consistent view throughout processing

### Usage

```go
// Enable cursor-based streaming
batchSize := 100
options := &dcb.ReadOptions{BatchSize: &batchSize}

// Use with ProjectDecisionModel
states, appendCondition, err := store.ProjectDecisionModel(ctx, query, options, projectors)

// Or use with ReadStream directly
iterator, err := store.ReadStream(ctx, query, options)
defer iterator.Close()

for iterator.Next() {
    event := iterator.Event()
    // Process event
}
```

## Running the Example

1. Set up PostgreSQL database
2. Set `DATABASE_URL` environment variable
3. Run the example:

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/dbname"
go run main.go
```

## Output

The example will:
1. Create 1000 test events
2. Demonstrate cursor-based streaming with ProjectDecisionModel
3. Show direct event streaming with ReadStream
4. Display memory-efficient processing of large datasets

## Configuration

- **BatchSize**: Controls how many events are fetched per batch (default: 1000)
- **Cursor Name**: Automatically generated unique names to avoid conflicts
- **Transaction Management**: Automatic cursor cleanup and transaction rollback

## 📄 **License**

This example is licensed under the Apache License 2.0 - see the [LICENSE](../../../LICENSE) file for details. 