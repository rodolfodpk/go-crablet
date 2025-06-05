package dcb_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go-crablet/internal/dcb"
	"io"
	"os"
	"testing"
	"time"
)

func TestEventStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventStore Integration Suite")
}

var (
	ctx      context.Context
	pool     *pgxpool.Pool
	teardown func()
	store    dcb.EventStore
)

var _ = BeforeSuite(func() {
	ctx = context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_USER":     "user",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred())

	host, err := postgresC.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	port, err := postgresC.MappedPort(ctx, "5432")
	Expect(err).NotTo(HaveOccurred())

	dsn := fmt.Sprintf("postgres://user:secret@%s:%s/testdb?sslmode=disable", host, port.Port())
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	// Configure prepared statement cache settings
	// In pgx v5, prepared statement behavior is different
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	// Optionally set the statement cache capacity (default is 512)
	poolConfig.ConnConfig.StatementCacheCapacity = 100

	pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		return pool.Ping(ctx)
	}, 10*time.Second, 200*time.Millisecond).Should(Succeed())

	// Use go:embed or another more robust path resolution approach
	projectRoot := "../.." // Go up two levels from internal/dcb to the project root
	schemaPath := projectRoot + "/docker-entrypoint-initdb.d/schema.sql"
	schema, err := os.ReadFile(schemaPath)
	Expect(err).NotTo(HaveOccurred())

	_, err = pool.Exec(ctx, string(schema))
	Expect(err).NotTo(HaveOccurred())

	store, err = dcb.NewEventStore(ctx, pool)
	Expect(err).NotTo(HaveOccurred())

	teardown = func() {
		// Attempt to retrieve and print container logs
		if postgresC != nil {
			logsReader, err := postgresC.Logs(ctx)
			if err == nil {
				defer logsReader.Close()
				logBytes, readErr := io.ReadAll(logsReader)
				if readErr == nil && len(logBytes) > 0 {
					GinkgoWriter.Printf("--- PostgreSQL Container Logs ---\n%s\n-------------------------------\n", string(logBytes))
				} else if readErr != nil {
					GinkgoWriter.Printf("--- Error reading PostgreSQL Container Logs: %v ---\n", readErr)
				} else {
					GinkgoWriter.Println("--- PostgreSQL Container Logs: No logs produced. ---")
				}
			} else {
				GinkgoWriter.Printf("--- Error retrieving PostgreSQL Container Logs stream: %v ---\n", err)
			}
		}

		if store != nil {
			store.Close()
		}
		if pool != nil {
			pool.Close()
		}
		if postgresC != nil {
			err := postgresC.Terminate(ctx)
			if err != nil {
				GinkgoWriter.Printf("--- Error terminating PostgreSQL Container: %v ---\n", err)
			}
		}
	}
})

var _ = AfterSuite(func() {
	if teardown != nil {
		teardown()
	}
})

