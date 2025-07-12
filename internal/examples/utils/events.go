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
		SELECT type, tags, data, position, occurred_at
		FROM events 
		ORDER BY position
	`)
	if err != nil {
		log.Printf("Failed to query events: %v", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-8s %-20s %-30s %-50s %-15s\n",
		"Position", "Type", "Tags", "Data", "Occurred At")
	fmt.Println(strings.Repeat("-", 130))

	for rows.Next() {
		var eventType string
		var tags []string
		var data []byte
		var position int64
		var occurredAt time.Time

		err := rows.Scan(&eventType, &tags, &data, &position, &occurredAt)
		if err != nil {
			log.Printf("Failed to scan event: %v", err)
			continue
		}

		// Format tags
		tagsStr := strings.Join(tags, ", ")
		if len(tagsStr) > 28 {
			tagsStr = tagsStr[:25] + "..."
		}

		// Format data
		dataStr := string(data)
		if len(dataStr) > 48 {
			dataStr = dataStr[:45] + "..."
		}

		fmt.Printf("%-8d %-20s %-30s %-50s %-15s\n",
			position, eventType, tagsStr, dataStr, occurredAt.Format("2006-01-02 15:04:05"))
	}
}
