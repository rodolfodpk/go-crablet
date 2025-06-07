// Package dcb provides domain-specific types and helpers for the course domain.
package dcb

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	maxCoursesPerStudent = 10
	maxStudentsPerCourse = 30
)

// CourseState represents the state of a course
type CourseState struct {
	Title           string
	EnrollmentCount int
	EventCount      int
}

// StudentState represents the state of a student
type StudentState struct {
	Name       string
	CourseIDs  map[string]bool
	EventCount int
}

// CourseAPI handles course enrollment commands
type CourseAPI struct {
	eventStore EventStore
}

// NewCourseAPI creates a new course API instance
func NewCourseAPI(eventStore EventStore) (*CourseAPI, error) {
	if eventStore == nil {
		return nil, fmt.Errorf("event store must not be nil")
	}
	return &CourseAPI{eventStore: eventStore}, nil
}

// EnrollStudentCommand represents the command to enroll a student in a course
type EnrollStudentCommand struct {
	CourseID  string `json:"courseId"`
	StudentID string `json:"studentId"`
}

// EnrollStudent attempts to enroll a student in a course
func (a *CourseAPI) EnrollStudent(ctx context.Context, cmd EnrollStudentCommand) error {
	// Check if student has reached the course limit
	studentProjector := StudentProjector(cmd.StudentID)
	_, studentState, err := a.eventStore.ProjectState(ctx, studentProjector)
	if err != nil {
		return fmt.Errorf("failed to check student state: %w", err)
	}

	student := studentState.(*StudentState)
	if len(student.CourseIDs) >= maxCoursesPerStudent {
		return fmt.Errorf("student %q has reached the maximum limit of %d courses", cmd.StudentID, maxCoursesPerStudent)
	}

	// Check if course has reached the student limit
	courseProjector := CourseProjector(cmd.CourseID)
	_, courseState, err := a.eventStore.ProjectState(ctx, courseProjector)
	if err != nil {
		return fmt.Errorf("failed to check course state: %w", err)
	}

	course := courseState.(*CourseState)
	if course.EnrollmentCount >= maxStudentsPerCourse {
		return fmt.Errorf("course %q has reached the maximum limit of %d students", cmd.CourseID, maxStudentsPerCourse)
	}

	// Create enrollment event
	enrollmentTags := NewTags(
		"course_id", cmd.CourseID,
		"student_id", cmd.StudentID,
	)
	event := NewEnrollmentEvent("active", enrollmentTags)

	// Create a query that includes both course and student tags
	query := Query{
		Tags:       enrollmentTags,
		EventTypes: []string{"Enrollment"},
	}

	// Get current position for the combined stream
	pos, _, err := a.eventStore.ProjectState(ctx, StateProjector{
		Query:        query,
		InitialState: []Event{},
		TransitionFn: func(state any, event Event) any {
			events := state.([]Event)
			return append(events, event)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get stream position: %w", err)
	}

	// Append the enrollment event
	_, err = a.eventStore.AppendEvents(ctx, []InputEvent{event}, query, pos)
	if err != nil {
		return fmt.Errorf("failed to append enrollment event: %w", err)
	}

	return nil
}

// CourseProjector creates a projector for course events
func CourseProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuery(NewTags("course_id", courseID), "CourseLaunched", "CourseUpdated", "Enrollment"),
		InitialState: &CourseState{},
		TransitionFn: func(state any, e Event) any {
			s := state.(*CourseState)
			s.EventCount++

			var data map[string]string
			_ = json.Unmarshal(e.Data, &data)

			switch e.Type {
			case "CourseLaunched", "CourseUpdated":
				s.Title = data["title"]
			case "Enrollment":
				s.EnrollmentCount++
			}
			return s
		},
	}
}

// StudentProjector creates a projector for student events
func StudentProjector(studentID string) StateProjector {
	return StateProjector{
		Query: NewQuery(
			NewTags("student_id", studentID),
			"StudentRegistered", "Enrollment",
		),
		InitialState: &StudentState{
			CourseIDs: make(map[string]bool),
		},
		TransitionFn: func(state any, e Event) any {
			s := state.(*StudentState)
			s.EventCount++

			var data map[string]string
			_ = json.Unmarshal(e.Data, &data)

			switch e.Type {
			case "StudentRegistered":
				s.Name = data["name"]
			case "Enrollment":
				for _, tag := range e.Tags {
					if tag.Key == "course_id" {
						s.CourseIDs[tag.Value] = true
					}
				}
			}
			return s
		},
	}
}

// CourseLaunchedEvent represents when a course is launched
type CourseLaunchedEvent struct {
	Title string `json:"title"`
}

// CourseUpdatedEvent represents when a course is updated
type CourseUpdatedEvent struct {
	Title string `json:"title"`
}

// UserRegisteredEvent represents when a user registers
type UserRegisteredEvent struct {
	Name string `json:"name"`
}

// EnrollmentEvent represents when a user enrolls in a course
type EnrollmentEvent struct {
	Status string `json:"status"`
}

// NewCourseLaunchedEvent creates a new course launched event
func NewCourseLaunchedEvent(title string, tags []Tag) InputEvent {
	data, _ := json.Marshal(CourseLaunchedEvent{Title: title})
	return NewInputEvent(
		"CourseLaunched",
		tags,
		data,
	)
}

