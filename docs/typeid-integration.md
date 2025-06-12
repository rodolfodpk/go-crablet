# TypeID Integration

go-crablet now includes **TypeID** integration for enhanced debugging and traceability. TypeIDs are type-safe, K-sortable unique identifiers that provide meaningful prefixes based on event tags.

## Overview

TypeIDs combine the benefits of UUIDs (uniqueness, K-sortability) with human-readable prefixes that make event identification and debugging much easier. Instead of random UUIDs like `550e8400-e29b-41d4-a716-446655440000`, you get meaningful IDs like `course_id_01jxfvsth3ezwvxjec1xp4ejvb`.

## Benefits

- **🎯 Better Debugging**: Event IDs include entity information (e.g., `course_id_01jxfvsth3ezwvxjec1xp4ejvb`)
- **🔍 Easier Tracing**: See event relationships and entity context at a glance
- **📊 Improved Monitoring**: Logs are more readable and informative
- **🔄 K-Sortable**: Maintains chronological ordering for efficient queries

## How It Works

TypeIDs are automatically generated from event tags:

```go
// Single tag event
event := dcb.InputEvent{
    Type: "CourseLaunched",
    Tags: []dcb.Tag{{Key: "course_id", Value: "course1"}},
    Data: []byte(`{"title": "Go Programming"}`),
}
// Generates: course_id_01jxfvsth3ezwvxjec1xp4ejvb

// Multi-tag event (tags sorted alphabetically)
event := dcb.InputEvent{
    Type: "StudentEnrolled", 
    Tags: []dcb.Tag{
        {Key: "course_id", Value: "course1"},
        {Key: "student_id", Value: "student123"},
    },
    Data: []byte(`{"enrolled_at": "2024-01-15"}`),
}
// Generates: course_id_student_id_01jxfvstchezwr2z7p6d3f1a7v
```

## Tag Processing

### Alphabetical Sorting
Tag keys are sorted alphabetically to ensure consistent TypeID generation regardless of the order they're provided:

```go
// These two events will generate the same TypeID prefix
event1 := dcb.InputEvent{
    Tags: []dcb.Tag{
        {Key: "course_id", Value: "course1"},
        {Key: "student_id", Value: "student123"},
    },
}

event2 := dcb.InputEvent{
    Tags: []dcb.Tag{
        {Key: "student_id", Value: "student123"},
        {Key: "course_id", Value: "course1"},
    },
}
// Both generate: course_id_student_id_01jxfvstchezwr2z7p6d3f1a7v
```

### Smart Truncation
Long prefixes are automatically truncated to fit within VARCHAR(64) database limits:

```go
// Very long tag keys
event := dcb.InputEvent{
    Tags: []dcb.Tag{
        {Key: "very_long_entity_type_name", Value: "value1"},
        {Key: "another_extremely_long_tag_key", Value: "value2"},
        {Key: "third_super_long_tag_key_name", Value: "value3"},
    },
}
// Generates: another_extremely_long_tag_key_third_super_long_tag_key_name_very_long_entity_type_name_01jxfvsth3ezwvxjec1xp4ejvb
// (truncated to fit within 64 characters including the UUID part)
```

### Sanitization
Special characters are converted to underscores for database compatibility:

```go
event := dcb.InputEvent{
    Tags: []dcb.Tag{
        {Key: "user-id", Value: "user123"},
        {Key: "order number", Value: "order456"},
    },
}
// Generates: order_number_user_id_01jxfvsth3ezwvxjec1xp4ejvb
```

## Database Schema

The library uses VARCHAR(64) columns to accommodate TypeID prefixes:

```sql
CREATE TABLE events (
    id VARCHAR(64) PRIMARY KEY,           -- TypeID with tag-based prefix
    type TEXT NOT NULL,
    tags JSONB NOT NULL,
    data JSONB NOT NULL,
    position BIGSERIAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    causation_id VARCHAR(64) NOT NULL,    -- TypeID reference
    correlation_id VARCHAR(64) NOT NULL   -- TypeID reference
);
```

## Example Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

