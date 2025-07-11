# Bounded Contexts and Append Conditions

## Overview

go-crablet supports bounded context isolation through configurable event tables. Each bounded context has its own event table, ensuring proper domain separation.

## Architecture

### EventStore Configuration

Each `EventStore` instance targets a specific event table via the `TargetEventsTable` configuration:

```go
config := dcb.EventStoreConfig{
    TargetEventsTable: "course_management_events", // BC-specific table
    // ... other config
}
store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

### Append Condition Scope

**Critical**: Append conditions are scoped to the specific bounded context table only.

```go
// This only checks conditions within the "course_management_events" table
query := dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined")
condition := dcb.NewAppendCondition(query)
err := store.Append(ctx, events, &condition)
```

## Why Cross-BC Append Conditions Are Not Supported

### 1. Complexity
Checking append conditions across multiple bounded context tables would require:
- Knowledge of all BC table names
- Complex multi-table queries
- Coordination between different BC schemas
- Performance overhead

### 2. Domain-Driven Design Principles
Bounded contexts should be:
- **Independent**: Each BC manages its own data
- **Isolated**: Changes in one BC don't directly affect others
- **Focused**: Each BC has a single responsibility

### 3. Proper Cross-BC Communication
Instead of cross-BC append conditions, use:

#### Domain Events
```go
// Course Management BC
courseEvent := dcb.NewInputEvent("CourseCreated", 
    dcb.NewTags("course_id", "c1"), 
    dcb.ToJSON(courseData))
courseStore.Append(ctx, []dcb.InputEvent{courseEvent}, nil)

// Student Management BC listens for domain events
studentQuery := dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseCreated")
events, _ := studentStore.Query(ctx, studentQuery, nil)
```

#### Saga Pattern
```go
// Coordinate across BCs using saga steps
func enrollStudentSaga(ctx context.Context, courseID, studentID string) error {
    // Step 1: Reserve seat in Course Management BC
    courseStore := getCourseManagementStore()
    // ... append reservation event
    
    // Step 2: Register student in Student Management BC  
    studentStore := getStudentManagementStore()
    // ... append registration event
    
    // Step 3: Create enrollment in Enrollment BC
    enrollmentStore := getEnrollmentStore()
    // ... append enrollment event
}
```

## Example: Course Enrollment System

### Bounded Contexts

1. **Course Management BC**
   - Table: `course_management_events`
   - Events: `CourseDefined`, `CourseCapacityChanged`
   - Commands: `CreateCourse`, `UpdateCourseCapacity`

2. **Student Management BC**
   - Table: `student_management_events`
   - Events: `StudentRegistered`, `StudentProfileUpdated`
   - Commands: `RegisterStudent`, `UpdateStudentProfile`

3. **Enrollment BC**
   - Table: `enrollment_events`
   - Events: `StudentEnrolled`, `StudentDropped`
   - Commands: `EnrollStudent`, `DropStudent`

### Implementation

```go
// Course Management BC
courseConfig := dcb.EventStoreConfig{
    TargetEventsTable: "course_management_events",
}
courseStore, _ := dcb.NewEventStoreWithConfig(ctx, pool, courseConfig)

// Student Management BC
studentConfig := dcb.EventStoreConfig{
    TargetEventsTable: "student_management_events", 
}
studentStore, _ := dcb.NewEventStoreWithConfig(ctx, pool, studentConfig)

// Enrollment BC
enrollmentConfig := dcb.EventStoreConfig{
    TargetEventsTable: "enrollment_events",
}
enrollmentStore, _ := dcb.NewEventStoreWithConfig(ctx, pool, enrollmentConfig)
```

### Append Conditions Within Each BC

```go
// Course Management BC - only checks course_management_events table
courseQuery := dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined")
courseCondition := dcb.NewAppendCondition(courseQuery)
courseStore.Append(ctx, courseEvents, &courseCondition)

// Student Management BC - only checks student_management_events table  
studentQuery := dcb.NewQuery(dcb.NewTags("student_id", "s1"), "StudentRegistered")
studentCondition := dcb.NewAppendCondition(studentQuery)
studentStore.Append(ctx, studentEvents, &studentCondition)

// Enrollment BC - only checks enrollment_events table
enrollmentQuery := dcb.NewQuery(dcb.NewTags("course_id", "c1", "student_id", "s1"), "StudentEnrolled")
enrollmentCondition := dcb.NewAppendCondition(enrollmentQuery)
enrollmentStore.Append(ctx, enrollmentEvents, &enrollmentCondition)
```

## Best Practices

### 1. Table Naming Convention
Use descriptive table names that clearly indicate the bounded context:

```
{domain}_{context}_events
```

Examples:
- `course_management_events`
- `student_management_events` 
- `enrollment_events`
- `payment_events`
- `notification_events`

### 2. Event Type Naming
Use BC-specific event type prefixes:

```go
// Course Management BC
"CourseDefined"
"CourseCapacityChanged"
"CourseCancelled"

// Student Management BC
"StudentRegistered"
"StudentProfileUpdated"
"StudentDeactivated"

// Enrollment BC
"StudentEnrolled"
"StudentDropped"
"EnrollmentConfirmed"
```

### 3. Cross-BC Coordination
Use domain events and saga patterns instead of cross-BC append conditions:

```go
// ❌ Don't: Try to check conditions across BCs
// This would be complex and violate BC isolation

// ✅ Do: Use domain events for cross-BC communication
courseEvent := dcb.NewInputEvent("CourseCreated", tags, data)
courseStore.Append(ctx, []dcb.InputEvent{courseEvent}, nil)

// Other BCs listen for domain events
studentStore.Query(ctx, dcb.NewQuery(tags, "CourseCreated"), nil)
```

### 4. Database Schema
Create BC-specific tables in your migration scripts:

```sql
-- Course Management BC
CREATE TABLE course_management_events (
    type VARCHAR(64) NOT NULL,
    tags TEXT[] NOT NULL,
    data JSON NOT NULL,
    transaction_id xid8 NOT NULL,
    position BIGSERIAL NOT NULL PRIMARY KEY,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Student Management BC  
CREATE TABLE student_management_events (
    type VARCHAR(64) NOT NULL,
    tags TEXT[] NOT NULL,
    data JSON NOT NULL,
    transaction_id xid8 NOT NULL,
    position BIGSERIAL NOT NULL PRIMARY KEY,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Enrollment BC
CREATE TABLE enrollment_events (
    type VARCHAR(64) NOT NULL,
    tags TEXT[] NOT NULL,
    data JSON NOT NULL,
    transaction_id xid8 NOT NULL,
    position BIGSERIAL NOT NULL PRIMARY KEY,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Summary

- **Append conditions are BC-scoped**: Each EventStore only checks conditions within its target table
- **Cross-BC coordination**: Use domain events and saga patterns, not cross-BC append conditions
- **Proper isolation**: Each BC manages its own data and business rules
- **Simple and performant**: No complex multi-table condition checking required

This approach maintains proper bounded context isolation while providing the necessary coordination mechanisms for complex business processes. 