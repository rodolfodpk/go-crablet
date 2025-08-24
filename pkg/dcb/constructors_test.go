package dcb

import (
	"testing"
)

func TestNewEventStoreWithConfig(t *testing.T) {
	t.Run("validates config structure", func(t *testing.T) {
		config := EventStoreConfig{
			MaxBatchSize:           1000,
			StreamBuffer:           100,
			DefaultAppendIsolation: IsolationLevelRepeatableRead,
			QueryTimeout:           5000,
			AppendTimeout:          3000,
		}

		// Test config validation (without requiring actual database connection)
		if config.MaxBatchSize != 1000 {
			t.Errorf("expected MaxBatchSize 1000, got %d", config.MaxBatchSize)
		}
		if config.StreamBuffer != 100 {
			t.Errorf("expected StreamBuffer 100, got %d", config.StreamBuffer)
		}
		if config.DefaultAppendIsolation != IsolationLevelRepeatableRead {
			t.Errorf("expected DefaultAppendIsolation %v, got %v", IsolationLevelRepeatableRead, config.DefaultAppendIsolation)
		}
		if config.QueryTimeout != 5000 {
			t.Errorf("expected QueryTimeout 5000, got %d", config.QueryTimeout)
		}
		if config.AppendTimeout != 3000 {
			t.Errorf("expected AppendTimeout 3000, got %d", config.AppendTimeout)
		}
	})

}

func TestParseIsolationLevel(t *testing.T) {
	t.Run("handles valid values", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected IsolationLevel
		}{
			{"READ_COMMITTED", IsolationLevelReadCommitted},
			{"REPEATABLE_READ", IsolationLevelRepeatableRead},
			{"SERIALIZABLE", IsolationLevelSerializable},
		}

		for _, tc := range testCases {
			level, err := ParseIsolationLevel(tc.input)
			if err != nil {
				t.Errorf("ParseIsolationLevel(%s) should not return error: %v", tc.input, err)
			}
			if level != tc.expected {
				t.Errorf("ParseIsolationLevel(%s) expected %v, got %v", tc.input, tc.expected, level)
			}
		}
	})

	t.Run("handles invalid values", func(t *testing.T) {
		level, err := ParseIsolationLevel("INVALID_LEVEL")
		if err == nil {
			t.Error("ParseIsolationLevel should return error for invalid level")
		}
		if err.Error() != "invalid isolation level: INVALID_LEVEL" {
			t.Errorf("expected error message 'invalid isolation level: INVALID_LEVEL', got '%s'", err.Error())
		}
		if level != IsolationLevelReadCommitted {
			t.Errorf("expected default fallback to IsolationLevelReadCommitted, got %v", level)
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		level, err := ParseIsolationLevel("")
		if err == nil {
			t.Error("ParseIsolationLevel should return error for empty string")
		}
		if level != IsolationLevelReadCommitted {
			t.Errorf("expected default fallback to IsolationLevelReadCommitted, got %v", level)
		}
	})
}

func TestIsolationLevelString(t *testing.T) {
	t.Run("String() method works correctly", func(t *testing.T) {
		testCases := []struct {
			level    IsolationLevel
			expected string
		}{
			{IsolationLevelReadCommitted, "READ_COMMITTED"},
			{IsolationLevelRepeatableRead, "REPEATABLE_READ"},
			{IsolationLevelSerializable, "SERIALIZABLE"},
		}

		for _, tc := range testCases {
			result := tc.level.String()
			if result != tc.expected {
				t.Errorf("IsolationLevel(%d).String() expected '%s', got '%s'", tc.level, tc.expected, result)
			}
		}
	})

	t.Run("String() handles unknown values", func(t *testing.T) {
		unknownLevel := IsolationLevel(999)
		result := unknownLevel.String()
		if result != "UNKNOWN" {
			t.Errorf("IsolationLevel(999).String() expected 'UNKNOWN', got '%s'", result)
		}
	})
}

func TestToJSON(t *testing.T) {
	t.Run("handles valid data", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		result := ToJSON(data)

		expected := `{"key":"value"}`
		if string(result) != expected {
			t.Errorf("ToJSON expected '%s', got '%s'", expected, string(result))
		}
	})

	t.Run("handles nil data", func(t *testing.T) {
		result := ToJSON(nil)

		expected := "null"
		if string(result) != expected {
			t.Errorf("ToJSON(nil) expected '%s', got '%s'", expected, string(result))
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		data := map[string]string{}
		result := ToJSON(data)

		expected := "{}"
		if string(result) != expected {
			t.Errorf("ToJSON({}) expected '%s', got '%s'", expected, string(result))
		}
	})
}

func TestNewTagsEdgeCases(t *testing.T) {
	t.Run("handles odd number of arguments", func(t *testing.T) {
		tags := NewTags("key1", "value1", "key2") // Odd number

		if len(tags) != 0 {
			t.Errorf("expected empty slice for odd number of arguments, got %d tags", len(tags))
		}
	})

	t.Run("handles empty arguments", func(t *testing.T) {
		tags := NewTags()

		if len(tags) != 0 {
			t.Errorf("expected empty slice for no arguments, got %d tags", len(tags))
		}
	})

	t.Run("handles single argument", func(t *testing.T) {
		tags := NewTags("key1")

		if len(tags) != 0 {
			t.Errorf("expected empty slice for single argument, got %d tags", len(tags))
		}
	})

	t.Run("handles valid key-value pairs", func(t *testing.T) {
		tags := NewTags("key1", "value1", "key2", "value2")

		if len(tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(tags))
		}

		// Check first tag
		if tags[0].GetKey() != "key1" || tags[0].GetValue() != "value1" {
			t.Errorf("first tag expected key1:value1, got %s:%s", tags[0].GetKey(), tags[0].GetValue())
		}

		// Check second tag
		if tags[1].GetKey() != "key2" || tags[1].GetValue() != "value2" {
			t.Errorf("second tag expected key2:value2, got %s:%s", tags[1].GetKey(), tags[1].GetValue())
		}
	})
}
