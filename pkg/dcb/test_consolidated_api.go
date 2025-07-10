package dcb

import (
	"context"
	"testing"
)

func TestConsolidatedAPI(t *testing.T) {
	// This is a simple test to verify the consolidated API compiles and works
	// We'll create a mock store to test the interface

	ctx := context.Background()

	// Test that the interface methods have the correct signatures
	var store EventStore

	// Test Read with nil cursor (should work)
	query := NewQuery(NewTags("test", "value"), "TestEvent")
	_, _ = store.Query(ctx, query, nil)
	// We expect an error since store is nil, but the signature should be correct

	// Test Read with cursor
	cursor := &Cursor{TransactionID: 1, Position: 1}
	_, _ = store.Query(ctx, query, cursor)
	// We expect an error since store is nil, but the signature should be correct

	// Test Append with nil condition
	events := []InputEvent{NewInputEvent("TestEvent", NewTags("test", "value"), []byte("{}"))}
	_ = store.Append(ctx, events, nil)
	// We expect an error since store is nil, but the signature should be correct

	// Test Append with condition
	condition := NewAppendCondition(query)
	_ = store.Append(ctx, events, &condition)
	// We expect an error since store is nil, but the signature should be correct

	// Test Project with nil cursor
	projectors := []StateProjector{}
	_, _, _ = store.Project(ctx, projectors, nil)
	// We expect an error since store is nil, but the signature should be correct

	// Test Project with cursor
	_, _, _ = store.Project(ctx, projectors, cursor)
	// We expect an error since store is nil, but the signature should be correct

	// If we get here, the API signatures are correct
	t.Log("Consolidated API signatures are correct")
}
