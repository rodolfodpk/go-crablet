package dcb

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Event type definitions
type CourseDefined struct {
	CourseID string `json:"courseId"`
	Capacity int    `json:"capacity"`
}

type CourseCapacityChanged struct {
	CourseID    string `json:"courseId"`
	NewCapacity int    `json:"newCapacity"`
}

type StudentSubscribedToCourse struct {
	StudentID string `json:"studentId"`
	CourseID  string `json:"courseId"`
}

// Event constructors
func NewCourseDefinedEvent(courseID string, capacity int) InputEvent {
	return NewInputEvent("CourseDefined", NewTags("course_id", courseID), mustJSON(CourseDefined{
		CourseID: courseID,
		Capacity: capacity,
	}))
}

func NewCourseCapacityChangedEvent(courseID string, newCapacity int) InputEvent {
	return NewInputEvent("CourseCapacityChanged", NewTags("course_id", courseID), mustJSON(CourseCapacityChanged{
		CourseID:    courseID,
		NewCapacity: newCapacity,
	}))
}

func NewStudentSubscribedToCourseEvent(studentID, courseID string) InputEvent {
	return NewInputEvent("StudentSubscribedToCourse", NewTags("student_id", studentID, "course_id", courseID), mustJSON(StudentSubscribedToCourse{
		StudentID: studentID,
		CourseID:  courseID,
	}))
}

// Projections for decision models
func CourseExistsProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuery(NewTags("course_id", courseID), "CourseDefined"),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		},
	}
}

func CourseCapacityProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuery(NewTags("course_id", courseID), "CourseDefined", "CourseCapacityChanged"),
		InitialState: 0,
		TransitionFn: func(state any, event Event) any {
			switch event.Type {
			case "CourseDefined":
				var data CourseDefined
				_ = json.Unmarshal(event.Data, &data)
				return data.Capacity
			case "CourseCapacityChanged":
				var data CourseCapacityChanged
				_ = json.Unmarshal(event.Data, &data)
				return data.NewCapacity
			}
			return state
		},
	}
}

func StudentAlreadySubscribedProjector(studentID, courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuery(NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribedToCourse"),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		},
	}
}

func NumberOfCourseSubscriptionsProjector(courseID string) StateProjector {
	return StateProjector{
		Query:        NewQuery(NewTags("course_id", courseID), "StudentSubscribedToCourse"),
		InitialState: 0,
		TransitionFn: func(state any, event Event) any {
			return state.(int) + 1
		},
	}
}

func NumberOfStudentSubscriptionsProjector(studentID string) StateProjector {
	return StateProjector{
		Query:        NewQuery(NewTags("student_id", studentID), "StudentSubscribedToCourse"),
		InitialState: 0,
		TransitionFn: func(state any, event Event) any {
			return state.(int) + 1
		},
	}
}

// API class for command handlers
type CourseAPI struct {
	eventStore EventStore
}

func NewCourseAPI(eventStore EventStore) *CourseAPI {
	return &CourseAPI{eventStore: eventStore}
}

func (api *CourseAPI) DefineCourse(courseID string, capacity int) error {
	projectors := []BatchProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(courseID)},
	}

	query := NewQuery(NewTags("course_id", courseID), "CourseDefined")
	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), query, nil, projectors)
	if err != nil {
		return fmt.Errorf("failed to project course state: %w", err)
	}

	if states["courseExists"].(bool) {
		return fmt.Errorf("course with id \"%s\" already exists", courseID)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewCourseDefinedEvent(courseID, capacity),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to append course defined event: %w", err)
	}

	return nil
}

func (api *CourseAPI) ChangeCourseCapacity(courseID string, newCapacity int) error {
	projectors := []BatchProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(courseID)},
		{ID: "courseCapacity", StateProjector: CourseCapacityProjector(courseID)},
	}

	query := NewQuery(NewTags("course_id", courseID), "CourseDefined", "CourseCapacityChanged")
	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), query, nil, projectors)
	if err != nil {
		return fmt.Errorf("failed to project course state: %w", err)
	}

	if !states["courseExists"].(bool) {
		return fmt.Errorf("course \"%s\" does not exist", courseID)
	}

	currentCapacity := states["courseCapacity"].(int)
	if currentCapacity == newCapacity {
		return fmt.Errorf("new capacity %d is the same as the current capacity", newCapacity)
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewCourseCapacityChangedEvent(courseID, newCapacity),
	}, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to append capacity changed event: %w", err)
	}

	return nil
}