// NewCourseUpdatedEvent creates a new course updated event
func NewCourseUpdatedEvent(title string, tags []Tag) InputEvent {
	data, _ := json.Marshal(CourseUpdatedEvent{Title: title})
	return NewInputEvent(
		"CourseUpdated",
		tags,
		data,
	)
}

// NewCourseUserRegisteredEvent creates a new user registered event for course domain
func NewCourseUserRegisteredEvent(userID string, tags []Tag) InputEvent {
	data, _ := json.Marshal(UserRegisteredEvent{Name: "Test User"})
	return NewInputEvent(
		"UserRegistered",
		tags,
		data,
	)
}

// NewEnrollmentEvent creates a new enrollment event
func NewEnrollmentEvent(status string, tags []Tag) InputEvent {
	data, _ := json.Marshal(EnrollmentEvent{Status: status})
	return NewInputEvent(
		"Enrollment",
		tags,
		data,
	)
}

var _ = Describe("Course Domain", func() {
	var (
		api       *CourseAPI
		courseID  string
		studentID string
	)

	BeforeEach(func() {
		// Truncate the events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		var apiErr error
		api, apiErr = NewCourseAPI(store)
		Expect(apiErr).NotTo(HaveOccurred())
		Expect(api).NotTo(BeNil())

		courseID = "course_test_1"
		studentID = "student_test_1"

		// Create a course
		courseTags := NewTags("course_id", courseID)
		courseEvent := NewCourseLaunchedEvent("Test Course", courseTags)
		_, err = store.AppendEvents(ctx, []InputEvent{courseEvent}, NewQuery(courseTags), 0)
		Expect(err).NotTo(HaveOccurred())

		// Register a student
		studentTags := NewTags("student_id", studentID)
		studentEvent := NewCourseUserRegisteredEvent(studentID, studentTags)
		_, err = store.AppendEvents(ctx, []InputEvent{studentEvent}, NewQuery(studentTags), 0)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When enrolling a student in a course", func() {
		It("should succeed when within limits", func() {
			// When
			err := api.EnrollStudent(ctx, EnrollStudentCommand{
				CourseID:  courseID,
				StudentID: studentID,
			})

			// Then
			Expect(err).NotTo(HaveOccurred())

			// Verify course state
			_, courseState, err := store.ProjectState(ctx, CourseProjector(courseID))
			Expect(err).NotTo(HaveOccurred())
			course := courseState.(*CourseState)
			Expect(course.EnrollmentCount).To(Equal(1))

			// Verify student state
			_, studentState, err := store.ProjectState(ctx, StudentProjector(studentID))
			Expect(err).NotTo(HaveOccurred())
			student := studentState.(*StudentState)
			Expect(len(student.CourseIDs)).To(Equal(1))
			Expect(student.CourseIDs[courseID]).To(BeTrue())
		})

		It("should fail when student has reached course limit", func() {
			// Given: Student is enrolled in maxCoursesPerStudent courses
			for i := 0; i < maxCoursesPerStudent; i++ {
				otherCourseID := fmt.Sprintf("course_test_%d", i+2)
				courseTags := NewTags("course_id", otherCourseID)
				courseEvent := NewCourseLaunchedEvent(fmt.Sprintf("Test Course %d", i+2), courseTags)
				_, err := store.AppendEvents(ctx, []InputEvent{courseEvent}, NewQuery(courseTags), 0)
				Expect(err).NotTo(HaveOccurred())

				enrollmentTags := NewTags(
					"course_id", otherCourseID,
					"student_id", studentID,
				)
				enrollmentEvent := NewEnrollmentEvent("active", enrollmentTags)
				_, err = store.AppendEvents(ctx, []InputEvent{enrollmentEvent}, NewQuery(enrollmentTags), 0)
				Expect(err).NotTo(HaveOccurred())
			}

			// When: Try to enroll in one more course
			err := api.EnrollStudent(ctx, EnrollStudentCommand{
				CourseID:  courseID,
				StudentID: studentID,
			})

			// Then
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("maximum limit of %d courses", maxCoursesPerStudent)))
		})

		It("should fail when course has reached student limit", func() {
			// Given: Course has maxStudentsPerCourse students
			for i := 0; i < maxStudentsPerCourse; i++ {
				otherStudentID := fmt.Sprintf("student_test_%d", i+2)
				studentTags := NewTags("student_id", otherStudentID)
				studentEvent := NewCourseUserRegisteredEvent(otherStudentID, studentTags)
				_, err := store.AppendEvents(ctx, []InputEvent{studentEvent}, NewQuery(studentTags), 0)
				Expect(err).NotTo(HaveOccurred())

				enrollmentTags := NewTags(
					"course_id", courseID,
					"student_id", otherStudentID,
				)
				enrollmentEvent := NewEnrollmentEvent("active", enrollmentTags)
				_, err = store.AppendEvents(ctx, []InputEvent{enrollmentEvent}, NewQuery(enrollmentTags), 0)
				Expect(err).NotTo(HaveOccurred())
			}

			// When: Try to enroll one more student
			err := api.EnrollStudent(ctx, EnrollStudentCommand{
				CourseID:  courseID,
				StudentID: studentID,
			})

			// Then
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("maximum limit of %d students", maxStudentsPerCourse)))
		})
	})
})