var _ = Describe("EventStore", func() {

	BeforeEach(func() {
		// Truncate the events table and reset sequences before each test
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("AppendEvents scenarios", func() {

		It("appends events successfully", func() {
			tags := dcb.NewTags("course_id", "course1")
			query := dcb.NewQuery(tags)
			event := dcb.NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
			events := []dcb.InputEvent{event}
			pos, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(1)))

			reducer := dcb.StateReducer{
				InitialState: 0,
				ReducerFn: func(state any, e dcb.Event) any {
					return state.(int) + 1
				},
			}
			readPos, state, err := store.ProjectState(ctx, query, reducer)
			Expect(err).NotTo(HaveOccurred())
			Expect(readPos).To(Equal(int64(1)))
			Expect(state).To(Equal(1))

			dumpEvents(pool)
		})

		It("appends events with multiple tags", func() {
			tags := dcb.NewTags("course_id", "course1", "user_id", "user123", "action", "enroll")
			query := dcb.NewQuery(dcb.NewTags("course_id", "course1"))
			events := []dcb.InputEvent{
				dcb.NewInputEvent("Enrollment", tags, []byte(`{"action":"enrolled"}`)),
			}

			pos, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(1)))
			// Query by different tag combinations
			queryByUser := dcb.NewQuery(dcb.NewTags("user_id", "user123"))
			reducer := dcb.StateReducer{
				InitialState: 0,
				ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
			}

			_, state, err := store.ProjectState(ctx, queryByUser, reducer)
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(1))
			Expect(state).To(Equal(1))

			dumpEvents(pool)
		})
		It("appends multiple events in a batch", func() {

			tags := dcb.NewTags("course_id", "course2")
			query := dcb.NewQuery(tags)
			events := []dcb.InputEvent{
				dcb.NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)),
				dcb.NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L1"}`)),
				dcb.NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L2"}`)),
			}

			pos, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(3)))

			reducer := dcb.StateReducer{
				InitialState: 0,
				ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
			}
			_, state, err := store.ProjectState(ctx, query, reducer)
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(3))

			dumpEvents(pool)

		})
		It("fails with concurrency error when position is outdated", func() {

			tags := dcb.NewTags("course_id", "course3")
			query := dcb.NewQuery(tags)

			// First append - will succeed
			_, err := store.AppendEvents(ctx, []dcb.InputEvent{
				dcb.NewInputEvent("Initial", tags, []byte(`{"status":"first"}`)),
			}, query, 0)
			Expect(err).NotTo(HaveOccurred())

			// Second append with outdated position - should fail
			_, err = store.AppendEvents(ctx, []dcb.InputEvent{
				dcb.NewInputEvent("Second", tags, []byte(`{"status":"second"}`)),
			}, query, 0) // Using 0 again when it should be 1

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Consistency"))

			dumpEvents(pool)
		})
		It("properly sets causation and correlation IDs", func() {

			tags := dcb.NewTags("entity_id", "E1")
			query := dcb.NewQuery(tags)
			events := []dcb.InputEvent{
				dcb.NewInputEvent("EntityRegistered", tags, []byte(`{"initial":true}`)),
				dcb.NewInputEvent("EntityAttributeChanged", tags, []byte(`{"step":1}`)),
				dcb.NewInputEvent("EntityAttributeChanged", tags, []byte(`{"step":2}`)),
			}

			pos, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(3)))

			// Check that causation and correlation IDs were set correctly
			dumpEvents(pool)

			// Custom reducer to check causation and correlation IDs
			type EventRelationships struct {
				Count          int
				FirstID        string
				CausationIDs   []string
				CorrelationIDs []string
			}

			relationshipReducer := dcb.StateReducer{
				InitialState: EventRelationships{Count: 0},
				ReducerFn: func(state any, e dcb.Event) any {
					s := state.(EventRelationships)
					s.Count++
					if s.Count == 1 {
						s.FirstID = e.ID
					}
					s.CausationIDs = append(s.CausationIDs, e.CausationID)
					s.CorrelationIDs = append(s.CorrelationIDs, e.CorrelationID)
					return s
				},
			}

			_, state, err := store.ProjectState(ctx, query, relationshipReducer)
			Expect(err).NotTo(HaveOccurred())
			relationships := state.(EventRelationships)

			// First event is self-caused
			Expect(relationships.CausationIDs[0]).To(Equal(relationships.FirstID))

			// All events have same correlation ID (the first event's ID)
			for _, cid := range relationships.CorrelationIDs {
				Expect(cid).To(Equal(relationships.FirstID))
			}

			// Later events are caused by their predecessors
			Expect(relationships.CausationIDs[1]).To(Equal(relationships.FirstID))
			Expect(relationships.CausationIDs[2]).NotTo(Equal(relationships.FirstID))
		})

		Describe("AppendEvents error scenarios", func() {

			It("returns error when appending events with empty tags", func() {
				tags := dcb.NewTags() // Empty tags
				query := dcb.NewQuery(dcb.NewTags("course_id", "C1"))
				events := []dcb.InputEvent{
					dcb.NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`)),
				}
				_, err := store.AppendEvents(ctx, events, query, 0)
				Expect(err).To(HaveOccurred())
			})

			It("returns error when appending invalid JSON data", func() {
				tags := dcb.NewTags("course_id", "C1")
				query := dcb.NewQuery(tags)
				events := []dcb.InputEvent{
					dcb.NewInputEvent("Subscription", tags, []byte(`not-json`)),
				}
				_, err := store.AppendEvents(ctx, events, query, 0)
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("ProjectState scenarios", func() {
			BeforeEach(func() {
				// Set up some test events for all ProjectState tests
				courseTags := dcb.NewTags("course_id", "course101")
				userTags := dcb.NewTags("user_id", "user101")
				mixedTags := dcb.NewTags("course_id", "course101", "user_id", "user101")

				query := dcb.NewQuery(courseTags)

				// Insert different event types with different tag combinations
				events := []dcb.InputEvent{
					dcb.NewInputEvent("CourseLaunched", courseTags, []byte(`{"title":"Test Course"}`)),
					dcb.NewInputEvent("UserRegistered", userTags, []byte(`{"name":"Test User"}`)),
					dcb.NewInputEvent("Enrollment", mixedTags, []byte(`{"status":"active"}`)),
					dcb.NewInputEvent("CourseUpdated", courseTags, []byte(`{"title":"Updated Course"}`)),
				}

				pos, err := store.AppendEvents(ctx, events, query, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(pos).To(Equal(int64(4)))
			})

			It("reads state with empty tags in query", func() {
				emptyTagsQuery := dcb.NewQuery(dcb.NewTags())

				reducer := dcb.StateReducer{
					InitialState: 0,
					ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
				}

				_, state, err := store.ProjectState(ctx, emptyTagsQuery, reducer)
				// Should return all events since no tag filtering is applied
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(4)) // All 4 events should be read
			})

			It("reads state with specific tags but empty eventTypes", func() {
				courseQuery := dcb.NewQuery(dcb.NewTags("course_id", "course101"))
				// Not setting any event types

				reducer := dcb.StateReducer{
					InitialState: 0,
					ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
				}

				_, state, err := store.ProjectState(ctx, courseQuery, reducer)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(3)) // Should match CourseLaunched, Enrollment, and CourseUpdated
			})

			It("reads state with empty tags but specific eventTypes", func() {
				query := dcb.NewQuery(dcb.NewTags())
				query.EventTypes = []string{"CourseLaunched", "CourseUpdated"}

				reducer := dcb.StateReducer{
					InitialState: 0,
					ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
				}

				_, state, err := store.ProjectState(ctx, query, reducer)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(2)) // Should match only CourseLaunched and CourseUpdated
			})

			It("reads state with both empty tags and empty eventTypes", func() {
				query := dcb.NewQuery(dcb.NewTags())
				// Event types remain empty

				reducer := dcb.StateReducer{
					InitialState: 0,
					ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
				}

				_, state, err := store.ProjectState(ctx, query, reducer)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(4)) // Should match all events
			})
		})
	})
	Describe("ProjectStateUpTo scenarios", func() {
		BeforeEach(func() {
			// Set up sequential events for position testing
			tags := dcb.NewTags("sequence_id", "seq1")
			query := dcb.NewQuery(tags)
			events := []dcb.InputEvent{
				dcb.NewInputEvent("Event1", tags, []byte(`{"order":1}`)),
				dcb.NewInputEvent("Event2", tags, []byte(`{"order":2}`)),
				dcb.NewInputEvent("Event3", tags, []byte(`{"order":3}`)),
				dcb.NewInputEvent("Event4", tags, []byte(`{"order":4}`)),
				dcb.NewInputEvent("Event5", tags, []byte(`{"order":5}`)),
			}

			pos, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(5)))
		})

		It("reads state up to a specific position limit", func() {
			query := dcb.NewQuery(dcb.NewTags("sequence_id", "seq1"))

			// Define a reducer that counts events
			countReducer := dcb.StateReducer{
				InitialState: 0,
				ReducerFn: func(state any, e dcb.Event) any {
					return state.(int) + 1
				},
			}

			// Read up to position 3 (should include events at positions 1, 2, and 3)
			pos, state, err := store.ProjectStateUpTo(ctx, query, countReducer, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(3)))
			Expect(state).To(Equal(3))

			// Read all events (maxPosition = -1)
			pos, state, err = store.ProjectStateUpTo(ctx, query, countReducer, -1)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(5)))
			Expect(state).To(Equal(5))

			// Read up to position 0 (should find no events)
			pos, state, err = store.ProjectStateUpTo(ctx, query, countReducer, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(0)))
			Expect(state).To(Equal(0))
		})
	})
})

// Test scenarios for AppendEventsIfNotExists
var _ = Describe("AppendEventsIfNotExists", func() {
	BeforeEach(func() {
		// Truncate the events table before each test
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	})

	It("appends events when they don't exist", func() {
		tags := dcb.NewTags("entity_id", "E100")
		query := dcb.NewQuery(tags)
		events := []dcb.InputEvent{
			dcb.NewInputEvent("EntityCreated", tags, []byte(`{"name":"Test Entity"}`)),
		}

		// Define a simple reducer
		reducer := dcb.StateReducer{
			InitialState: nil,
			ReducerFn: func(state any, e dcb.Event) any {
				return e.Type
			},
		}

		pos, err := store.AppendEventsIfNotExists(ctx, events, query, 0, reducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))

		// Verify the event was added
		_, state, err := store.ProjectState(ctx, query, reducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal("EntityCreated"))
	})

	It("doesn't append events when they already exist", func() {
		tags := dcb.NewTags("entity_id", "E101")
		query := dcb.NewQuery(tags)
		events := []dcb.InputEvent{
			dcb.NewInputEvent("EntityCreated", tags, []byte(`{"name":"Test Entity"}`)),
		}

		// Define a reducer that simply returns a non-nil value if any event exists
		reducer := dcb.StateReducer{
			InitialState: nil,
			ReducerFn: func(state any, e dcb.Event) any {
				return true
			},
		}

		// First append should succeed
		pos1, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos1).To(Equal(int64(1)))

		// AppendEventsIfNotExists should not append and return the existing position
		pos2, err := store.AppendEventsIfNotExists(ctx, events, query, pos1, reducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos2).To(Equal(pos1))

		// Verify only one event exists
		countReducer := dcb.StateReducer{
			InitialState: 0,
			ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
		}
		_, count, err := store.ProjectState(ctx, query, countReducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(1))
	})

	It("handles complex state checking before append", func() {
		tags := dcb.NewTags("order_id", "O123")
		query := dcb.NewQuery(tags)

		// Define a reducer that checks for specific event types
		type OrderState struct {
			IsProcessed bool
		}

		reducer := dcb.StateReducer{
			InitialState: &OrderState{IsProcessed: false},
			ReducerFn: func(state any, e dcb.Event) any {
				orderState := state.(*OrderState)
				if e.Type == "OrderProcessed" {
					orderState.IsProcessed = true
				}
				return orderState
			},
		}

		// First add an order created event
		pos1, err := store.AppendEvents(ctx, []dcb.InputEvent{
			dcb.NewInputEvent("OrderCreated", tags, []byte(`{"amount":100}`)),
		}, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos1).To(Equal(int64(1)))

		// Verify state before the conditional append
		_, state1, err := store.ProjectState(ctx, query, reducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(state1.(*OrderState).IsProcessed).To(BeFalse())

		// Try to append "OrderProcessed" conditionally
		processingEvents := []dcb.InputEvent{
			dcb.NewInputEvent("OrderProcessed", tags, []byte(`{"status":"complete"}`)),
		}

		// This should append since the order isn't processed yet
		pos, err := store.AppendEvents(ctx, processingEvents, query, pos1)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(2)))

		// Now the state should show the order is processed
		_, state2, err := store.ProjectState(ctx, query, reducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(state2.(*OrderState).IsProcessed).To(BeTrue())

		// Verify only 2 events total
		dumpEvents(pool)
		countReducer := dcb.StateReducer{
			InitialState: 0,
			ReducerFn:    func(state any, e dcb.Event) any { return state.(int) + 1 },
		}
		_, count, err := store.ProjectState(ctx, query, countReducer)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(2))
	})
})

// dumpEvents queries the events table and prints the results as JSON
func dumpEvents(pool *pgxpool.Pool) {

	rows, err := pool.Query(ctx, `
		SELECT id, type, position, tags, data, causation_id, correlation_id
		FROM events
		ORDER BY position
	`)
	Expect(err).NotTo(HaveOccurred())
	defer rows.Close()

	type EventRecord struct {
		ID            string      `json:"id"`
		Type          string      `json:"type"`
		Position      int64       `json:"position"`
		Tags          interface{} `json:"tags"`
		Data          interface{} `json:"data"`
		CausationID   string      `json:"causation_id"`
		CorrelationID string      `json:"correlation_id"`
	}

	events := []EventRecord{}
	for rows.Next() {
		var (
			id            string
			eventType     string
			position      int64
			tagsBytes     []byte
			dataBytes     []byte
			causationID   string
			correlationID string
		)

		err := rows.Scan(&id, &eventType, &position, &tagsBytes, &dataBytes, &causationID, &correlationID)
		Expect(err).NotTo(HaveOccurred())

		var tags interface{}
		err = json.Unmarshal(tagsBytes, &tags)
		Expect(err).NotTo(HaveOccurred())

		var data interface{}
		err = json.Unmarshal(dataBytes, &data)
		Expect(err).NotTo(HaveOccurred())

		events = append(events, EventRecord{
			ID:            id,
			Type:          eventType,
			Position:      position,
			Tags:          tags,
			Data:          data,
			CausationID:   causationID,
			CorrelationID: correlationID,
		})
	}
	Expect(rows.Err()).NotTo(HaveOccurred())

	// Convert events to JSON
	jsonData, err := json.MarshalIndent(events, "", "  ")
	Expect(err).NotTo(HaveOccurred())

	// Print the JSON data
	GinkgoWriter.Println("--- Events Table Contents (JSON) ---")
	GinkgoWriter.Println(string(jsonData))
	GinkgoWriter.Printf("Total events: %d\n", len(events))
	GinkgoWriter.Println("------------------------------------")

	// Also print to standard output to ensure visibility
	fmt.Println("--- Events Table Contents (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Printf("Total events: %d\n", len(events))
	fmt.Println("------------------------------------")

}
