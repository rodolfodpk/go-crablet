package dcb

import (
	"regexp"
	"sort"
	"strings"

	"go.jetify.com/typeid"
)

var typeidUUIDPattern = regexp.MustCompile(`^[0-9a-z]{26}$`)

// generateTagBasedTypeID creates a TypeID using sorted tag keys as prefix
// The prefix is truncated to fit within VARCHAR(64) including the UUID part
func generateTagBasedTypeID(tags []Tag) string {
	// Sort tag keys alphabetically for consistency
	keys := make([]string, len(tags))
	for i, tag := range tags {
		keys[i] = tag.Key
	}
	sort.Strings(keys)

	// Build prefix from sorted tag keys
	prefix := strings.Join(keys, "_")

	// Truncate to fit in VARCHAR(64) with UUID part
	// TypeID format: prefix_01h2xcejqtf2nbrexx3vqjhp41 (26 chars for UUID + 1 underscore)
	maxPrefixLength := 64 - 26 - 1 // 64 total - 26 UUID chars - 1 underscore = 37 chars
	if len(prefix) > maxPrefixLength {
		prefix = prefix[:maxPrefixLength]
	}

	// Generate TypeID with tag-based prefix
	tid, err := typeid.WithPrefix(prefix)
	if err != nil {
		// Fallback to default TypeID if prefix is invalid
		tid, _ = typeid.WithPrefix("event")
	}
	return tid.String()
}

// extractUUIDFromTypeID extracts UUID part from TypeID if present
// This is used when users provide TypeID values in their tags
func extractUUIDFromTypeID(typeID string) string {
	parts := strings.Split(typeID, "_")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if typeidUUIDPattern.MatchString(last) {
			return last // Only return if it matches TypeID UUID format
		}
	}
	return typeID // Fallback if not a valid TypeID
}

// sanitizeForTypeID sanitizes a string for use in TypeID prefix
// Replaces spaces and special chars with underscores, converts to lowercase
func sanitizeForTypeID(s string) string {
	// Convert to lowercase and replace special chars with underscores
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, strings.ToLower(s))

	// Remove consecutive underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	return sanitized
}
