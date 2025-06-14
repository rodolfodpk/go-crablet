package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DumpEvents prints all events in the database
func DumpEvents(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT id, type, tags, data, position, causation_id, correlation_id, created_at
		FROM events 
		ORDER BY position
	`)
	if err != nil {
		log.Printf("Failed to query events: %v", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-8s %-20s %-30s %-50s %-8s %-20s %-20s %-20s\n",
		"Position", "Type", "Tags", "Data", "ID", "Causation ID", "Correlation ID", "Created At")
	fmt.Println(strings.Repeat("-", 180))

	for rows.Next() {
		var id, eventType, causationID, correlationID string
		var tags, data []byte
		var position int64
		var createdAt time.Time

		err := rows.Scan(&id, &eventType, &tags, &data, &position, &causationID, &correlationID, &createdAt)
		if err != nil {
			log.Printf("Failed to scan event: %v", err)
			continue
		}

		// Truncate long fields for display
		tagsStr := string(tags)
		dataStr := string(data)
		if len(tagsStr) > 28 {
			tagsStr = tagsStr[:25] + "..."
		}
		if len(dataStr) > 48 {
			dataStr = dataStr[:45] + "..."
		}
		if len(id) > 18 {
			id = id[:15] + "..."
		}
		if len(causationID) > 18 {
			causationID = causationID[:15] + "..."
		}
		if len(correlationID) > 18 {
			correlationID = correlationID[:15] + "..."
		}

		fmt.Printf("%-8d %-20s %-30s %-50s %-8s %-20s %-20s %-20s\n",
			position, eventType, tagsStr, dataStr, id, causationID, correlationID, createdAt.Format("15:04:05"))
	}
}
