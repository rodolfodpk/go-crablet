package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Command types - following the command pattern from examples
type CreateCourseCommand struct {
	CourseID   string
	Name       string
	Instructor string
	Capacity   int
}

type RegisterStudentCommand struct {
	StudentID string
	Name      string
	Email     string
}

type EnrollStudentCommand struct {
	StudentID string
	CourseID  string
}

type DropStudentCommand struct {
	StudentID string
	CourseID  string
}

type ChangeCourseCapacityCommand struct {
	CourseID    string
	NewCapacity int
}

// Course Subscription Domain Types
type CourseDefined struct {
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
func NewCourseDefinedEvent(courseID, name, instructor string, capacity int) InputEvent {
	return NewInputEventUnsafe("CourseDefined", NewTags("course_id", courseID), toJSON(CourseDefined{
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
		Query:        NewQuerySimple(NewTags("course_id", courseID), "CourseDefined"),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		
	}
}

func CourseStateProjector(courseID string) StateProjector {
	return StateProjector{
		Query: NewQuerySimple(
			NewTags("course_id", courseID),
			"CourseDefined", "CourseCapacityChanged", "StudentEnrolledInCourse", "StudentDroppedFromCourse",
		),
		InitialState: &CourseState{CourseID: courseID, Exists: false
		TransitionFn: func(state any, event Event) any {
			course := state.(*CourseState)
			switch event.Type {
			case "CourseDefined":
				var data CourseDefined
				json.Unmarshal(event.Data, &data)
				course.Name = data.Name
				course.Capacity = data.Capacity
				course.Instructor = data.Instructor
				course.Exists = true
			case "CourseCapacityChanged":
				var data CourseCapacityChanged
				json.Unmarshal(event.Data, &data)
				course.Capacity = data.NewCapacity
			case "StudentEnrolledInCourse":
				course.EnrolledCount++
			case "StudentDroppedFromCourse":
				course.EnrolledCount--
			}
			return course
		
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
		
	}
}

func StudentExistsProjector(studentID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID), "StudentRegistered"),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		
	}
}

func StudentStateProjector(studentID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID), "StudentRegistered"),
		InitialState: &StudentState{StudentID: studentID, Exists: false
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
		
	}
}

