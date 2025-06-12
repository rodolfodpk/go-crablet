package dcb

import (
	"strings"
	"testing"
)

func TestGenerateTagBasedTypeID(t *testing.T) {
	tests := []struct {
		name     string
		tags     []Tag
		expected string
	}{
		{
			name: "single tag",
			tags: []Tag{{Key: "course_id", Value: "course1"}},
			// Should generate: course_id_<uuid>
		},
		{
			name: "multiple tags sorted alphabetically",
			tags: []Tag{
				{Key: "student_id", Value: "student123"},
				{Key: "course_id", Value: "course1"},
			},
			// Should generate: course_id_student_id_<uuid>
		},
		{
			name: "tags with special characters",
			tags: []Tag{
				{Key: "user-id", Value: "user123"},
				{Key: "order number", Value: "order456"},
			},
			// Should generate: order_number_user_id_<uuid>
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTagBasedTypeID(tt.tags)

			// Verify TypeID format
			if len(result) == 0 {
				t.Error("TypeID should not be empty")
			}

			// Verify it contains underscore (prefix_uuid format)
			if !strings.Contains(result, "_") {
				t.Errorf("TypeID should contain underscore: %s", result)
			}

			// Verify it's not too long (VARCHAR(64) limit)
			if len(result) > 64 {
				t.Errorf("TypeID too long (%d chars): %s", len(result), result)
			}

			t.Logf("Generated TypeID: %s", result)
		})
	}
}

func TestSanitizeForTypeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user-id", "user_id"},
		{"order number", "order_number"},
		{"Course ID", "course_id"},
		{"user@domain.com", "user_domain_com"},
		{"normal_key", "normal_key"},
		{"multiple__underscores", "multiple_underscores"},
		{"_leading_underscore", "leading_underscore"},
		{"trailing_underscore_", "trailing_underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeForTypeID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeForTypeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractUUIDFromTypeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"course_id_01jxfvsth3ezwvxjec1xp4ejvb", "01jxfvsth3ezwvxjec1xp4ejvb"},
		{"course_id_student_id_01jxfvstchezwr2z7p6d3f1a7v", "01jxfvstchezwr2z7p6d3f1a7v"},
		{"simple_uuid", "simple_uuid"}, // Fallback for invalid TypeID
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractUUIDFromTypeID(tt.input)
			if result != tt.expected {
				t.Errorf("extractUUIDFromTypeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTypeIDConsistency(t *testing.T) {
	// Test that same tags always generate same prefix (UUID part will be different)
	tags := []Tag{
		{Key: "student_id", Value: "student123"},
		{Key: "course_id", Value: "course1"},
	}

	// Generate multiple TypeIDs
	typeIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		typeIDs[i] = generateTagBasedTypeID(tags)
	}

	// Extract prefixes (everything before the last underscore)
	prefixes := make([]string, 5)
	for i, typeID := range typeIDs {
		prefixes[i] = typeID[:len(typeID)-27] // Remove UUID part (26 chars + underscore)
	}

	// All prefixes should be the same
	firstPrefix := prefixes[0]
	for i, prefix := range prefixes {
		if prefix != firstPrefix {
			t.Errorf("Prefix %d (%s) differs from first prefix (%s)", i, prefix, firstPrefix)
		}
	}

	t.Logf("Consistent prefix: %s", firstPrefix)
	t.Logf("Generated TypeIDs: %v", typeIDs)
}