func (api *CourseAPI) SubscribeStudentToCourse(studentID, courseID string) error {
	projectors := []BatchProjector{
		{ID: "courseExists", StateProjector: CourseExistsProjector(courseID)},
		{ID: "courseCapacity", StateProjector: CourseCapacityProjector(courseID)},
		{ID: "numberOfCourseSubscriptions", StateProjector: NumberOfCourseSubscriptionsProjector(courseID)},
		{ID: "numberOfStudentSubscriptions", StateProjector: NumberOfStudentSubscriptionsProjector(studentID)},
		{ID: "studentAlreadySubscribed", StateProjector: StudentAlreadySubscribedProjector(studentID, courseID)},
	}

	// Use a more specific query that only includes events relevant to the current operation
	// This avoids optimistic locking issues when subscribing to multiple courses
	query := NewQueryFromItems(
		NewQueryItem([]string{"CourseDefined", "CourseCapacityChanged"}, NewTags("course_id", courseID)),
		NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("course_id", courseID)),
		NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("student_id", studentID)),
		NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("student_id", studentID, "course_id", courseID)),
	)

	states, appendCondition, err := api.eventStore.ProjectDecisionModel(context.Background(), query, nil, projectors)
	if err != nil {
		return fmt.Errorf("failed to project subscription state: %w", err)
	}

	if !states["courseExists"].(bool) {
		return fmt.Errorf("course \"%s\" does not exist", courseID)
	}

	numberOfSubscriptions := states["numberOfCourseSubscriptions"].(int)
	courseCapacity := states["courseCapacity"].(int)
	if numberOfSubscriptions >= courseCapacity {
		return fmt.Errorf("course \"%s\" is already fully booked", courseID)
	}

	if states["studentAlreadySubscribed"].(bool) {
		return fmt.Errorf("student already subscribed to this course")
	}

	numberOfStudentSubscriptions := states["numberOfStudentSubscriptions"].(int)
	if numberOfStudentSubscriptions >= 5 {
		return fmt.Errorf("student already subscribed to 5 courses")
	}

	// For the append condition, we only need to check for conflicts on the specific course and student
	// This avoids the optimistic locking issue when subscribing to multiple courses
	specificAppendCondition := AppendCondition{
		FailIfEventsMatch: &Query{
			Items: []QueryItem{
				NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("student_id", studentID, "course_id", courseID)),
			},
		},
		After: appendCondition.After, // Keep the same After field for optimistic locking
	}

	_, err = api.eventStore.Append(context.Background(), []InputEvent{
		NewStudentSubscribedToCourseEvent(studentID, courseID),
	}, &specificAppendCondition)
	if err != nil {
		return fmt.Errorf("failed to append subscription event: %w", err)
	}

	return nil
}

