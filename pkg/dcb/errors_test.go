package dcb

import (
	"errors"
	"testing"
)

func TestIsConcurrencyError(t *testing.T) {
	t.Run("detects ConcurrencyError correctly", func(t *testing.T) {
		err := &ConcurrencyError{
			EventStoreError: EventStoreError{
				Op:  "test",
				Err: errors.New("concurrency issue"),
			},
			ExpectedPosition: 100,
			ActualPosition:   101,
		}

		if !IsConcurrencyError(err) {
			t.Error("IsConcurrencyError should return true for ConcurrencyError")
		}
	})

	t.Run("returns false for non-ConcurrencyError", func(t *testing.T) {
		err := errors.New("regular error")
		if IsConcurrencyError(err) {
			t.Error("IsConcurrencyError should return false for regular error")
		}
	})
}

func TestIsTableStructureError(t *testing.T) {
	t.Run("detects TableStructureError correctly", func(t *testing.T) {
		err := &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "test",
				Err: errors.New("table structure issue"),
			},
			TableName:    "test_table",
			ColumnName:   "test_column",
			ExpectedType: "VARCHAR",
			ActualType:   "INTEGER",
			Issue:        "type mismatch",
		}

		if !IsTableStructureError(err) {
			t.Error("IsTableStructureError should return true for TableStructureError")
		}
	})

	t.Run("returns false for non-TableStructureError", func(t *testing.T) {
		err := errors.New("regular error")
		if IsTableStructureError(err) {
			t.Error("IsTableStructureError should return false for regular error")
		}
	})
}

func TestGetTableStructureError(t *testing.T) {
	t.Run("extracts TableStructureError correctly", func(t *testing.T) {
		err := &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "test",
				Err: errors.New("table structure issue"),
			},
			TableName:    "test_table",
			ColumnName:   "test_column",
			ExpectedType: "VARCHAR",
			ActualType:   "INTEGER",
			Issue:        "type mismatch",
		}

		tableErr, ok := GetTableStructureError(err)
		if !ok {
			t.Error("GetTableStructureError should return true for TableStructureError")
		}
		if tableErr.TableName != "test_table" {
			t.Errorf("expected TableName 'test_table', got '%s'", tableErr.TableName)
		}
		if tableErr.ColumnName != "test_column" {
			t.Errorf("expected ColumnName 'test_column', got '%s'", tableErr.ColumnName)
		}
		if tableErr.ExpectedType != "VARCHAR" {
			t.Errorf("expected ExpectedType 'VARCHAR', got '%s'", tableErr.ExpectedType)
		}
		if tableErr.ActualType != "INTEGER" {
			t.Errorf("expected ActualType 'INTEGER', got '%s'", tableErr.ActualType)
		}
		if tableErr.Issue != "type mismatch" {
			t.Errorf("expected Issue 'type mismatch', got '%s'", tableErr.Issue)
		}
	})

	t.Run("returns false when extracting non-TableStructureError", func(t *testing.T) {
		err := errors.New("regular error")
		_, ok := GetTableStructureError(err)
		if ok {
			t.Error("GetTableStructureError should return false for regular error")
		}
	})
}

func TestAsConcurrencyError(t *testing.T) {
	t.Run("works as alias for GetConcurrencyError", func(t *testing.T) {
		err := &ConcurrencyError{
			EventStoreError: EventStoreError{
				Op:  "test",
				Err: errors.New("concurrency issue"),
			},
			ExpectedPosition: 100,
			ActualPosition:   101,
		}

		concurrencyErr, ok := AsConcurrencyError(err)
		if !ok {
			t.Error("AsConcurrencyError should return true for ConcurrencyError")
		}
		if concurrencyErr.ExpectedPosition != 100 {
			t.Errorf("expected ExpectedPosition 100, got %d", concurrencyErr.ExpectedPosition)
		}
		if concurrencyErr.ActualPosition != 101 {
			t.Errorf("expected ActualPosition 101, got %d", concurrencyErr.ActualPosition)
		}
	})
}

func TestAsResourceError(t *testing.T) {
	t.Run("works as alias for GetResourceError", func(t *testing.T) {
		err := &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "test",
				Err: errors.New("resource issue"),
			},
			Resource: "database",
		}

		resourceErr, ok := AsResourceError(err)
		if !ok {
			t.Error("AsResourceError should return true for ResourceError")
		}
		if resourceErr.Resource != "database" {
			t.Errorf("expected Resource 'database', got '%s'", resourceErr.Resource)
		}
	})
}

func TestAsTableStructureError(t *testing.T) {
	t.Run("works as alias for GetTableStructureError", func(t *testing.T) {
		err := &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "test",
				Err: errors.New("table structure issue"),
			},
			TableName: "test_table",
			Issue:     "test issue",
		}

		tableErr, ok := AsTableStructureError(err)
		if !ok {
			t.Error("AsTableStructureError should return true for TableStructureError")
		}
		if tableErr.TableName != "test_table" {
			t.Errorf("expected TableName 'test_table', got '%s'", tableErr.TableName)
		}
		if tableErr.Issue != "test issue" {
			t.Errorf("expected Issue 'test issue', got '%s'", tableErr.Issue)
		}
	})
}
