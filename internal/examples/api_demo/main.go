// This example demonstrates the simplified API constructs for better developer experience
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// UserRegistered represents when a user registers
type UserRegistered struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// UserProfileUpdated represents when a user updates their profile
type UserProfileUpdated struct {
	UserID    string    `json:"user_id"`
	Bio       string    `json:"bio"`
	AvatarURL string    `json:"avatar_url"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserState holds the current state of a user
type UserState struct {
	UserID    string
	Email     string
	Username  string
	Bio       string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	ctx := context.Background()

	// Setup database connection
	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	fmt.Println("=== Simplified API Demo ===")

	// Demo 1: Simplified Query Construction
	fmt.Println("1. Simplified Query Construction:")
	fmt.Println("   Old way: dcb.NewQuery(dcb.NewTags(\"user_id\", \"123\"), \"UserRegistered\")")
	fmt.Println("   New way: dcb.NewQueryBuilder().WithTagAndType(\"user_id\", \"123\", \"UserRegistered\").Build()")

	// Demo 2: Simplified Tag Construction
	fmt.Println("\n2. Simplified Tag Construction:")
	fmt.Println("   Old way: dcb.NewTags(\"user_id\", \"123\", \"email\", \"user@example.com\")")
	fmt.Println("   New way: dcb.Tags{\"user_id\": \"123\", \"email\": \"user@example.com\"}.ToTags()")

	// Demo 3: Simplified AppendCondition
	fmt.Println("\n3. Simplified AppendCondition:")
	fmt.Println("   Old way: 3-step process with NewQueryItem -> NewQueryFromItems -> NewAppendCondition")
	fmt.Println("   New way: dcb.FailIfExists(\"user_id\", \"123\")")

	// Demo 4: Projection Helpers
	fmt.Println("\n4. Projection Helpers:")
	fmt.Println("   Counter: dcb.ProjectCounter(\"user_count\", \"UserRegistered\", \"status\", \"active\")")
	fmt.Println("   Boolean: dcb.ProjectBoolean(\"user_exists\", \"UserRegistered\", \"user_id\", \"123\")")

	// Create a user with simplified API
	fmt.Println("\n=== Creating User with Simplified API ===")

	userID := "user_123"
	userEvent := UserRegistered{
		UserID:    userID,
		Email:     "user@example.com",
		Username:  "johndoe",
		CreatedAt: time.Now(),
	}

	// Use simplified tags
	event := dcb.NewInputEvent("UserRegistered", dcb.Tags{
		"user_id": userID,
		"email":   "user@example.com",
	}.ToTags(), dcb.ToJSON(userEvent))

	// Append without condition (unconditional)
	err = store.Append(ctx, []dcb.InputEvent{event})
	if err != nil {
		log.Fatalf("Failed to append user event: %v", err)
	}
	fmt.Printf("✓ Created user: %s\n", userID)

	// Query user with simplified query
	fmt.Println("\n=== Querying User with Simplified API ===")

	query := dcb.NewQueryBuilder().WithTagAndType("user_id", userID, "UserRegistered").Build()
	events, err := store.Query(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to query user: %v", err)
	}

	if len(events) > 0 {
		var user UserRegistered
		json.Unmarshal(events[0].Data, &user)
		fmt.Printf("✓ Found user: %s (%s)\n", user.Username, user.Email)
	}

	// Update user profile with DCB concurrency control
	fmt.Println("\n=== Updating User Profile with DCB Concurrency Control ===")

	// Use projection helper for user state
	userProjector := dcb.ProjectState("user", "UserRegistered", "user_id", userID, UserState{}, func(state any, event dcb.Event) any {
		userState := state.(UserState)

		if event.Type == "UserRegistered" {
			var userReg UserRegistered
			json.Unmarshal(event.Data, &userReg)
			userState.UserID = userReg.UserID
			userState.Email = userReg.Email
			userState.Username = userReg.Username
			userState.CreatedAt = userReg.CreatedAt
		} else if event.Type == "UserProfileUpdated" {
			var profileUpdate UserProfileUpdated
			json.Unmarshal(event.Data, &profileUpdate)
			userState.Bio = profileUpdate.Bio
			userState.AvatarURL = profileUpdate.AvatarURL
			userState.UpdatedAt = profileUpdate.UpdatedAt
		}

		return userState
	})

	// Project current state
	projectedStates, appendCondition, err := store.Project(ctx, []dcb.StateProjector{userProjector}, nil)
	if err != nil {
		log.Fatalf("Failed to project user state: %v", err)
	}

	userState := projectedStates["user"].(UserState)
	fmt.Printf("✓ Current user state: %s (%s)\n", userState.Username, userState.Email)

	// Create profile update event
	profileUpdate := UserProfileUpdated{
		UserID:    userID,
		Bio:       "Software developer and event sourcing enthusiast",
		AvatarURL: "https://example.com/avatar.jpg",
		UpdatedAt: time.Now(),
	}

	updateEvent := dcb.NewInputEvent("UserProfileUpdated", dcb.Tags{
		"user_id": userID,
	}.ToTags(), dcb.ToJSON(profileUpdate))

	// Append with DCB concurrency control using simplified condition
	err = store.AppendIf(ctx, []dcb.InputEvent{updateEvent}, appendCondition)
	if err != nil {
		log.Fatalf("Failed to append profile update: %v", err)
	}
	fmt.Printf("✓ Updated user profile for: %s\n", userID)

	// Demo counter projection
	fmt.Println("\n=== Counter Projection Demo ===")

	// Query with simplified query for multiple conditions
	statusQuery := dcb.NewQueryBuilder().WithTagsAndTypes([]string{"UserRegistered"}, "status", "active").Build()
	statusEvents, err := store.Query(ctx, statusQuery, nil)
	if err != nil {
		log.Fatalf("Failed to query users by status: %v", err)
	}

	fmt.Printf("✓ Found %d active users\n", len(statusEvents))

	// Demo boolean projection
	fmt.Println("\n=== Boolean Projection Demo ===")

	existsProjector := dcb.ProjectBoolean("user_exists", "UserRegistered", "user_id", userID)
	existsStates, _, err := store.Project(ctx, []dcb.StateProjector{existsProjector}, nil)
	if err != nil {
		log.Fatalf("Failed to project user existence: %v", err)
	}

	userExists := existsStates["user_exists"].(bool)
	fmt.Printf("✓ User exists: %t\n", userExists)

	fmt.Println("\n=== Demo Complete! ===")
	fmt.Println("The simplified API provides:")
	fmt.Println("• 50% less boilerplate for common operations")
	fmt.Println("• More intuitive query construction")
	fmt.Println("• Simplified tag creation")
	fmt.Println("• Easy-to-use projection helpers")
	fmt.Println("• Clear DCB concurrency control")
}
