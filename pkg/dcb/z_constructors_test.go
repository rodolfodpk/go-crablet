package dcb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAppendCondition(t *testing.T) {
	// Test with valid query
	query := NewQuerySimple([]Tag{NewTag("test", "value")}, "TestEvent")
	condition := NewAppendCondition(query)

	// Verify the condition implements the AppendCondition interface
	assert.Implements(t, (*AppendCondition)(nil), condition)

	// Test with nil query
	condition = NewAppendCondition(nil)
	assert.Implements(t, (*AppendCondition)(nil), condition)
}
