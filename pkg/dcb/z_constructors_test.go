package dcb

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAppendConditionWithAfter(t *testing.T) {
	tests := []struct {
		name              string
		failIfEventsMatch Query
		after             *int64
		description       string
	}{
		{
			name:              "Both fail condition and after position",
			failIfEventsMatch: NewQuery(NewTags("user_id", "123"), "UserCreated"),
			after:             int64Ptr(100),
			description:       "Should create condition with both fail condition and after position",
		},
		{
			name:              "Only fail condition (nil after)",
			failIfEventsMatch: NewQuery(NewTags("user_id", "123"), "UserCreated"),
			after:             nil,
			description:       "Should create condition with only fail condition",
		},
		{
			name:              "Only after position (nil fail condition)",
			failIfEventsMatch: nil,
			after:             int64Ptr(200),
			description:       "Should create condition with only after position",
		},
		{
			name:              "Both nil",
			failIfEventsMatch: nil,
			after:             nil,
			description:       "Should create condition with both parameters nil",
		},
		{
			name:              "Complex fail condition with multiple event types",
			failIfEventsMatch: NewQuery(NewTags("user_id", "123"), "UserCreated", "UserUpdated"),
			after:             int64Ptr(300),
			description:       "Should handle complex fail conditions",
		},
		{
			name:              "Fail condition with multiple tags",
			failIfEventsMatch: NewQuery(NewTags("user_id", "123", "status", "active"), "UserCreated"),
			after:             int64Ptr(400),
			description:       "Should handle fail conditions with multiple tags",
		},
		{
			name:              "Zero after position",
			failIfEventsMatch: NewQuery(NewTags("user_id", "123"), "UserCreated"),
			after:             int64Ptr(0),
			description:       "Should handle zero after position",
		},
		{
			name:              "Large after position",
			failIfEventsMatch: NewQuery(NewTags("user_id", "123"), "UserCreated"),
			after:             int64Ptr(9223372036854775807), // max int64
			description:       "Should handle large after position values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := NewAppendConditionWithAfter(tt.failIfEventsMatch, tt.after)

			// Verify the condition is not nil
			assert.NotNil(t, condition, "Condition should not be nil")

			// Verify the condition implements the AppendCondition interface
			assert.Implements(t, (*AppendCondition)(nil), condition)

			// Test the getAfter method
			afterResult := condition.getAfter()
			assert.Equal(t, tt.after, afterResult, "getAfter() should return the correct after position")

			// Test the getFailIfEventsMatch method
			failResult := condition.getFailIfEventsMatch()
			if tt.failIfEventsMatch != nil {
				assert.NotNil(t, failResult, "getFailIfEventsMatch() should not be nil when fail condition is provided")
				assert.Equal(t, tt.failIfEventsMatch, *failResult, "getFailIfEventsMatch() should return the correct fail condition")
			} else {
				assert.Nil(t, failResult, "getFailIfEventsMatch() should be nil when fail condition is nil")
			}

			// Test the setAfterPosition method
			newAfter := int64Ptr(999)
			condition.setAfterPosition(newAfter)
			assert.Equal(t, newAfter, condition.getAfter(), "setAfterPosition() should update the after position")

			// Test JSON marshaling (the condition should be marshalable)
			jsonData, err := json.Marshal(condition)
			assert.NoError(t, err, "Condition should be JSON marshalable")
			assert.NotEmpty(t, jsonData, "JSON data should not be empty")

			// Verify the JSON structure contains expected fields
			var jsonMap map[string]interface{}
			err = json.Unmarshal(jsonData, &jsonMap)
			assert.NoError(t, err, "JSON should be valid")

			// Check that after position is present if provided
			if tt.after != nil {
				assert.Contains(t, jsonMap, "after", "JSON should contain 'after' field when after position is provided")
			}

			// Check that fail_if_events_match is present if provided
			if tt.failIfEventsMatch != nil {
				assert.Contains(t, jsonMap, "fail_if_events_match", "JSON should contain 'fail_if_events_match' field when fail condition is provided")
			}
		})
	}
}

// Helper function to create int64 pointers
func int64Ptr(v int64) *int64 {
	return &v
}
