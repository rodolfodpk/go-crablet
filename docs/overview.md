# Overview and Key Concepts

go-crablet is a Go library that implements the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern, introduced by Sara Pellegrini in her blog post "Killing the Aggregate". DCB provides a pragmatic approach to balancing strong consistency with flexibility in event-driven systems, without relying on rigid transactional boundaries.

Unlike traditional event sourcing approaches that use strict constraints to maintain immediate consistency, DCB allows for selective enforcement of strong consistency where needed, particularly for operations that span multiple entities. This ensures critical business processes and cross-entity invariants remain reliable while avoiding the constraints of traditional transactional models.

The implementation leverages PostgreSQL's robust concurrency control mechanisms (MVCC and optimistic locking) to handle concurrent operations efficiently, while maintaining ACID guarantees at the database level.

## Key Concepts

- **Single Event Stream**: While traditional event sourcing uses one stream per aggregate (e.g., one stream for Course aggregate, another for Student aggregate), DCB uses a single event stream per bounded context. You can still use aggregates if they make sense for your domain, but they're not required to enforce consistency
- **Tag-based Events**: Events are tagged with relevant identifiers, allowing one event to affect multiple concepts without artificial boundaries
- **Dynamic Consistency**: Consistency is enforced by checking if any events matching a query appeared after a known position. This ensures that events affecting the same concept are processed in order
- **Flexible Boundaries**: No need for predefined aggregates or rigid transactional boundaries - consistency boundaries emerge naturally from your queries, though you can still use aggregates where they provide value
- **Concurrent Operations**: The implementation allows true concurrent operations by leveraging PostgreSQL's concurrency control mechanisms, rather than using application-level locks

## Comparison with Traditional Event Sourcing

The key difference from traditional event sourcing:

Traditional Event Sourcing | DCB Approach
-------------------------|------------
One stream per aggregate (required) | One stream per bounded context (aggregates optional)
Aggregates enforce consistency | Query-based position checks
Rigid aggregate boundaries | Dynamic query-based boundaries
Predefined consistency rules | Emergent consistency through queries
Application-level locking | Database-level concurrency control

For example, in a course subscription system:

Traditional Approach | DCB Approach
-------------------|------------
Separate streams for `Course` and `Student` aggregates | Single stream with events tagged with both `course_id` and `student_id`
Saga to coordinate subscription | Single event with both tags
Two separate events for the same fact | One event affecting multiple concepts
Aggregate boundaries limit flexibility | Natural consistency through query-based position checks 