// Test scenarios
var _ = Describe("Multiple Subscriptions", func() {
	var (
		api *CourseAPI
	)

	BeforeEach(func() {
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
		api = NewCourseAPI(store)
	})

	It("should define course successfully", func() {
		err := api.DefineCourse("course-define-1", 10)
		Expect(err).NotTo(HaveOccurred())

		// Verify course exists
		projectors := []BatchProjector{
			{ID: "courseExists", StateProjector: CourseExistsProjector("course-define-1")},
		}
		query := NewQuery(NewTags("course_id", "course-define-1"), "CourseDefined")
		states, _, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states["courseExists"].(bool)).To(BeTrue())
	})

	It("should fail to define duplicate course", func() {
		err := api.DefineCourse("course-duplicate-1", 10)
		Expect(err).NotTo(HaveOccurred())

		err = api.DefineCourse("course-duplicate-1", 15)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists"))
	})

	It("should change course capacity successfully", func() {
		err := api.DefineCourse("course-capacity-1", 10)
		Expect(err).NotTo(HaveOccurred())

		err = api.ChangeCourseCapacity("course-capacity-1", 20)
		Expect(err).NotTo(HaveOccurred())

		// Verify capacity changed
		projectors := []BatchProjector{
			{ID: "courseCapacity", StateProjector: CourseCapacityProjector("course-capacity-1")},
		}
		query := NewQuery(NewTags("course_id", "course-capacity-1"), "CourseDefined", "CourseCapacityChanged")
		states, _, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states["courseCapacity"].(int)).To(Equal(20))
	})

	It("should fail to change capacity to same value", func() {
		err := api.DefineCourse("course-same-capacity-1", 10)
		Expect(err).NotTo(HaveOccurred())

		err = api.ChangeCourseCapacity("course-same-capacity-1", 10)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("same as the current capacity"))
	})

	It("should fail to change capacity of non-existent course", func() {
		err := api.ChangeCourseCapacity("non-existent", 30)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not exist"))
	})

	It("should subscribe student successfully", func() {
		err := api.DefineCourse("course-subscribe-1", 10)
		Expect(err).NotTo(HaveOccurred())

		err = api.SubscribeStudentToCourse("student-subscribe-1", "course-subscribe-1")
		Expect(err).NotTo(HaveOccurred())

		// Verify subscription
		projectors := []BatchProjector{
			{ID: "studentAlreadySubscribed", StateProjector: StudentAlreadySubscribedProjector("student-subscribe-1", "course-subscribe-1")},
			{ID: "numberOfCourseSubscriptions", StateProjector: NumberOfCourseSubscriptionsProjector("course-subscribe-1")},
			{ID: "numberOfStudentSubscriptions", StateProjector: NumberOfStudentSubscriptionsProjector("student-subscribe-1")},
		}
		query := NewQueryFromItems(
			NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("course_id", "course-subscribe-1")),
			NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("student_id", "student-subscribe-1")),
			NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("student_id", "student-subscribe-1", "course_id", "course-subscribe-1")),
		)
		states, _, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states["studentAlreadySubscribed"].(bool)).To(BeTrue())
		Expect(states["numberOfCourseSubscriptions"].(int)).To(Equal(1))
		Expect(states["numberOfStudentSubscriptions"].(int)).To(Equal(1))
	})

	It("should fail to subscribe same student twice", func() {
		err := api.DefineCourse("course-duplicate-sub-1", 10)
		Expect(err).NotTo(HaveOccurred())

		err = api.SubscribeStudentToCourse("student-duplicate-sub-1", "course-duplicate-sub-1")
		Expect(err).NotTo(HaveOccurred())

		err = api.SubscribeStudentToCourse("student-duplicate-sub-1", "course-duplicate-sub-1")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already subscribed to this course"))
	})

	It("should subscribe multiple students", func() {
		err := api.DefineCourse("course-multiple-1", 20)
		Expect(err).NotTo(HaveOccurred())

		// Subscribe 5 students individually (no loop to avoid concurrency issues)
		err = api.SubscribeStudentToCourse("student-multiple-1", "course-multiple-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-multiple-2", "course-multiple-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-multiple-3", "course-multiple-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-multiple-4", "course-multiple-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-multiple-5", "course-multiple-1")
		Expect(err).NotTo(HaveOccurred())

		// Verify course has 5 subscriptions
		projectors := []BatchProjector{
			{ID: "numberOfCourseSubscriptions", StateProjector: NumberOfCourseSubscriptionsProjector("course-multiple-1")},
			{ID: "courseCapacity", StateProjector: CourseCapacityProjector("course-multiple-1")},
		}
		query := NewQueryFromItems(
			NewQueryItem([]string{"StudentSubscribedToCourse"}, NewTags("course_id", "course-multiple-1")),
			NewQueryItem([]string{"CourseDefined", "CourseCapacityChanged"}, NewTags("course_id", "course-multiple-1")),
		)
		states, _, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states["numberOfCourseSubscriptions"].(int)).To(Equal(5))
		Expect(states["courseCapacity"].(int)).To(Equal(20))
	})

	It("should fail to subscribe when course is full", func() {
		err := api.DefineCourse("course-full-1", 2)
		Expect(err).NotTo(HaveOccurred())

		// Subscribe 2 students
		err = api.SubscribeStudentToCourse("student-full-1", "course-full-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-full-2", "course-full-1")
		Expect(err).NotTo(HaveOccurred())

		// Try to subscribe a 3rd student
		err = api.SubscribeStudentToCourse("student-full-3", "course-full-1")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already fully booked"))
	})

	It("should allow student to subscribe to multiple courses", func() {
		// Define two courses
		err := api.DefineCourse("course-multi-1", 5)
		Expect(err).NotTo(HaveOccurred())
		err = api.DefineCourse("course-multi-2", 5)
		Expect(err).NotTo(HaveOccurred())

		// Subscribe student-1 to both courses
		err = api.SubscribeStudentToCourse("student-multi-1", "course-multi-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-multi-1", "course-multi-2")
		Expect(err).NotTo(HaveOccurred())

		// Verify student has 2 subscriptions
		projectors := []BatchProjector{
			{ID: "numberOfStudentSubscriptions", StateProjector: NumberOfStudentSubscriptionsProjector("student-multi-1")},
		}
		query := NewQuery(NewTags("student_id", "student-multi-1"), "StudentSubscribedToCourse")
		states, _, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states["numberOfStudentSubscriptions"].(int)).To(Equal(2))
	})

	It("should limit student to 5 courses maximum", func() {
		// Define 6 courses
		for i := 1; i <= 6; i++ {
			err := api.DefineCourse(fmt.Sprintf("course-limit-%d", i), 1)
			Expect(err).NotTo(HaveOccurred())
		}

		// Subscribe student-1 to 5 courses individually (no loop to avoid concurrency issues)
		err := api.SubscribeStudentToCourse("student-limit-1", "course-limit-1")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-limit-1", "course-limit-2")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-limit-1", "course-limit-3")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-limit-1", "course-limit-4")
		Expect(err).NotTo(HaveOccurred())
		err = api.SubscribeStudentToCourse("student-limit-1", "course-limit-5")
		Expect(err).NotTo(HaveOccurred())

		// Verify student has 5 subscriptions
		projectors := []BatchProjector{
			{ID: "numberOfStudentSubscriptions", StateProjector: NumberOfStudentSubscriptionsProjector("student-limit-1")},
		}
		query := NewQuery(NewTags("student_id", "student-limit-1"), "StudentSubscribedToCourse")
		states, _, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states["numberOfStudentSubscriptions"].(int)).To(Equal(5))

		// Try to subscribe to a 6th course (should fail)
		err = api.SubscribeStudentToCourse("student-limit-1", "course-limit-6")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already subscribed to 5 courses"))
	})

	It("should fail to subscribe to non-existent course", func() {
		err := api.SubscribeStudentToCourse("student-nonexistent-1", "non-existent")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not exist"))
	})
})
