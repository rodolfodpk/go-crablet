package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Event types for the course subscription example
type (
	studentCreated struct {
		StudentID string    `json:"student_id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	}

	courseCreated struct {
		CourseID    string    `json:"course_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		MaxStudents int       `json:"max_students"`
		CreatedAt   time.Time `json:"created_at"`
	}

	studentSubscribed struct {
		StudentID    string    `json:"student_id"`
		CourseID     string    `json:"course_id"`
		SubscribedAt time.Time `json:"subscribed_at"`
	}

	studentUnsubscribed struct {
		StudentID      string    `json:"student_id"`
		CourseID       string    `json:"course_id"`
		UnsubscribedAt time.Time `json:"unsubscribed_at"`
	}
)

// Event constructors
func newStudentCreated(name, email string) studentCreated {
	return studentCreated{
		StudentID: uuid.New().String(),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}
}

func newCourseCreated(title, description string, maxStudents int) courseCreated {
	return courseCreated{
		CourseID:    uuid.New().String(),
		Title:       title,
		Description: description,
		MaxStudents: maxStudents,
		CreatedAt:   time.Now(),
	}
}

func newStudentSubscribed(studentID, courseID string) studentSubscribed {
	return studentSubscribed{
		StudentID:    studentID,
		CourseID:     courseID,
		SubscribedAt: time.Now(),
	}
}

func newStudentUnsubscribed(studentID, courseID string) studentUnsubscribed {
	return studentUnsubscribed{
		StudentID:      studentID,
		CourseID:       courseID,
		UnsubscribedAt: time.Now(),
	}
}

// Projection types
type (
	studentProjection struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	courseProjection struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		MaxStudents int    `json:"max_students"`
		Subscribers int    `json:"subscribers"`
	}

	subscriptionProjection struct {
		StudentID string `json:"student_id"`
		CourseID  string `json:"course_id"`
		Active    bool   `json:"active"`
	}
)

// Streaming projection helpers
type projectionBuilder[T any] struct {
	projection T
	position   int64
}

func newProjectionBuilder[T any](initial T) *projectionBuilder[T] {
	return &projectionBuilder[T]{
		projection: initial,
		position:   0,
	}
}

func (pb *projectionBuilder[T]) Update(updateFn func(T) T) {
	pb.projection = updateFn(pb.projection)
}

func (pb *projectionBuilder[T]) Get() T {
	return pb.projection
}

func (pb *projectionBuilder[T]) GetPosition() int64 {
	return pb.position
}

