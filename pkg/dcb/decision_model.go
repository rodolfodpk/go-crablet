package dcb

import (
	"context"
	"encoding/json"
)

// DecisionModel represents the result of building a decision model
// It contains the projected states and a combined append condition for optimistic locking
type DecisionModel struct {
	// States contains the projected states keyed by projector ID
	States map[string]any

	// AppendCondition combines all projector queries with OR logic
	// This ensures consistency by checking that no conflicting events exist
	AppendCondition AppendCondition
}

// BuildDecisionModel projects multiple states and returns a combined append condition
// This is inspired by the TypeScript DCB pattern for command handlers
// It encapsulates the common pattern of projecting states and combining queries for optimistic locking
func BuildDecisionModel(ctx context.Context, store EventStore, projectors map[string]BatchProjector) (*DecisionModel, error) {
	// Convert map to slice for ProjectBatch
	projectorSlice := make([]BatchProjector, 0, len(projectors))
	for id, projector := range projectors {
		projectorSlice = append(projectorSlice, BatchProjector{
			ID:             id,
			StateProjector: projector.StateProjector,
		})
	}

	// Project all states in one query
	result, err := store.ProjectBatch(ctx, projectorSlice)
	if err != nil {
		return nil, err
	}

	// Combine all queries for optimistic locking
	combinedQuery := CombineProjectorQueries(projectorSlice)

	return &DecisionModel{
		States: result.States,
		AppendCondition: AppendCondition{
			FailIfEventsMatch: combinedQuery,
			After:             &result.Position,
		},
	}, nil
}

// Example usage functions to demonstrate the pattern:

// CourseExistsProjection creates a projection to check if a course exists
func CourseExistsProjection(courseID string) BatchProjector {
	return BatchProjector{
		ID: "courseExists",
		StateProjector: StateProjector{
			Query:        NewQuery(NewTags("course_id", courseID), "CourseDefined"),
			InitialState: false,
			TransitionFn: func(state any, event Event) any {
				if event.Type == "CourseDefined" {
					return true
				}
				return state
			},
		},
	}
}

// CourseCapacityProjection creates a projection to track course capacity
func CourseCapacityProjection(courseID string) BatchProjector {
	return BatchProjector{
		ID: "courseCapacity",
		StateProjector: StateProjector{
			Query:        NewQuery(NewTags("course_id", courseID), "CourseDefined", "CapacityChanged"),
			InitialState: 0,
			TransitionFn: func(state any, event Event) any {
				switch event.Type {
				case "CourseDefined":
					// Parse capacity from event data
					var data map[string]any
					if err := json.Unmarshal(event.Data, &data); err == nil {
						if capacity, ok := data["capacity"].(float64); ok {
							return int(capacity)
						}
					}
					return state
				case "CapacityChanged":
					// Parse new capacity from event data
					var data map[string]any
					if err := json.Unmarshal(event.Data, &data); err == nil {
						if capacity, ok := data["capacity"].(float64); ok {
							return int(capacity)
						}
					}
					return state
				}
				return state
			},
		},
	}
}

// StudentEnrollmentProjection creates a projection to track student enrollments
func StudentEnrollmentProjection(courseID string) BatchProjector {
	return BatchProjector{
		ID: "studentEnrollments",
		StateProjector: StateProjector{
			Query:        NewQuery(NewTags("course_id", courseID), "StudentEnrolled", "StudentUnenrolled"),
			InitialState: make([]string, 0),
			TransitionFn: func(state any, event Event) any {
				enrollments := state.([]string)
				switch event.Type {
				case "StudentEnrolled":
					var data map[string]any
					if err := json.Unmarshal(event.Data, &data); err == nil {
						if studentID, ok := data["studentId"].(string); ok {
							// Add student if not already enrolled
							for _, enrolled := range enrollments {
								if enrolled == studentID {
									return enrollments // Already enrolled
								}
							}
							return append(enrollments, studentID)
						}
					}
				case "StudentUnenrolled":
					var data map[string]any
					if err := json.Unmarshal(event.Data, &data); err == nil {
						if studentID, ok := data["studentId"].(string); ok {
							// Remove student from enrollments
							newEnrollments := make([]string, 0, len(enrollments))
							for _, enrolled := range enrollments {
								if enrolled != studentID {
									newEnrollments = append(newEnrollments, enrolled)
								}
							}
							return newEnrollments
						}
					}
				}
				return enrollments
			},
		},
	}
}
