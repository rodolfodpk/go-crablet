# State Projection with PostgreSQL Streaming

go-crablet implements efficient state projection by leveraging PostgreSQL's streaming capabilities. Instead of loading all events into memory, events are streamed directly from the database and processed one at a time. This approach provides several benefits:

1. **Memory Efficiency**: Events are processed in a streaming fashion, making it suitable for large event streams
2. **Database Efficiency**: Uses PostgreSQL's native JSONB indexing and querying capabilities
3. **Consistent Views**: The same query used for consistency checks is used for state projection

## Implementation Details

Here's how it works under the hood:

```go
// The ProjectState method streams events from PostgreSQL
func (es *eventStore) ProjectState(ctx context.Context, projector StateProjector) (int64, any, error) {
    // Handle empty or nil tags
    var queryTags []byte
    if len(projector.Tags) > 0 {
        // Build JSONB query condition from tags
        tagMap := make(map[string]string)
        for _, t := range projector.Tags {
            tagMap[t.Key] = t.Value
        }
        var err error
        queryTags, err = json.Marshal(tagMap)
        if err != nil {
            return 0, projector.InitialState, fmt.Errorf("failed to marshal query tags: %w", err)
        }
    }

    // Construct SQL query
    var sqlQuery string
    var args []interface{}
    
    if queryTags != nil {
        // Use JSONB containment operator @> when tags are provided
        sqlQuery = "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE tags @> $1"
        args = append(args, queryTags)
    } else {
        // When no tags are provided, select all events
        sqlQuery = "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events"
    }

    // Add event type filtering if specified
    if len(projector.EventTypes) > 0 {
        if len(args) > 0 {
            sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
        } else {
            sqlQuery += fmt.Sprintf(" WHERE type = ANY($%d)", len(args)+1)
        }
        args = append(args, projector.EventTypes)
    }

    // Stream rows from PostgreSQL
    rows, err := es.pool.Query(ctx, sqlQuery, args...)
    if err != nil {
        return 0, projector.InitialState, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    // Initialize state
    state := projector.InitialState
    position := int64(0)

    // Process events one at a time
    for rows.Next() {
        var row rowEvent
        if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
            return 0, projector.InitialState, fmt.Errorf("failed to scan row: %w", err)
        }

        // Convert row to Event
        event := convertRowToEvent(row)
        
        // Apply projector
        state = projector.TransitionFn(state, event)
        position = row.Position
    }

    return position, state, nil
}
```

## Query Behavior

The `ProjectState` method provides flexible state projection capabilities. Here are examples of how to use it:

1. **Projecting All Events**:
   ```go
   // Create a projector that handles all events
   projector := dcb.StateProjector{
       Query: dcb.NewQuery(nil), // Empty query matches all events
       InitialState: &MyState{},
       TransitionFn: func(state any, event dcb.Event) any {
           // Handle all events
           return state
       },
   }
   
   // Project state using the projector
   position, state, err := store.ProjectState(ctx, projector)
   if err != nil {
       panic(err)
   }
   ```

2. **Projecting Specific Event Types**:
   ```go
   // Create a projector that handles specific event types
   projector := dcb.StateProjector{
       Query: dcb.NewQuery(nil, "StudentSubscribedToCourse", "StudentUnsubscribedFromCourse"),
       InitialState: &SubscriptionState{},
       TransitionFn: func(state any, event dcb.Event) any {
           // Only subscription events will be received due to Query.EventTypes
           switch event.Type {
           case "StudentSubscribedToCourse":
               // Handle subscription event
           case "StudentUnsubscribedFromCourse":
               // Handle unsubscription event
           }
           return state
       },
   }
   
   // Project state using the projector
   position, state, err := store.ProjectState(ctx, projector)
   if err != nil {
       panic(err)
   }
   ```

3. **Building Different Views**:
   ```go
   // Course view projector
   courseProjector := dcb.StateProjector{
       Query: dcb.NewQuery(dcb.NewTags("course_id", "c1")), // Filter by course_id at database level
       InitialState: &CourseState{
           StudentIDs: make(map[string]bool),
       },
       TransitionFn: func(state any, event dcb.Event) any {
           course := state.(*CourseState)
           // Only events for course c1 will be received due to Query.Tags
           switch event.Type {
           case "StudentSubscribedToCourse":
               // Add student to course
           case "StudentUnsubscribedFromCourse":
               // Remove student from course
           }
           return course
       },
   }

   // Student view projector
   studentProjector := dcb.StateProjector{
       Query: dcb.NewQuery(dcb.NewTags("student_id", "s1")), // Filter by student_id at database level
       InitialState: &StudentState{
           CourseIDs: make(map[string]bool),
       },
       TransitionFn: func(state any, event dcb.Event) any {
           student := state.(*StudentState)
           // Only events for student s1 will be received due to Query.Tags
           switch event.Type {
           case "StudentSubscribedToCourse":
               // Add course to student
           case "StudentUnsubscribedFromCourse":
               // Remove course from student
           }
           return student
       },
   }
   