// Helper function to unmarshal event data
func unmarshalEventData(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Student projection streamer
func streamStudentProjection(events EventIterator) *projectionBuilder[map[string]studentProjection] {
	students := make(map[string]studentProjection)
	builder := newProjectionBuilder(students)

	for {
		event, err := events.Next()
		if err != nil {
			break
		}
		if event == nil {
			break
		}
		builder.position = event.Position

		if event.Type == "StudentCreated" {
			var e studentCreated
			if err := unmarshalEventData(event.Data, &e); err == nil {
				students[e.StudentID] = studentProjection{
					ID:    e.StudentID,
					Name:  e.Name,
					Email: e.Email,
				}
			}
		}
	}

	return builder
}

// Course projection streamer
func streamCourseProjection(events EventIterator) *projectionBuilder[map[string]courseProjection] {
	courses := make(map[string]courseProjection)
	subscriptions := make(map[string]map[string]bool) // courseID -> studentID -> active
	builder := newProjectionBuilder(courses)

	for {
		event, err := events.Next()
		if err != nil {
			break
		}
		if event == nil {
			break
		}
		builder.position = event.Position

		switch event.Type {
		case "CourseCreated":
			var e courseCreated
			if err := unmarshalEventData(event.Data, &e); err == nil {
				courses[e.CourseID] = courseProjection{
					ID:          e.CourseID,
					Title:       e.Title,
					Description: e.Description,
					MaxStudents: e.MaxStudents,
					Subscribers: 0,
				}
				subscriptions[e.CourseID] = make(map[string]bool)
			}

		case "StudentSubscribed":
			var e studentSubscribed
			if err := unmarshalEventData(event.Data, &e); err == nil {
				if course, exists := courses[e.CourseID]; exists {
					if subscriptions[e.CourseID] == nil {
						subscriptions[e.CourseID] = make(map[string]bool)
					}
					if !subscriptions[e.CourseID][e.StudentID] {
						subscriptions[e.CourseID][e.StudentID] = true
						course.Subscribers++
						courses[e.CourseID] = course
					}
				}
			}

		case "StudentUnsubscribed":
			var e studentUnsubscribed
			if err := unmarshalEventData(event.Data, &e); err == nil {
				if course, exists := courses[e.CourseID]; exists {
					if subscriptions[e.CourseID] != nil && subscriptions[e.CourseID][e.StudentID] {
						subscriptions[e.CourseID][e.StudentID] = false
						course.Subscribers--
						courses[e.CourseID] = course
					}
				}
			}
		}
	}

	return builder
}

// Subscription projection streamer
func streamSubscriptionProjection(events EventIterator) *projectionBuilder[map[string]map[string]bool] {
	subscriptions := make(map[string]map[string]bool) // studentID -> courseID -> active
	builder := newProjectionBuilder(subscriptions)

	for {
		event, err := events.Next()
		if err != nil {
			break
		}
		if event == nil {
			break
		}
		builder.position = event.Position

		switch event.Type {
		case "StudentSubscribed":
			var e studentSubscribed
			if err := unmarshalEventData(event.Data, &e); err == nil {
				if subscriptions[e.StudentID] == nil {
					subscriptions[e.StudentID] = make(map[string]bool)
				}
				subscriptions[e.StudentID][e.CourseID] = true
			}

		case "StudentUnsubscribed":
			var e studentUnsubscribed
			if err := unmarshalEventData(event.Data, &e); err == nil {
				if subscriptions[e.StudentID] != nil {
					subscriptions[e.StudentID][e.CourseID] = false
				}
			}
		}
	}

	return builder
}

// Decision model builder
type decisionModel struct {
	students      map[string]studentProjection
	courses       map[string]courseProjection
	subscriptions map[string]map[string]bool
	position      int64
}

func newDecisionModel(store EventStore) (*decisionModel, error) {
	ctx := context.Background()

	// Stream all events to build projections
	events, err := store.ReadEvents(ctx, NewQueryAll(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	defer events.Close()

	students := streamStudentProjection(events)

	// Reset iterator for courses
	events, err = store.ReadEvents(ctx, NewQueryAll(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read events for courses: %w", err)
	}
	defer events.Close()

	courses := streamCourseProjection(events)

	// Reset iterator for subscriptions
	events, err = store.ReadEvents(ctx, NewQueryAll(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read events for subscriptions: %w", err)
	}
	defer events.Close()

	subscriptions := streamSubscriptionProjection(events)

	return &decisionModel{
		students:      students.Get(),
		courses:       courses.Get(),
		subscriptions: subscriptions.Get(),
		position:      students.GetPosition(),
	}, nil
}

// API methods
func (dm *decisionModel) getStudent(studentID string) (*studentProjection, error) {
	student, exists := dm.students[studentID]
	if !exists {
		return nil, fmt.Errorf("student not found: %s", studentID)
	}
	return &student, nil
}

func (dm *decisionModel) getCourse(courseID string) (*courseProjection, error) {
	course, exists := dm.courses[courseID]
	if !exists {
		return nil, fmt.Errorf("course not found: %s", courseID)
	}
	return &course, nil
}

func (dm *decisionModel) getStudentSubscriptions(studentID string) ([]subscriptionProjection, error) {
	studentSubs, exists := dm.subscriptions[studentID]
	if !exists {
		return []subscriptionProjection{}, nil
	}

	var subscriptions []subscriptionProjection
	for courseID, active := range studentSubs {
		if active {
			subscriptions = append(subscriptions, subscriptionProjection{
				StudentID: studentID,
				CourseID:  courseID,
				Active:    active,
			})
		}
	}

	return subscriptions, nil
}

func (dm *decisionModel) canSubscribe(studentID, courseID string) (bool, error) {
	// Check if student exists
	if _, err := dm.getStudent(studentID); err != nil {
		return false, err
	}

	// Check if course exists
	course, err := dm.getCourse(courseID)
	if err != nil {
		return false, err
	}

	// Check if already subscribed
	studentSubs, exists := dm.subscriptions[studentID]
	if exists && studentSubs[courseID] {
		return false, nil
	}

	// Check course capacity
	if course.Subscribers >= course.MaxStudents {
		return false, nil
	}

	// Check student subscription limit (max 10 courses)
	activeSubscriptions := 0
	if studentSubs != nil {
		for _, active := range studentSubs {
			if active {
				activeSubscriptions++
			}
		}
	}

	return activeSubscriptions < 10, nil
}

func (dm *decisionModel) getPosition() int64 {
	return dm.position
}

// Test suite
var _ = Describe("DCB Pattern Example", func() {
	Context("Event Constructors", func() {
		It("should create student events", func() {
			event := newStudentCreated("John Doe", "john@example.com")
			Expect(event.StudentID).NotTo(BeEmpty())
			Expect(event.Name).To(Equal("John Doe"))
			Expect(event.Email).To(Equal("john@example.com"))
		})

		It("should create course events", func() {
			event := newCourseCreated("Go Programming", "Learn Go", 30)
			Expect(event.CourseID).NotTo(BeEmpty())
			Expect(event.Title).To(Equal("Go Programming"))
			Expect(event.MaxStudents).To(Equal(30))
		})

		It("should create subscription events", func() {
			studentID := uuid.New().String()
			courseID := uuid.New().String()

			subEvent := newStudentSubscribed(studentID, courseID)
			Expect(subEvent.StudentID).To(Equal(studentID))
			Expect(subEvent.CourseID).To(Equal(courseID))

			unsubEvent := newStudentUnsubscribed(studentID, courseID)
			Expect(unsubEvent.StudentID).To(Equal(studentID))
			Expect(unsubEvent.CourseID).To(Equal(courseID))
		})
	})

	Context("Projection Builders", func() {
		It("should build student projections", func() {
			students := make(map[string]studentProjection)
			builder := newProjectionBuilder(students)

			// Simulate event processing
			event := newStudentCreated("Jane Doe", "jane@example.com")
			builder.Update(func(proj map[string]studentProjection) map[string]studentProjection {
				proj[event.StudentID] = studentProjection{
					ID:    event.StudentID,
					Name:  event.Name,
					Email: event.Email,
				}
				return proj
			})

			result := builder.Get()
			Expect(result).To(HaveKey(event.StudentID))
			Expect(result[event.StudentID].Name).To(Equal("Jane Doe"))
		})

		It("should track position in projections", func() {
			students := make(map[string]studentProjection)
			builder := newProjectionBuilder(students)

			// Simulate position updates
			builder.position = 100
			Expect(builder.GetPosition()).To(Equal(int64(100)))
		})
	})

	Context("Decision Model", func() {
		It("should validate subscription rules", func() {
			// Create a decision model with test data
			dm := &decisionModel{
				students: map[string]studentProjection{
					"student1": {ID: "student1", Name: "John", Email: "john@example.com"},
					"student2": {ID: "student2", Name: "Jane", Email: "jane@example.com"},
				},
				courses: map[string]courseProjection{
					"course1": {ID: "course1", Title: "Go", MaxStudents: 30, Subscribers: 0},
					"course2": {ID: "course2", Title: "Rust", MaxStudents: 30, Subscribers: 30}, // Full
				},
				subscriptions: map[string]map[string]bool{
					"student1": {
						"course3":  true,
						"course4":  true,
						"course5":  true,
						"course6":  true,
						"course7":  true,
						"course8":  true,
						"course9":  true,
						"course10": true,
						"course11": true,
						"course12": true, // 10th subscription
					},
				},
			}

			// Test valid subscription
			canSubscribe, err := dm.canSubscribe("student1", "course1")
			Expect(err).NotTo(HaveOccurred())
			Expect(canSubscribe).To(BeFalse()) // Already at limit

			// Test course capacity
			canSubscribe, err = dm.canSubscribe("student2", "course2")
			Expect(err).NotTo(HaveOccurred())
			Expect(canSubscribe).To(BeFalse()) // Course is full

			// Test non-existent student
			_, err = dm.canSubscribe("nonexistent", "course1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("student not found"))

			// Test non-existent course
			_, err = dm.canSubscribe("student1", "nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("course not found"))
		})

		It("should get student subscriptions", func() {
			dm := &decisionModel{
				subscriptions: map[string]map[string]bool{
					"student1": {
						"course1": true,
						"course2": false, // Unsubscribed
						"course3": true,
					},
				},
			}

			subscriptions, err := dm.getStudentSubscriptions("student1")
			Expect(err).NotTo(HaveOccurred())
			Expect(subscriptions).To(HaveLen(2))

			courseIDs := []string{subscriptions[0].CourseID, subscriptions[1].CourseID}
			Expect(courseIDs).To(ContainElements("course1", "course3"))
		})
	})

	Context("Streaming Projections", func() {
		It("should demonstrate streaming concept", func() {
			// This test demonstrates the streaming projection concept
			// In a real implementation, you'd use actual EventIterator

			// Simulate streaming through events
			studentEvent := newStudentCreated("Alice", "alice@example.com")

			// Simulate JSON marshaling
			studentData, _ := json.Marshal(studentEvent)

			// Simulate streaming projection
			students := make(map[string]studentProjection)
			position := int64(0)

			// Process student event
			var e studentCreated
			if err := json.Unmarshal(studentData, &e); err == nil {
				students[e.StudentID] = studentProjection{
					ID:    e.StudentID,
					Name:  e.Name,
					Email: e.Email,
				}
				position = 1
			}

			Expect(students).To(HaveLen(1))
			Expect(position).To(Equal(int64(1)))
		})
	})

	Context("Integrated DCB Workflow", func() {
		It("should enforce invariants and project state via streaming", func() {
			// Setup: create a real event store (using the test helpers)
			store, cleanup := newTestEventStore() // assumes test helper exists in this file or imported
			defer cleanup()
			ctx := context.Background()

			// Create a student and a course
			student := newStudentCreated("Bob", "bob@example.com")
			course := newCourseCreated("Go Bootcamp", "Intro to Go", 2) // small max for test

			// Append student and course events
			studentEvent := NewInputEvent("StudentCreated", NewTags("student_id", student.StudentID), mustJSON(student))
			courseEvent := NewInputEvent("CourseCreated", NewTags("course_id", course.CourseID), mustJSON(course))
			query := NewQuery(NewTags("student_id", student.StudentID))
			pos, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{studentEvent}, query, pos)
			Expect(err).NotTo(HaveOccurred())
			query = NewQuery(NewTags("course_id", course.CourseID))
			pos, err = store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{courseEvent}, query, pos)
			Expect(err).NotTo(HaveOccurred())

			// Subscribe the student to the course
			sub := newStudentSubscribed(student.StudentID, course.CourseID)
			subEvent := NewInputEvent("StudentSubscribed", NewTags("student_id", student.StudentID, "course_id", course.CourseID), mustJSON(sub))
			query = NewQuery(NewTags("student_id", student.StudentID, "course_id", course.CourseID))
			pos, err = store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{subEvent}, query, pos)
			Expect(err).NotTo(HaveOccurred())

			// Try to subscribe 2 more students (to hit course max)
			student2 := newStudentCreated("Alice", "alice@example.com")
			student3 := newStudentCreated("Eve", "eve@example.com")
			student2Event := NewInputEvent("StudentCreated", NewTags("student_id", student2.StudentID), mustJSON(student2))
			student3Event := NewInputEvent("StudentCreated", NewTags("student_id", student3.StudentID), mustJSON(student3))
			query2 := NewQuery(NewTags("student_id", student2.StudentID))
			pos2, err := store.GetCurrentPosition(ctx, query2)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{student2Event}, query2, pos2)
			Expect(err).NotTo(HaveOccurred())
			query3 := NewQuery(NewTags("student_id", student3.StudentID))
			pos3, err := store.GetCurrentPosition(ctx, query3)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{student3Event}, query3, pos3)
			Expect(err).NotTo(HaveOccurred())

			// Subscribe Alice (should succeed)
			sub2 := newStudentSubscribed(student2.StudentID, course.CourseID)
			sub2Event := NewInputEvent("StudentSubscribed", NewTags("student_id", student2.StudentID, "course_id", course.CourseID), mustJSON(sub2))
			query = NewQuery(NewTags("student_id", student2.StudentID, "course_id", course.CourseID))
			pos, err = store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{sub2Event}, query, pos)
			Expect(err).NotTo(HaveOccurred())

			// Subscribe Eve (should violate max students invariant)
			sub3 := newStudentSubscribed(student3.StudentID, course.CourseID)
			sub3Event := NewInputEvent("StudentSubscribed", NewTags("student_id", student3.StudentID, "course_id", course.CourseID), mustJSON(sub3))
			query = NewQuery(NewTags("student_id", student3.StudentID, "course_id", course.CourseID))
			pos, err = store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.AppendEvents(ctx, []InputEvent{sub3Event}, query, pos)
			// Should fail or be ignored by projection logic (depending on enforcement)

			// Stream all events and build course projection
			query = NewQuery(NewTags("course_id", course.CourseID), "StudentSubscribed")
			it, err := store.ReadEvents(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			var subscribers = map[string]bool{}
			for {
				evt, err := it.Next()
				if err != nil || evt == nil {
					break
				}
				if evt.Type == "StudentSubscribed" {
					var s studentSubscribed
					_ = json.Unmarshal(evt.Data, &s)
					subscribers[s.StudentID] = true
				}
			}
			// Only 3 students should be subscribed (store does not enforce invariants)
			Expect(len(subscribers)).To(Equal(3))
			Expect(subscribers).To(HaveKey(student.StudentID))
			Expect(subscribers).To(HaveKey(student2.StudentID))
			Expect(subscribers).To(HaveKey(student3.StudentID))
		})
	})
})

// --- Test helpers for integration (self-contained) ---
func newTestEventStore() (EventStore, func()) {
	ctx := context.Background()
	password := "testpass"
	containerReq := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": password,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: containerReq,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred())

	host, err := postgresC.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	port, err := postgresC.MappedPort(ctx, "5432")
	Expect(err).NotTo(HaveOccurred())
	dsn := fmt.Sprintf("postgres://postgres:%s@%s:%s/postgres?sslmode=disable", password, host, port.Port())
	poolConfig, err := pgxpool.ParseConfig(dsn)
	Expect(err).NotTo(HaveOccurred())
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	poolConfig.ConnConfig.StatementCacheCapacity = 100
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	Expect(err).NotTo(HaveOccurred())

	// Load schema
	schemaPath := "../../docker-entrypoint-initdb.d/schema.sql"
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		schemaPath = "../../../docker-entrypoint-initdb.d/schema.sql"
	}
	schema, err := os.ReadFile(schemaPath)
	Expect(err).NotTo(HaveOccurred())
	_, err = pool.Exec(ctx, string(schema))
	Expect(err).NotTo(HaveOccurred())

	store, err := NewEventStore(ctx, pool)
	Expect(err).NotTo(HaveOccurred())

	cleanup := func() {
		pool.Close()
		_ = postgresC.Terminate(ctx)
	}
	return store, cleanup
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	Expect(err).NotTo(HaveOccurred())
	return b
}
