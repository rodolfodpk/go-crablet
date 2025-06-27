package dcb

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ToArray converts a slice of Tags to a PostgreSQL TEXT[] array
func TagsToArray(tags []Tag) []string {
	if len(tags) == 0 {
		return []string{}
	}

	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Key + ":" + tag.Value
	}

	// Sort for consistent ordering
	sort.Strings(result)
	return result
}

// ParseTagsArray converts a PostgreSQL TEXT[] array back to a slice of Tags
func ParseTagsArray(arr []string) []Tag {
	if len(arr) == 0 {
		return []Tag{}
	}

	tags := make([]Tag, 0, len(arr))
	for _, item := range arr {
		if item == "" {
			continue
		}

		// Split on first ":" only to handle values with colons
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := parts[1] // Keep original value (including colons)
			if key != "" {
				tags = append(tags, Tag{Key: key, Value: value})
			}
		}
	}
	return tags
}

// ConnectionPoolHealth represents the health status of a connection pool
type ConnectionPoolHealth struct {
	TotalConns        int32
	IdleConns         int32
	AcquiredConns     int32
	ConstructingConns int32
	Healthy           bool
	Message           string
}

// CheckConnectionPoolHealth checks the health of a connection pool
func CheckConnectionPoolHealth(pool *pgxpool.Pool) ConnectionPoolHealth {
	stats := pool.Stat()

	health := ConnectionPoolHealth{
		TotalConns:        stats.TotalConns(),
		IdleConns:         stats.IdleConns(),
		AcquiredConns:     stats.AcquiredConns(),
		ConstructingConns: stats.ConstructingConns(),
		Healthy:           true,
	}

	// Check for potential connection leaks (high acquired connections)
	if stats.AcquiredConns() > stats.TotalConns()*80/100 {
		health.Healthy = false
		health.Message = "High number of acquired connections - potential connection leak"
	}

	// Check for no idle connections available
	if stats.IdleConns() == 0 && stats.AcquiredConns() > 0 {
		health.Healthy = false
		health.Message = "No idle connections available - pool may be exhausted"
	}

	// Check for connections being constructed (normal during startup/load)
	if stats.ConstructingConns() > 0 {
		if stats.ConstructingConns() > 5 {
			health.Healthy = false
			health.Message = fmt.Sprintf("High number of connections being constructed: %d", stats.ConstructingConns())
		} else {
			health.Message = fmt.Sprintf("Building %d new connections", stats.ConstructingConns())
		}
	}

	return health
}

// LogConnectionPoolHealth logs the health status of a connection pool
func LogConnectionPoolHealth(pool *pgxpool.Pool, operation string) {
	health := CheckConnectionPoolHealth(pool)
	if health.Healthy {
		log.Printf("[POOL HEALTH] %s: Healthy - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
			operation, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)
	} else {
		log.Printf("[POOL HEALTH] %s: UNHEALTHY - %s - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
			operation, health.Message, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)
	}
}

// LogConnectionPoolHealthDebug logs detailed health information for debugging
func LogConnectionPoolHealthDebug(pool *pgxpool.Pool, operation string) {
	health := CheckConnectionPoolHealth(pool)
	log.Printf("[POOL DEBUG] %s: %s - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
		operation, health.Message, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)
}

// LogConnectionPoolHealthWithLevel logs health with a custom log level
func LogConnectionPoolHealthWithLevel(pool *pgxpool.Pool, operation string, level string) {
	health := CheckConnectionPoolHealth(pool)
	message := fmt.Sprintf("[POOL %s] %s: %s - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
		strings.ToUpper(level), operation, health.Message, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)

	switch strings.ToLower(level) {
	case "error":
		log.Printf("[ERROR] %s", message)
	case "warn", "warning":
		log.Printf("[WARN] %s", message)
	case "debug":
		log.Printf("[DEBUG] %s", message)
	default:
		log.Printf("[INFO] %s", message)
	}
}