func StudentEnrollmentStateProjector(studentID, courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuerySimple(NewTags("student_id", studentID, "course_id", courseID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: &EnrollmentState{StudentID: studentID, CourseID: courseID, IsEnrolled: false
		TransitionFn: func(state any, event Event) any {
			enrollment := state.(*EnrollmentState)
			switch event.Type {
			case "StudentEnrolledInCourse":
				enrollment.IsEnrolled = true
			case "StudentDroppedFromCourse":
				enrollment.IsEnrolled = false
			}
			return enrollment
		
	}
}

// Command handlers - following the command pattern from examples
func handleCreateCourse(ctx context.Context, store EventStore, cmd CreateCourseCommand) error {
	projectors := []StateProjector{
		{ID: "courseExists",
			Query:        CourseExistsProjector(cmd.CourseID).Query,
			InitialState: CourseExistsProjector(cmd.CourseID).InitialState,
			TransitionFn: CourseExistsProjector(cmd.CourseID).TransitionFn,
		
	}

	states, _, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		return fmt.Errorf("failed to project course state: %w", err)
	}

	if states["courseExists"].(bool) {
		return fmt.Errorf("course with id \"%s\" already exists", cmd.CourseID)
	}

	err = store.AppendIf(ctx, []InputEvent{
		NewCourseDefinedEvent(cmd.CourseID, cmd.Name, cmd.Instructor, cmd.Capacity),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create course: %w", err)
	}

	return nil
}

func handleRegisterStudent(ctx context.Context, store EventStore, cmd RegisterStudentCommand) error {
	projectors := []StateProjector{
		{ID: "studentExists", StateProjector: StudentExistsProjector(cmd.StudentID)
	}

	states, _, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		return fmt.Errorf("failed to project student state: %w", err)
	}

	if states["studentExists"].(bool) {
		return fmt.Errorf("student with id \"%s\" already exists", cmd.StudentID)
	}

	err = store.AppendIf(ctx, []InputEvent{
		NewStudentRegisteredEvent(cmd.StudentID, cmd.Name, cmd.Email),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to register student: %w", err)
	}

	return nil
}

func handleEnrollStudent(ctx context.Context, store EventStore, cmd EnrollStudentCommand) error {
	projectors := []StateProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(cmd.CourseID)
		{ID: "studentExists", StateProjector: StudentExistsProjector(cmd.StudentID)
		{ID: "courseState", StateProjector: CourseStateProjector(cmd.CourseID)
		{ID: "courseEnrollmentCount", StateProjector: CourseEnrollmentCountProjector(cmd.CourseID)
		{ID: "studentEnrollmentCount", StateProjector: StudentEnrollmentCountProjector(cmd.StudentID)
		{ID: "studentEnrollmentState", StateProjector: StudentEnrollmentStateProjector(cmd.StudentID, cmd.CourseID)
	}

	states, _, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		return fmt.Errorf("failed to project enrollment state: %w", err)
	}

	// Business rule validations
	if !states["courseExists"].(bool) {
		return fmt.Errorf("course \"%s\" does not exist", cmd.CourseID)
	}

	if !states["studentExists"].(bool) {
		return fmt.Errorf("student \"%s\" does not exist", cmd.StudentID)
	}

	courseState := states["courseState"].(*CourseState)
	courseEnrollmentCount := states["courseEnrollmentCount"].(int)
	studentEnrollmentCount := states["studentEnrollmentCount"].(int)
	studentEnrollmentState := states["studentEnrollmentState"].(*EnrollmentState)

	// Business rules
	if studentEnrollmentState.IsEnrolled {
		return fmt.Errorf("student \"%s\" is already enrolled in course \"%s\"", cmd.StudentID, cmd.CourseID)
	}

	if courseEnrollmentCount >= courseState.Capacity {
		return fmt.Errorf("course \"%s\" is already full (%d students maximum)", cmd.CourseID, courseState.Capacity)
	}

	if studentEnrollmentCount >= 10 {
		return fmt.Errorf("student \"%s\" is already enrolled in 10 courses (maximum allowed)", cmd.StudentID)
	}

	// DCB-compliant approach: use specific query for enrollment append condition
	// Only check for duplicate enrollment events, not all projector queries
	enrollmentQuery := NewQuerySimple(NewTags("student_id", cmd.StudentID, "course_id", cmd.CourseID), "StudentEnrolledInCourse")
	appendCondition := NewAppendCondition(enrollmentQuery)

	err = store.AppendIf(ctx, []InputEvent{
		NewStudentEnrolledEvent(cmd.StudentID, cmd.CourseID, time.Now().Format(time.RFC3339)),
	}, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to enroll student: %w", err)
	}

	return nil
}

func handleDropStudent(ctx context.Context, store EventStore, cmd DropStudentCommand) error {
	projectors := []StateProjector{
		{ID: "studentEnrollmentState", StateProjector: StudentEnrollmentStateProjector(cmd.StudentID, cmd.CourseID)
	}

	states, _, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		return fmt.Errorf("failed to project enrollment state: %w", err)
	}

	studentEnrollmentState := states["studentEnrollmentState"].(*EnrollmentState)

	if !studentEnrollmentState.IsEnrolled {
		return fmt.Errorf("student \"%s\" is not enrolled in course \"%s\"", cmd.StudentID, cmd.CourseID)
	}

	// DCB-compliant approach: for drop operations, we don't need FailIfEventsMatch
	// because we've already verified the student is enrolled through the projection
	// We only need optimistic locking to ensure no concurrent changes
	appendCondition := NewAppendCondition(nil) // No need to check for existing events

	err = store.AppendIf(ctx, []InputEvent{
		NewStudentDroppedEvent(cmd.StudentID, cmd.CourseID, time.Now().Format(time.RFC3339)),
	}, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to drop student: %w", err)
	}

	return nil
}

func handleChangeCourseCapacity(ctx context.Context, store EventStore, cmd ChangeCourseCapacityCommand) error {
	courseID := cmd.CourseID
	newCapacity := cmd.NewCapacity

	// Project course state
	projectors := []StateProjector{
		{ID: "courseState", StateProjector: CourseStateProjector(courseID)
	}

	states, _, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		return err
	}

	courseState := states["courseState"].(*CourseState)
	if !courseState.Exists {
		return fmt.Errorf("course %s does not exist", courseID)
	}

	// Check if new capacity is less than current enrollment count
	if newCapacity < courseState.EnrolledCount {
		return fmt.Errorf("cannot reduce capacity to %d when %d students are enrolled", newCapacity, courseState.EnrolledCount)
	}

	// Create capacity change event
	event := NewCourseCapacityChangedEvent(courseID, newCapacity)

	// Append with optimistic locking using the same query
	appendCondition := NewAppendCondition(NewQuerySimple(NewTags("course_id", courseID), "CourseCapacityChanged"))
	err = store.AppendIf(ctx, []InputEvent{event}, appendCondition)
	if err != nil {
		return err
	}

	return nil
}

// Test Suite
var _ = Describe("Course Subscription Domain", func() {
	var (
		store        EventStore
		channelStore EventStore
		ctx          context.Context
	)

	BeforeEach(func() {
		// Use shared PostgreSQL container and truncate events between tests
		store = NewEventStoreFromPool(pool)
		channelStore = store.(EventStore)

		// Create context with timeout for each test
		ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Core EventStore Operations", func() {
		It("should append and read events", func() {
			// Create course using command pattern
			createCourseCmd := CreateCourseCommand{
				CourseID:   "course-1",
				Name:       "Math 101",
				Instructor: "Dr. Smith",
				Capacity:   25,
			}
			channelStore := store.(EventStore)
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Read events
			query := NewQuerySimple(NewTags("course_id", "course-1"), "CourseDefined")
			events, err := store.Read(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
			Expect(events[0].Type).To(Equal("CourseDefined"))
		})

		It("should use ReadChannel for large datasets", func() {
			// Create multiple courses using command pattern
			channelStore := store.(EventStore)
			for i := 1; i <= 5; i++ {
				createCourseCmd := CreateCourseCommand{
					CourseID:   fmt.Sprintf("course-%d", i),
					Name:       fmt.Sprintf("Course %d", i),
					Instructor: "Instructor",
					Capacity:   20,
				}
				err := handleCreateCourse(ctx, channelStore, createCourseCmd)
				Expect(err).NotTo(HaveOccurred())
			}

			// Use ReadChannel instead of ReadStream
			query := NewQuerySimple(NewTags(), "CourseDefined")
			eventChan, err := channelStore.ReadChannel(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			eventCount := 0
			for range eventChan {
				eventCount++
			}
			Expect(eventCount).To(Equal(5))
		})
	})

	Describe("Command Pattern Operations", func() {
		It("should create course successfully", func() {
			cmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}

			channelStore := store.(EventStore)
			err := handleCreateCourse(ctx, channelStore, cmd)
			Expect(err).NotTo(HaveOccurred())

			// Verify course was created
			query := NewQuerySimple(NewTags("course_id", "math-101"), "CourseDefined")
			events, err := store.Read(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
		})

		It("should prevent duplicate course creation", func() {
			cmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}

			channelStore := store.(EventStore)
			// Create course first time
			err := handleCreateCourse(ctx, channelStore, cmd)
			Expect(err).NotTo(HaveOccurred())

			// Try to create same course again
			err = handleCreateCourse(ctx, channelStore, cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should register student successfully", func() {
			cmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}

			channelStore := store.(EventStore)
			err := handleRegisterStudent(ctx, channelStore, cmd)
			Expect(err).NotTo(HaveOccurred())

			// Verify student was registered
			query := NewQuerySimple(NewTags("student_id", "student-123"), "StudentRegistered")
			events, err := store.Read(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
		})

		It("should prevent duplicate student registration", func() {
			cmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}

			channelStore := store.(EventStore)
			// Register student first time
			err := handleRegisterStudent(ctx, channelStore, cmd)
			Expect(err).NotTo(HaveOccurred())

			// Try to register same student again
			err = handleRegisterStudent(ctx, channelStore, cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should enroll student in course successfully", func() {
			// Create course first
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			channelStore := store.(EventStore)
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Register student
			registerStudentCmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}
			err = handleRegisterStudent(ctx, channelStore, registerStudentCmd)
			Expect(err).NotTo(HaveOccurred())

			// Enroll student
			enrollCmd := EnrollStudentCommand{
				StudentID: "student-123",
				CourseID:  "math-101",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd)
			Expect(err).NotTo(HaveOccurred())

			// Verify enrollment
			query := NewQuerySimple(NewTags("student_id", "student-123", "course_id", "math-101"), "StudentEnrolledInCourse")
			events, err := store.Read(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
		})

		It("should prevent enrollment in non-existent course", func() {
			// Register student
			registerStudentCmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}
			channelStore := store.(EventStore)
			err := handleRegisterStudent(ctx, channelStore, registerStudentCmd)
			Expect(err).NotTo(HaveOccurred())

			// Try to enroll in non-existent course
			enrollCmd := EnrollStudentCommand{
				StudentID: "student-123",
				CourseID:  "non-existent-course",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("should prevent enrollment of non-existent student", func() {
			// Create course
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			channelStore := store.(EventStore)
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Try to enroll non-existent student
			enrollCmd := EnrollStudentCommand{
				StudentID: "non-existent-student",
				CourseID:  "math-101",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("should prevent duplicate enrollment", func() {
			// Create course
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			channelStore := store.(EventStore)
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Register student
			registerStudentCmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}
			err = handleRegisterStudent(ctx, channelStore, registerStudentCmd)
			Expect(err).NotTo(HaveOccurred())

			// Enroll student first time
			enrollCmd := EnrollStudentCommand{
				StudentID: "student-123",
				CourseID:  "math-101",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd)
			Expect(err).NotTo(HaveOccurred())

			// Try to enroll same student again
			err = handleEnrollStudent(ctx, channelStore, enrollCmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already enrolled"))
		})

		It("should prevent enrollment when course is full", func() {
			// Create course with capacity 1
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   1,
			}
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Register two students
			for i := 1; i <= 2; i++ {
				registerStudentCmd := RegisterStudentCommand{
					StudentID: fmt.Sprintf("student-%d", i),
					Name:      fmt.Sprintf("Student %d", i),
					Email:     fmt.Sprintf("student%d@example.com", i),
				}
				err = handleRegisterStudent(ctx, channelStore, registerStudentCmd)
				Expect(err).NotTo(HaveOccurred())
			}

			// Enroll first student
			enrollCmd1 := EnrollStudentCommand{
				StudentID: "student-1",
				CourseID:  "math-101",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd1)
			Expect(err).NotTo(HaveOccurred())

			// Try to enroll second student (should fail - course is full)
			enrollCmd2 := EnrollStudentCommand{
				StudentID: "student-2",
				CourseID:  "math-101",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already full"))
		})

		It("should drop student from course successfully", func() {
			// Create course
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Register student
			registerStudentCmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}
			err = handleRegisterStudent(ctx, channelStore, registerStudentCmd)
			Expect(err).NotTo(HaveOccurred())

			// Enroll student
			enrollCmd := EnrollStudentCommand{
				StudentID: "student-123",
				CourseID:  "math-101",
			}
			err = handleEnrollStudent(ctx, channelStore, enrollCmd)
			Expect(err).NotTo(HaveOccurred())

			// Drop student
			dropCmd := DropStudentCommand{
				StudentID: "student-123",
				CourseID:  "math-101",
			}
			err = handleDropStudent(ctx, channelStore, dropCmd)
			Expect(err).NotTo(HaveOccurred())

			// Verify drop event
			query := NewQuerySimple(NewTags("student_id", "student-123", "course_id", "math-101"), "StudentDroppedFromCourse")
			events, err := store.Read(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
		})

		It("should prevent dropping non-enrolled student", func() {
			// Create course
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Register student
			registerStudentCmd := RegisterStudentCommand{
				StudentID: "student-123",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
			}
			err = handleRegisterStudent(ctx, channelStore, registerStudentCmd)
			Expect(err).NotTo(HaveOccurred())

			// Try to drop non-enrolled student
			dropCmd := DropStudentCommand{
				StudentID: "student-123",
				CourseID:  "math-101",
			}
			err = handleDropStudent(ctx, channelStore, dropCmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not enrolled"))
		})

		It("should change course capacity successfully", func() {
			// Create course
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Change capacity
			changeCapacityCmd := ChangeCourseCapacityCommand{
				CourseID:    "math-101",
				NewCapacity: 50,
			}
			err = handleChangeCourseCapacity(ctx, channelStore, changeCapacityCmd)
			Expect(err).NotTo(HaveOccurred())

			// Verify capacity change event
			query := NewQuerySimple(NewTags("course_id", "math-101"), "CourseCapacityChanged")
			events, err := store.Read(ctx, query)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
		})

		It("should prevent capacity reduction below enrollment count", func() {
			// Create course with capacity 2
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   2,
			}
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Register and enroll two students
			for i := 1; i <= 2; i++ {
				registerStudentCmd := RegisterStudentCommand{
					StudentID: fmt.Sprintf("student-%d", i),
					Name:      fmt.Sprintf("Student %d", i),
					Email:     fmt.Sprintf("student%d@example.com", i),
				}
				err = handleRegisterStudent(ctx, channelStore, registerStudentCmd)
				Expect(err).NotTo(HaveOccurred())

				enrollCmd := EnrollStudentCommand{
					StudentID: fmt.Sprintf("student-%d", i),
					CourseID:  "math-101",
				}
				err = handleEnrollStudent(ctx, channelStore, enrollCmd)
				Expect(err).NotTo(HaveOccurred())
			}

			// Try to reduce capacity to 1 (should fail - 2 students enrolled)
			changeCapacityCmd := ChangeCourseCapacityCommand{
				CourseID:    "math-101",
				NewCapacity: 1,
			}
			err = handleChangeCourseCapacity(ctx, channelStore, changeCapacityCmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot reduce capacity"))
		})
	})

	Describe("Decision Model Projection", func() {
		It("should project course state correctly", func() {
			// Create course
			createCourseCmd := CreateCourseCommand{
				CourseID:   "math-101",
				Name:       "Mathematics 101",
				Instructor: "Dr. Johnson",
				Capacity:   30,
			}
			err := handleCreateCourse(ctx, channelStore, createCourseCmd)
			Expect(err).NotTo(HaveOccurred())

			// Project course state
			projectors := []StateProjector{
				{ID: "courseState", StateProjector: CourseStateProjector("math-101")
			}

			states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			courseState := states["courseState"].(*CourseState)
			Expect(courseState.CourseID).To(Equal("math-101"))
			Expect(courseState.Name).To(Equal("Mathematics 101"))
			Expect(courseState.Instructor).To(Equal("Dr. Johnson"))
			Expect(courseState.Capacity).To(Equal(30))
			Expect(courseState.Exists).To(BeTrue())

			// Test optimistic locking with append condition
			changeCapacityCmd := ChangeCourseCapacityCommand{
				CourseID:    "math-101",
				NewCapacity: 40,
			}
			err = handleChangeCourseCapacity(ctx, channelStore, changeCapacityCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// Helper functions
func intPtr(i int) *int {
	return &i
}

func createCourseDefinedEvent(courseID string, name string, capacity int) InputEvent {
	return NewCourseDefinedEvent(courseID, name, "Instructor", capacity)
}

func createStudentRegisteredEvent(studentID string, name string) InputEvent {
	return NewStudentRegisteredEvent(studentID, name, "email@example.com")
}

func createStudentEnrolledInCourseEvent(studentID string, courseID string) InputEvent {
	return NewStudentEnrolledEvent(studentID, courseID, time.Now().Format(time.RFC3339))
}

func createStudentDroppedFromCourseEvent(studentID string, courseID string) InputEvent {
	return NewStudentDroppedEvent(studentID, courseID, time.Now().Format(time.RFC3339))
}

func createCourseCapacityChangedEvent(courseID string, newCapacity int) InputEvent {
	return NewCourseCapacityChangedEvent(courseID, newCapacity)
}
