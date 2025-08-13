package dcb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// Database Validation Functions
// =============================================================================

// validateEventsTableExists validates that the events table exists with correct structure
// This is a required table for EventStore to function
func validateEventsTableExists(ctx context.Context, pool *pgxpool.Pool) error {
	return validateTableExists(ctx, pool, "events", true)
}

// validateCommandsTableExists validates that the commands table exists with correct structure
// This is an optional table for command tracking functionality
func validateCommandsTableExists(ctx context.Context, pool *pgxpool.Pool) error {
	return validateTableExists(ctx, pool, "commands", false)
}

// validateTableExists validates that a table exists with correct structure
// required: if true, missing table is an error; if false, missing table is just logged
func validateTableExists(ctx context.Context, pool *pgxpool.Pool, tableName string, required bool) error {
	// Check if table exists
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = $1
		)
	`, tableName).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !exists {
		if required {
			return &TableStructureError{
				EventStoreError: EventStoreError{
					Op:  "validate_table_exists",
					Err: fmt.Errorf("required table %s does not exist", tableName),
				},
				TableName: tableName,
				Issue:     "required table does not exist",
			}
		}
		// Optional table missing - just log and continue
		return nil
	}

	// Table exists, validate its structure
	if err := validateTableStructure(ctx, pool, tableName); err != nil {
		// If it's already a TableStructureError, wrap it with more context
		if tableErr, ok := err.(*TableStructureError); ok {
			tableErr.EventStoreError.Op = "validate_table_exists"
			return tableErr
		}
		// Otherwise, wrap it as a generic error
		return &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "validate_table_exists",
				Err: err,
			},
			TableName: tableName,
			Issue:     "table structure validation failed",
		}
	}

	return nil
}

// validateTableStructure checks that the table has the expected columns and types
func validateTableStructure(ctx context.Context, pool *pgxpool.Pool, tableName string) error {
	// Query to check column structure
	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = $1 
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		return &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "validate_table_structure",
				Err: fmt.Errorf("failed to query table structure: %w", err),
			},
			TableName: tableName,
			Issue:     "failed to query table structure",
		}
	}
	defer rows.Close()

	// Define expected columns for each table
	var expectedColumns map[string]struct {
		dataType   string
		isNullable string
		hasDefault bool
	}

	switch tableName {
	case "events":
		expectedColumns = map[string]struct {
			dataType   string
			isNullable string
			hasDefault bool
		}{
			"type":           {dataType: "character varying", isNullable: "NO", hasDefault: false},
			"tags":           {dataType: "ARRAY", isNullable: "NO", hasDefault: false},
			"data":           {dataType: "json", isNullable: "NO", hasDefault: false},
			"transaction_id": {dataType: "xid8", isNullable: "NO", hasDefault: false},
			"position":       {dataType: "bigint", isNullable: "NO", hasDefault: false},
			"occurred_at":    {dataType: "timestamp with time zone", isNullable: "NO", hasDefault: true},
		}
	case "commands":
		expectedColumns = map[string]struct {
			dataType   string
			isNullable string
			hasDefault bool
		}{
			"transaction_id": {dataType: "xid8", isNullable: "NO", hasDefault: false},
			"type":           {dataType: "character varying", isNullable: "NO", hasDefault: false},
			"data":           {dataType: "jsonb", isNullable: "NO", hasDefault: false},
			"metadata":       {dataType: "jsonb", isNullable: "YES", hasDefault: false},
			"occurred_at":    {dataType: "timestamp with time zone", isNullable: "NO", hasDefault: true},
		}
	default:
		return &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "validate_table_structure",
				Err: fmt.Errorf("unknown table '%s' for validation", tableName),
			},
			TableName: tableName,
			Issue:     "unknown table for validation",
		}
	}

	foundColumns := make(map[string]bool)

	for rows.Next() {
		var columnName, dataType, isNullable, columnDefault sql.NullString
		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault); err != nil {
			return &TableStructureError{
				EventStoreError: EventStoreError{
					Op:  "validate_table_structure",
					Err: fmt.Errorf("failed to scan column info: %w", err),
				},
				TableName: tableName,
				Issue:     "failed to scan column information",
			}
		}

		if !columnName.Valid {
			continue
		}

		foundColumns[columnName.String] = true

		expected, exists := expectedColumns[columnName.String]
		if !exists {
			return &TableStructureError{
				EventStoreError: EventStoreError{
					Op:  "validate_table_structure",
					Err: fmt.Errorf("unexpected column '%s' found", columnName.String),
				},
				TableName:  tableName,
				ColumnName: columnName.String,
				Issue:      "unexpected column found",
			}
		}

		// Check data type (handle array types specially)
		if expected.dataType == "ARRAY" {
			if dataType.String != "ARRAY" {
				return &TableStructureError{
					EventStoreError: EventStoreError{
						Op:  "validate_table_structure",
						Err: fmt.Errorf("column '%s' should be ARRAY type, got %s", columnName.String, dataType.String),
					},
					TableName:    tableName,
					ColumnName:   columnName.String,
					ExpectedType: "ARRAY",
					ActualType:   dataType.String,
					Issue:        "incorrect data type",
				}
			}
		} else if dataType.String != expected.dataType {
			return &TableStructureError{
				EventStoreError: EventStoreError{
					Op:  "validate_table_structure",
					Err: fmt.Errorf("column '%s' should be %s type, got %s", columnName.String, expected.dataType, dataType.String),
				},
				TableName:    tableName,
				ColumnName:   columnName.String,
				ExpectedType: expected.dataType,
				ActualType:   dataType.String,
				Issue:        "incorrect data type",
			}
		}

		// Check nullable constraint
		if isNullable.String != expected.isNullable {
			return &TableStructureError{
				EventStoreError: EventStoreError{
					Op:  "validate_table_structure",
					Err: fmt.Errorf("column '%s' should be %s, got %s", columnName.String, expected.isNullable, isNullable.String),
				},
				TableName:  tableName,
				ColumnName: columnName.String,
				Issue:      fmt.Sprintf("incorrect nullable constraint: expected %s, got %s", expected.isNullable, isNullable.String),
			}
		}

		// Check default value for occurred_at
		if columnName.String == "occurred_at" && expected.hasDefault {
			if !columnDefault.Valid {
				return &TableStructureError{
					EventStoreError: EventStoreError{
						Op:  "validate_table_structure",
						Err: fmt.Errorf("column 'occurred_at' should have a default value"),
					},
					TableName:  tableName,
					ColumnName: "occurred_at",
					Issue:      "missing default value",
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return &TableStructureError{
			EventStoreError: EventStoreError{
				Op:  "validate_table_structure",
				Err: fmt.Errorf("error iterating table columns: %w", err),
			},
			TableName: tableName,
			Issue:     "error iterating table columns",
		}
	}

	// Check that all expected columns were found
	for columnName := range expectedColumns {
		if !foundColumns[columnName] {
			return &TableStructureError{
				EventStoreError: EventStoreError{
					Op:  "validate_table_structure",
					Err: fmt.Errorf("missing required column '%s'", columnName),
				},
				TableName:  tableName,
				ColumnName: columnName,
				Issue:      "missing required column",
			}
		}
	}

	return nil
}

// Remove ReadWithOptions and Read methods (now in read.go)
