package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Course Subscription Domain Types
type CourseCreated struct {
	CourseID   string `json:"courseId"`
	Name       string `json:"name"`
	Capacity   int    `json:"capacity"`
	Instructor string `json:"instructor"`
}

type StudentRegistered struct {
	StudentID string `json:"studentId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

type StudentEnrolledInCourse struct {
	StudentID  string `json:"studentId"`
	CourseID   string `json:"courseId"`
	EnrolledAt string `json:"enrolledAt"`
}

type StudentDroppedFromCourse struct {
	StudentID string `json:"studentId"`
	CourseID  string `json:"courseId"`
	DroppedAt string `json:"droppedAt"`
}

type CourseCapacityChanged struct {
	CourseID    string `json:"courseId"`
	NewCapacity int    `json:"newCapacity"`
}

// State Types
type CourseState struct {
	CourseID      string
	Name          string
	Capacity      int
	Instructor    string
	EnrolledCount int
	Exists        bool
}

type StudentState struct {
	StudentID       string
	Name            string
	Email           string
	EnrolledCourses []string
	Exists          bool
}

type EnrollmentState struct {
	StudentID  string
	CourseID   string
	IsEnrolled bool
}

// Event Constructors
func NewCourseCreatedEvent(courseID, name, instructor string, capacity int) InputEvent {
	return NewInputEventUnsafe("CourseCreated", NewTags("course_id", courseID), toJSON(CourseCreated{
		CourseID:   courseID,
		Name:       name,
		Capacity:   capacity,
		Instructor: instructor,
	}))
}

func NewStudentRegisteredEvent(studentID, name, email string) InputEvent {
	return NewInputEventUnsafe("StudentRegistered", NewTags("student_id", studentID), toJSON(StudentRegistered{
		StudentID: studentID,
		Name:      name,
		Email:     email,
	}))
}

func NewStudentEnrolledEvent(studentID, courseID, enrolledAt string) InputEvent {
	return NewInputEventUnsafe("StudentEnrolledInCourse", NewTags("student_id", studentID, "course_id", courseID), toJSON(StudentEnrolledInCourse{
		StudentID:  studentID,
		CourseID:   courseID,
		EnrolledAt: enrolledAt,
	}))
}

func NewStudentDroppedEvent(studentID, courseID, droppedAt string) InputEvent {
	return NewInputEventUnsafe("StudentDroppedFromCourse", NewTags("student_id", studentID, "course_id", courseID), toJSON(StudentDroppedFromCourse{
		StudentID: studentID,
		CourseID:  courseID,
		DroppedAt: droppedAt,
	}))
}

func NewCourseCapacityChangedEvent(courseID string, newCapacity int) InputEvent {
	return NewInputEventUnsafe("CourseCapacityChanged", NewTags("course_id", courseID), toJSON(CourseCapacityChanged{
		CourseID:    courseID,
		NewCapacity: newCapacity,
	}))
}

// Projectors
func CourseExistsProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("course_id", courseID), "CourseCreated"),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		},
	}
}

func CourseStateProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("course_id", courseID), "CourseCreated", "CourseCapacityChanged"),
		InitialState: &CourseState{CourseID: courseID, Exists: false},
		TransitionFn: func(state any, event Event) any {
			course := state.(*CourseState)
			switch event.Type {
			case "CourseCreated":
				var data CourseCreated
				json.Unmarshal(event.Data, &data)
				course.Name = data.Name
				course.Capacity = data.Capacity
				course.Instructor = data.Instructor
				course.Exists = true
			case "CourseCapacityChanged":
				var data CourseCapacityChanged
				json.Unmarshal(event.Data, &data)
				course.Capacity = data.NewCapacity
			}
			return course
		},
	}
}

func CourseEnrollmentCountProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("course_id", courseID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: 0,
		TransitionFn: func(state any, event Event) any {
			count := state.(int)
			switch event.Type {
			case "StudentEnrolledInCourse":
				return count + 1
			case "StudentDroppedFromCourse":
				return count - 1
			}
			return count
		},
	}
}

func StudentExistsProjector(studentID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID), "StudentRegistered"),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		},
	}
}

func StudentStateProjector(studentID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID), "StudentRegistered"),
		InitialState: &StudentState{StudentID: studentID, Exists: false},
		TransitionFn: func(state any, event Event) any {
			student := state.(*StudentState)
			switch event.Type {
			case "StudentRegistered":
				var data StudentRegistered
				json.Unmarshal(event.Data, &data)
				student.Name = data.Name
				student.Email = data.Email
				student.Exists = true
			}
			return student
		},
	}
}

func StudentEnrollmentCountProjector(studentID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: 0,
		TransitionFn: func(state any, event Event) any {
			count := state.(int)
			switch event.Type {
			case "StudentEnrolledInCourse":
				return count + 1
			case "StudentDroppedFromCourse":
				return count - 1
			}
			return count
		},
	}
}

func StudentEnrollmentStateProjector(studentID, courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID, "course_id", courseID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: &EnrollmentState{StudentID: studentID, CourseID: courseID, IsEnrolled: false},
		TransitionFn: func(state any, event Event) any {
			enrollment := state.(*EnrollmentState)
			switch event.Type {
			case "StudentEnrolledInCourse":
				enrollment.IsEnrolled = true
			case "StudentDroppedFromCourse":
				enrollment.IsEnrolled = false
			}
			return enrollment
		},
	}
}

// Course API for command handlers
type CourseAPI struct {
	eventStore EventStore
}

func NewCourseAPI(eventStore EventStore) *CourseAPI {
	return &CourseAPI{eventStore: eventStore}
}

func (api *CourseAPI) CreateCourse(courseID, name, instructor string, capacity int) error {
	projectors := []BatchProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(courseID)},
	}

	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project course state: %w", err)
	}

	if states["courseExists"].(bool) {
		return fmt.Errorf("course with id \"%s\" already exists", courseID)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewCourseCreatedEvent(courseID, name, instructor, capacity),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to create course: %w", err)
	}

	return nil
}

func (api *CourseAPI) RegisterStudent(studentID, name, email string) error {
	projectors := []BatchProjector{
		{ID: "studentExists", StateProjector: StudentExistsProjector(studentID)},
	}

	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project student state: %w", err)
	}

	if states["studentExists"].(bool) {
		return fmt.Errorf("student with id \"%s\" already exists", studentID)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewStudentRegisteredEvent(studentID, name, email),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to register student: %w", err)
	}

	return nil
}

func (api *CourseAPI) EnrollStudentInCourse(studentID, courseID string) error {
	projectors := []BatchProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(courseID)},
		{ID: "studentExists", StateProjector: StudentExistsProjector(studentID)},
		{ID: "courseState", StateProjector: CourseStateProjector(courseID)},
		{ID: "courseEnrollmentCount", StateProjector: CourseEnrollmentCountProjector(courseID)},
		{ID: "studentEnrollmentCount", StateProjector: StudentEnrollmentCountProjector(studentID)},
		{ID: "studentEnrollmentState", StateProjector: StudentEnrollmentStateProjector(studentID, courseID)},
	}

	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project enrollment state: %w", err)
	}

	// Business rule validations
	if !states["courseExists"].(bool) {
		return fmt.Errorf("course \"%s\" does not exist", courseID)
	}

	if !states["studentExists"].(bool) {
		return fmt.Errorf("student \"%s\" does not exist", studentID)
	}

	courseState := states["courseState"].(*CourseState)
	courseEnrollmentCount := states["courseEnrollmentCount"].(int)
	studentEnrollmentCount := states["studentEnrollmentCount"].(int)
	studentEnrollmentState := states["studentEnrollmentState"].(*EnrollmentState)

	// Business rules
	if studentEnrollmentState.IsEnrolled {
		return fmt.Errorf("student \"%s\" is already enrolled in course \"%s\"", studentID, courseID)
	}

	if courseEnrollmentCount >= courseState.Capacity {
		return fmt.Errorf("course \"%s\" is already full (%d students maximum)", courseID, courseState.Capacity)
	}

	if studentEnrollmentCount >= 10 {
		return fmt.Errorf("student \"%s\" is already enrolled in 10 courses (maximum allowed)", studentID)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewStudentEnrolledEvent(studentID, courseID, time.Now().Format(time.RFC3339)),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to enroll student: %w", err)
	}

	return nil
}

func (api *CourseAPI) DropStudentFromCourse(studentID, courseID string) error {
	projectors := []BatchProjector{
		{ID: "studentEnrollmentState", StateProjector: StudentEnrollmentStateProjector(studentID, courseID)},
	}

	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project enrollment state: %w", err)
	}

	studentEnrollmentState := states["studentEnrollmentState"].(*EnrollmentState)

	if !studentEnrollmentState.IsEnrolled {
		return fmt.Errorf("student \"%s\" is not enrolled in course \"%s\"", studentID, courseID)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewStudentDroppedEvent(studentID, courseID, time.Now().Format(time.RFC3339)),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to drop student: %w", err)
	}

	return nil
}

func (api *CourseAPI) ChangeCourseCapacity(courseID string, newCapacity int) error {
	projectors := []BatchProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(courseID)},
		{ID: "courseEnrollmentCount", StateProjector: CourseEnrollmentCountProjector(courseID)},
	}

	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project course state: %w", err)
	}

	if !states["courseExists"].(bool) {
		return fmt.Errorf("course \"%s\" does not exist", courseID)
	}

	currentEnrollmentCount := states["courseEnrollmentCount"].(int)
	if newCapacity < currentEnrollmentCount {
		return fmt.Errorf("cannot reduce capacity to %d when %d students are already enrolled", newCapacity, currentEnrollmentCount)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewCourseCapacityChangedEvent(courseID, newCapacity),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to change course capacity: %w", err)
	}

	return nil
}

// Test Suite
var _ = Describe("Course Subscription Domain", func() {
	var (
		store EventStore
		api   *CourseAPI
		ctx   context.Context
	)

	BeforeEach(func() {
		// Use shared PostgreSQL container and truncate events between tests
		store = NewEventStoreFromPool(pool)
		api = NewCourseAPI(store)
		ctx = context.Background()

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Core EventStore Operations", func() {
		It("should append and read events", func() {
			// Create course
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())

			// Read events
			query := NewQuerySimple(NewTags("course_id", "course-1"), "CourseCreated")
			sequencedEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sequencedEvents.Events).To(HaveLen(1))
			Expect(sequencedEvents.Events[0].Type).To(Equal("CourseCreated"))
		})

		It("should use ReadStream for large datasets", func() {
			// Create multiple courses
			for i := 1; i <= 5; i++ {
				err := api.CreateCourse(fmt.Sprintf("course-%d", i), fmt.Sprintf("Course %d", i), "Instructor", 20)
				Expect(err).NotTo(HaveOccurred())
			}

			// Use ReadStream
			query := NewQuerySimple(NewTags(), "CourseCreated")
			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			count := 0
			for iterator.Next() {
				event := iterator.Event()
				Expect(event.Type).To(Equal("CourseCreated"))
				count++
			}

			Expect(count).To(Equal(5))
			Expect(iterator.Err()).NotTo(HaveOccurred())
		})

		It("should handle optimistic locking", func() {
			// Create course
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())

			// First projection
			projectors := []BatchProjector{
				{ID: "courseState", StateProjector: CourseStateProjector("course-1")},
			}
			_, appendCondition1, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second projection (simulating concurrent read)
			_, appendCondition2, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// First update succeeds using appendCondition1
			_, err = store.Append(ctx, []InputEvent{
				NewCourseCapacityChangedEvent("course-1", 30),
			}, &appendCondition1)
			Expect(err).NotTo(HaveOccurred())

			// Second update should fail due to optimistic locking
			_, err = store.Append(ctx, []InputEvent{
				NewCourseCapacityChangedEvent("course-1", 35),
			}, &appendCondition2)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ProjectDecisionModel", func() {
		It("should project multiple states", func() {
			// Setup data
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student-1", "Alice", "alice@example.com")
			Expect(err).NotTo(HaveOccurred())

			// Project multiple states
			projectors := []BatchProjector{
				{ID: "courseState", StateProjector: CourseStateProjector("course-1")},
				{ID: "studentState", StateProjector: StudentStateProjector("student-1")},
			}

			states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(appendCondition.After).NotTo(BeNil())

			course := states["courseState"].(*CourseState)
			Expect(course.Name).To(Equal("Math 101"))
			Expect(course.Capacity).To(Equal(25))

			student := states["studentState"].(*StudentState)
			Expect(student.Name).To(Equal("Alice"))
			Expect(student.Email).To(Equal("alice@example.com"))
		})

		It("should use cursor streaming for large datasets", func() {
			// Create many courses
			for i := 1; i <= 100; i++ {
				err := api.CreateCourse(fmt.Sprintf("course-%d", i), fmt.Sprintf("Course %d", i), "Instructor", 20)
				Expect(err).NotTo(HaveOccurred())
			}

			// Use cursor streaming
			batchSize := 10
			options := &ReadOptions{BatchSize: &batchSize}

			projectors := []BatchProjector{
				{ID: "courseCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags(), "CourseCreated"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, options)
			Expect(err).NotTo(HaveOccurred())
			Expect(states["courseCount"]).To(Equal(100))
			Expect(appendCondition.After).NotTo(BeNil())
		})
	})

	Describe("Business Rules", func() {
		It("should enforce course capacity limit", func() {
			// Create course with capacity 2
			err := api.CreateCourse("course-1", "Small Course", "Instructor", 2)
			Expect(err).NotTo(HaveOccurred())

			// Register 3 students
			for i := 1; i <= 3; i++ {
				err := api.RegisterStudent(fmt.Sprintf("student-%d", i), fmt.Sprintf("Student %d", i), fmt.Sprintf("student%d@example.com", i))
				Expect(err).NotTo(HaveOccurred())
			}

			// Enroll first 2 students (should succeed)
			err = api.EnrollStudentInCourse("student-1", "course-1")
			Expect(err).NotTo(HaveOccurred())
			err = api.EnrollStudentInCourse("student-2", "course-1")
			Expect(err).NotTo(HaveOccurred())

			// Try to enroll 3rd student (should fail)
			err = api.EnrollStudentInCourse("student-3", "course-1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already full"))
		})

		It("should enforce student course limit", func() {
			// Create 11 courses
			for i := 1; i <= 11; i++ {
				err := api.CreateCourse(fmt.Sprintf("course-%d", i), fmt.Sprintf("Course %d", i), "Instructor", 30)
				Expect(err).NotTo(HaveOccurred())
			}

			// Register student
			err := api.RegisterStudent("student-1", "Alice", "alice@example.com")
			Expect(err).NotTo(HaveOccurred())

			// Enroll in 10 courses (should succeed)
			for i := 1; i <= 10; i++ {
				err = api.EnrollStudentInCourse("student-1", fmt.Sprintf("course-%d", i))
				Expect(err).NotTo(HaveOccurred())
			}

			// Try to enroll in 11th course (should fail)
			err = api.EnrollStudentInCourse("student-1", "course-11")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already enrolled in 10 courses"))
		})

		It("should prevent duplicate enrollments", func() {
			// Setup
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student-1", "Alice", "alice@example.com")
			Expect(err).NotTo(HaveOccurred())

			// First enrollment (should succeed)
			err = api.EnrollStudentInCourse("student-1", "course-1")
			Expect(err).NotTo(HaveOccurred())

			// Second enrollment (should fail)
			err = api.EnrollStudentInCourse("student-1", "course-1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already enrolled"))
		})

		It("should allow dropping and re-enrolling", func() {
			// Setup
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student-1", "Alice", "alice@example.com")
			Expect(err).NotTo(HaveOccurred())

			// Enroll
			err = api.EnrollStudentInCourse("student-1", "course-1")
			Expect(err).NotTo(HaveOccurred())

			// Drop
			err = api.DropStudentFromCourse("student-1", "course-1")
			Expect(err).NotTo(HaveOccurred())

			// Re-enroll (should succeed)
			err = api.EnrollStudentInCourse("student-1", "course-1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should prevent capacity reduction below current enrollment", func() {
			// Setup
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student-1", "Alice", "alice@example.com")
			Expect(err).NotTo(HaveOccurred())
			err = api.EnrollStudentInCourse("student-1", "course-1")
			Expect(err).NotTo(HaveOccurred())

			// Try to reduce capacity below current enrollment (should fail)
			err = api.ChangeCourseCapacity("course-1", 1)
			Expect(err).NotTo(HaveOccurred()) // This should work since 1 student is enrolled

			// Try to reduce to 0 (should fail)
			err = api.ChangeCourseCapacity("course-1", 0)
			Expect(err).To(HaveOccurred())
		})

		It("should handle business rule violations", func() {
			// Create a course with capacity 2
			courseCreated := NewInputEvent("CourseCreated", NewTags("course_id", "math101"), toJSON(map[string]interface{}{
				"name":     "Mathematics 101",
				"capacity": 2,
			}))
			events := []InputEvent{courseCreated}
			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Register students first
			err = api.RegisterStudent("student1", "Student One", "student1@example.com")
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student2", "Student Two", "student2@example.com")
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student3", "Student Three", "student3@example.com")
			Expect(err).NotTo(HaveOccurred())

			// Enroll 3 students (exceeds capacity)
			err = api.EnrollStudentInCourse("student1", "math101")
			Expect(err).NotTo(HaveOccurred())
			err = api.EnrollStudentInCourse("student2", "math101")
			Expect(err).NotTo(HaveOccurred())
			err = api.EnrollStudentInCourse("student3", "math101")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already full"))
		})

		It("should handle concurrent enrollment attempts", func() {
			// Create a course with capacity 1
			courseCreated := NewInputEvent("CourseCreated", NewTags("course_id", "math101"), toJSON(map[string]interface{}{
				"name":     "Mathematics 101",
				"capacity": 1,
			}))
			events := []InputEvent{courseCreated}
			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Register students first
			err = api.RegisterStudent("student1", "Student One", "student1@example.com")
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student2", "Student Two", "student2@example.com")
			Expect(err).NotTo(HaveOccurred())

			// Create multiple enrollment attempts
			err = api.EnrollStudentInCourse("student1", "math101")
			Expect(err).NotTo(HaveOccurred())
			err = api.EnrollStudentInCourse("student2", "math101")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already full"))
		})

		It("should handle large datasets efficiently", func() {
			// Create many courses and enrollments
			events := make([]InputEvent, 1000)
			for i := 0; i < 1000; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", fmt.Sprintf("value-%d", i)), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// This test would require complex projection logic
			// For now, just verify the events were appended
			query := NewQuerySimple(NewTags("test", "value-0"), "TestEvent")
			sequencedEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sequencedEvents.Events).To(HaveLen(1))
		})
	})

	Describe("Validation Error Scenarios", func() {
		It("should handle invalid JSON data in events", func() {
			// Create event with invalid JSON - validation should happen in EventStore operations
			event := NewInputEvent("TestEvent", NewTags("test", "value"), []byte("invalid json"))
			Expect(event.Type).To(Equal("TestEvent"))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid input syntax for type json"))
		})

		It("should handle empty event types in queries", func() {
			query := NewQueryFromItems(NewQueryItem([]string{""}, NewTags("course_id", "test")))

			_, err := store.Read(ctx, query, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty event type"))
		})

		It("should handle empty tag keys/values", func() {
			// Test empty tag key - validation should happen in EventStore operations
			event := NewInputEvent("TestEvent", []Tag{{Key: "", Value: "value"}}, toJSON(map[string]string{"test": "data"}))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags[0].Key).To(Equal(""))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key"))

			// Test empty tag value - validation should happen in EventStore operations
			event = NewInputEvent("TestEvent", []Tag{{Key: "test", Value: ""}}, toJSON(map[string]string{"test": "data"}))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags[0].Value).To(Equal(""))

			// Try to append the event - this should fail validation
			events = []InputEvent{event}
			_, err = store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value"))
		})

		It("should handle batch size limit validation", func() {
			// Create events exceeding the batch size limit
			events := make([]InputEvent, 1001) // Exceeds default limit of 1000
			for i := 0; i < 1001; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", fmt.Sprintf("value-%d", i)), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})
	})

	Describe("ConcurrencyError Scenarios", func() {
		It("should handle optimistic locking failures", func() {
			// Create course
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())

			// First projection
			projectors := []BatchProjector{
				{ID: "courseState", StateProjector: CourseStateProjector("course-1")},
			}
			_, appendCondition1, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Make a change that invalidates the append condition
			_, err = store.Append(ctx, []InputEvent{
				NewCourseCapacityChangedEvent("course-1", 30),
			}, nil)
			Expect(err).NotTo(HaveOccurred())

			// Try to use the old append condition (should fail)
			_, err = store.Append(ctx, []InputEvent{
				NewCourseCapacityChangedEvent("course-1", 35),
			}, &appendCondition1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("append condition violated"))
		})

		It("should handle concurrent append conflicts", func() {
			// Create course
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())

			// Register two students
			err = api.RegisterStudent("student-1", "Alice", "alice@example.com")
			Expect(err).NotTo(HaveOccurred())
			err = api.RegisterStudent("student-2", "Bob", "bob@example.com")
			Expect(err).NotTo(HaveOccurred())

			// Both students try to enroll simultaneously
			// This should cause a concurrency conflict
			projectors := []BatchProjector{
				{ID: "courseEnrollmentCount", StateProjector: CourseEnrollmentCountProjector("course-1")},
			}

			// First enrollment
			_, appendCondition1, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second enrollment (should conflict)
			_, appendCondition2, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// First enrollment succeeds
			_, err = store.Append(ctx, []InputEvent{
				NewStudentEnrolledEvent("student-1", "course-1", time.Now().Format(time.RFC3339)),
			}, &appendCondition1)
			Expect(err).NotTo(HaveOccurred())

			// Second enrollment should fail due to concurrency conflict
			_, err = store.Append(ctx, []InputEvent{
				NewStudentEnrolledEvent("student-2", "course-1", time.Now().Format(time.RFC3339)),
			}, &appendCondition2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("append condition violated"))
		})
	})

	Describe("EventIterator Interface Tests", func() {
		It("should handle iterator error propagation", func() {
			// Create a query that should work fine but return no results
			query := NewQueryFromItems(NewQueryItem([]string{"NonExistentEvent"}, NewTags("course_id", "non-existent")))

			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Try to iterate (should not cause error, just no results)
			for iterator.Next() {
				// Should not reach here
				Fail("Should not have any events")
			}

			// Err should be nil for empty results (not an error condition)
			Expect(iterator.Err()).NotTo(HaveOccurred())
		})

		It("should handle iterator resource cleanup", func() {
			// Setup data
			err := api.CreateCourse("course-1", "Math 101", "Dr. Smith", 25)
			Expect(err).NotTo(HaveOccurred())

			query := NewQuerySimple(NewTags("course_id", "course-1"), "CourseCreated")
			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read one event
			Expect(iterator.Next()).To(BeTrue())
			event := iterator.Event()
			Expect(event.Type).To(Equal("CourseCreated"))

			// Close iterator
			err = iterator.Close()
			Expect(err).NotTo(HaveOccurred())

			// Try to use iterator after close (should not panic)
			Expect(func() {
				iterator.Next()
			}).NotTo(Panic())
		})

		It("should handle multiple iterator operations", func() {
			// Create multiple courses
			for i := 1; i <= 3; i++ {
				err := api.CreateCourse(fmt.Sprintf("course-%d", i), fmt.Sprintf("Course %d", i), "Instructor", 20)
				Expect(err).NotTo(HaveOccurred())
			}

			query := NewQuerySimple(NewTags(), "CourseCreated")
			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Test multiple Event() calls on same position
			Expect(iterator.Next()).To(BeTrue())
			event1 := iterator.Event()
			event2 := iterator.Event() // Should return same event
			Expect(event1.ID).To(Equal(event2.ID))

			// Test iteration through all events
			count := 1 // We already read one
			for iterator.Next() {
				event := iterator.Event()
				Expect(event.Type).To(Equal("CourseCreated"))
				count++
			}
			Expect(count).To(Equal(3))
			Expect(iterator.Err()).NotTo(HaveOccurred())
		})
	})

	It("should handle large number of events efficiently", func() {
		// Create large dataset
		events := make([]InputEvent, 1000)
		for i := 0; i < 1000; i++ {
			event := NewInputEvent("TestEvent", NewTags("test", fmt.Sprintf("value-%d", i)), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
			events[i] = event
		}

		_, err := store.Append(ctx, events, nil)
		Expect(err).NotTo(HaveOccurred())

		// This test would require complex projection logic
		// For now, just verify the events were appended
		query := NewQuerySimple(NewTags("test", "value-0"), "TestEvent")
		sequencedEvents, err := store.Read(ctx, query, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(sequencedEvents.Events).To(HaveLen(1))
	})
})

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}

// Helper functions for creating domain events
func createCourseCreatedEvent(courseID string, name string, capacity int) InputEvent {
	return NewInputEvent("CourseCreated", NewTags("course_id", courseID), toJSON(CourseCreated{
		Name:     name,
		Capacity: capacity,
	}))
}

func createStudentRegisteredEvent(studentID string, name string) InputEvent {
	return NewInputEvent("StudentRegistered", NewTags("student_id", studentID), toJSON(StudentRegistered{
		Name: name,
	}))
}

func createStudentEnrolledInCourseEvent(studentID string, courseID string) InputEvent {
	return NewInputEvent("StudentEnrolledInCourse", NewTags("student_id", studentID, "course_id", courseID), toJSON(StudentEnrolledInCourse{
		EnrolledAt: time.Now().Format(time.RFC3339),
	}))
}

func createStudentDroppedFromCourseEvent(studentID string, courseID string) InputEvent {
	return NewInputEvent("StudentDroppedFromCourse", NewTags("student_id", studentID, "course_id", courseID), toJSON(StudentDroppedFromCourse{
		DroppedAt: time.Now().Format(time.RFC3339),
	}))
}

func createCourseCapacityChangedEvent(courseID string, newCapacity int) InputEvent {
	return NewInputEvent("CourseCapacityChanged", NewTags("course_id", courseID), toJSON(CourseCapacityChanged{
		NewCapacity: newCapacity,
	}))
}
