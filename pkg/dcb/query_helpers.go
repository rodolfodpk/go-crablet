package dcb

import (
	"sort"
	"strings"
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