func main() {
    store := dcb.NewPostgresStore("postgres://user:pass@localhost/db")
    
    // Events automatically get TypeIDs based on tags
    events := []dcb.InputEvent{
        {
            Type: "CourseCreated",
            Tags: []dcb.Tag{{Key: "course_id", Value: "go101"}},
            Data: []byte(`{"title": "Go Fundamentals"}`),
        },
        {
            Type: "StudentEnrolled",
            Tags: []dcb.Tag{
                {Key: "course_id", Value: "go101"},
                {Key: "student_id", Value: "alice123"},
            },
            Data: []byte(`{"enrolled_at": "2024-01-15"}`),
        },
        {
            Type: "LessonCompleted",
            Tags: []dcb.Tag{
                {Key: "course_id", Value: "go101"},
                {Key: "lesson_id", Value: "lesson1"},
                {Key: "student_id", Value: "alice123"},
            },
            Data: []byte(`{"completed_at": "2024-01-16"}`),
        },
    }
    
    positions, err := store.AppendEvents(context.Background(), events)
    if err != nil {
        log.Fatal(err)
    }
    
    // Event IDs will be meaningful TypeIDs like:
    // course_id_01jxfvsth3ezwvxjec1xp4ejvb
    // course_id_student_id_01jxfvstchezwr2z7p6d3f1a7v
    // course_id_lesson_id_student_id_01jxfvstdezwr2z7p6d3f1a7v
    
    log.Printf("Events appended at positions: %v", positions)
}
```

## Real-World Examples

### E-commerce System
```go
// Order events
events := []dcb.InputEvent{
    {
        Type: "OrderCreated",
        Tags: []dcb.Tag{
            {Key: "order_id", Value: "order123"},
            {Key: "customer_id", Value: "customer456"},
        },
        Data: []byte(`{"total": 99.99}`),
    },
    {
        Type: "PaymentProcessed",
        Tags: []dcb.Tag{
            {Key: "order_id", Value: "order123"},
            {Key: "payment_id", Value: "payment789"},
        },
        Data: []byte(`{"amount": 99.99}`),
    },
}
// Generates: customer_id_order_id_01jxfvsth3ezwvxjec1xp4ejvb
//           order_id_payment_id_01jxfvstchezwr2z7p6d3f1a7v
```

### Learning Management System
```go
// Course events
events := []dcb.InputEvent{
    {
        Type: "CoursePublished",
        Tags: []dcb.Tag{
            {Key: "course_id", Value: "go101"},
            {Key: "instructor_id", Value: "instructor123"},
        },
        Data: []byte(`{"title": "Go Programming"}`),
    },
    {
        Type: "StudentEnrolled",
        Tags: []dcb.Tag{
            {Key: "course_id", Value: "go101"},
            {Key: "student_id", Value: "student456"},
        },
        Data: []byte(`{"enrolled_at": "2024-01-15"}`),
    },
}
// Generates: course_id_instructor_id_01jxfvsth3ezwvxjec1xp4ejvb
//           course_id_student_id_01jxfvstchezwr2z7p6d3f1a7v
```

## Performance Considerations

- **Storage**: TypeIDs use slightly more storage than UUIDs (64 vs 36 characters)
- **Indexing**: VARCHAR(64) indexes perform well for TypeID queries
- **Sorting**: K-sortable nature maintains efficient chronological ordering
- **Prefix Queries**: TypeID prefixes enable efficient entity-based queries

## Best Practices

1. **Use Descriptive Tag Keys**: Choose tag keys that clearly identify entities
2. **Keep Tag Keys Short**: Shorter keys leave more room for the UUID part
3. **Be Consistent**: Use the same tag key patterns across your application
4. **Monitor Prefix Length**: Very long prefixes will be truncated

## Troubleshooting

### Long Prefixes
If you see truncated TypeIDs, consider using shorter tag keys:

```go
// Instead of this:
{Key: "very_long_entity_type_name", Value: "value"}

// Use this:
{Key: "entity_type", Value: "value"}
```

### Special Characters
Tag keys with special characters are automatically sanitized:

```go
// This:
{Key: "user-id", Value: "123"}

// Becomes this in the TypeID:
// user_id_01jxfvsth3ezwvxjec1xp4ejvb
```

## References

- [TypeID Specification](https://github.com/jetify-com/typeid) - The official TypeID specification
- [Stripe IDs](https://stripe.com/docs/api#resource_object-id) - Inspiration for TypeID design
- [K-Sortable IDs](https://en.wikipedia.org/wiki/K-sorted_sequence) - Understanding K-sortable identifiers 