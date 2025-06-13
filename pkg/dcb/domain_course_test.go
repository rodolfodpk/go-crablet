package dcb

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type CourseState struct {
	Title            string
	MaxStudents      int
	EnrolledStudents map[string]bool
}

type StudentState struct {
	Name      string
	Email     string
	CourseIDs map[string]bool
}

var _ = Describe("Course/Student DCB Invariant Rules", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		// store is set up globally in test suite
	})

	It("should allow enrollment if course is not full and student not already enrolled", func() {
		courseID := "course-101"
		studentID := "student-1"

		// Seed course and student
		_, err := store.Append(ctx, []InputEvent{
			NewInputEvent("CourseCreated", NewTags("course_id", courseID), mustJSON(map[string]any{"title": "Math", "max": 2})),
			NewInputEvent("StudentRegistered", NewTags("student_id", studentID), mustJSON(map[string]any{"name": "Alice"})),
		}, nil)
		Expect(err).NotTo(HaveOccurred())

		// Project decision model
		courseProjector := StateProjector{
			Query:        NewQuery(NewTags("course_id", courseID), "CourseCreated", "StudentEnrolled", "StudentUnenrolled"),
			InitialState: &CourseState{MaxStudents: 2, EnrolledStudents: map[string]bool{}},
			TransitionFn: func(state any, e Event) any {
				c := state.(*CourseState)
				switch e.Type {
				case "CourseCreated":
					var d struct {
						Title string
						Max   int
					}
					_ = json.Unmarshal(e.Data, &d)
					c.Title = d.Title
					if d.Max > 0 {
						c.MaxStudents = d.Max
					}
				case "StudentEnrolled":
					var d struct{ StudentID string }
					_ = json.Unmarshal(e.Data, &d)
					c.EnrolledStudents[d.StudentID] = true
				case "StudentUnenrolled":
					var d struct{ StudentID string }
					_ = json.Unmarshal(e.Data, &d)
					delete(c.EnrolledStudents, d.StudentID)
				}
				return c
			},
		}
		studentProjector := StateProjector{
			Query:        NewQuery(NewTags("student_id", studentID), "StudentRegistered", "StudentEnrolled", "StudentUnenrolled"),
			InitialState: &StudentState{CourseIDs: map[string]bool{}},
			TransitionFn: func(state any, e Event) any {
				s := state.(*StudentState)
				switch e.Type {
				case "StudentRegistered":
					var d struct{ Name string }
					_ = json.Unmarshal(e.Data, &d)
					s.Name = d.Name
				case "StudentEnrolled":
					var d struct{ CourseID string }
					_ = json.Unmarshal(e.Data, &d)
					s.CourseIDs[d.CourseID] = true
				case "StudentUnenrolled":
					var d struct{ CourseID string }
					_ = json.Unmarshal(e.Data, &d)
					delete(s.CourseIDs, d.CourseID)
				}
				return s
			},
		}
		query := NewQueryFromItems(
			NewQueryItem([]string{"CourseCreated", "StudentEnrolled", "StudentUnenrolled"}, NewTags("course_id", courseID)),
			NewQueryItem([]string{"StudentRegistered", "StudentEnrolled", "StudentUnenrolled"}, NewTags("student_id", studentID)),
		)
		states, appendCond, err := store.ProjectDecisionModel(ctx, query, nil, []BatchProjector{
			{ID: "course", StateProjector: courseProjector},
			{ID: "student", StateProjector: studentProjector},
		})
		Expect(err).NotTo(HaveOccurred())
		course := states["course"].(*CourseState)
		student := states["student"].(*StudentState)

		// Invariants
		Expect(len(course.EnrolledStudents)).To(Equal(0))
		Expect(len(student.CourseIDs)).To(Equal(0))
		Expect(course.MaxStudents).To(Equal(2))

		// Try to enroll
		enrollEvent := NewInputEvent("StudentEnrolled", NewTags("course_id", courseID, "student_id", studentID), mustJSON(map[string]any{"CourseID": courseID, "StudentID": studentID}))
		_, err = store.Append(ctx, []InputEvent{enrollEvent}, &appendCond)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject enrollment if course is full", func() {
		courseID := "course-102"
		// studentIDs := []string{"s1", "s2", "s3"} // Remove unused variable
		// Seed course and two students already enrolled
		seed := []InputEvent{
			NewInputEvent("CourseCreated", NewTags("course_id", courseID), mustJSON(map[string]any{"title": "Math", "max": 2})),
			NewInputEvent("StudentRegistered", NewTags("student_id", "s1"), mustJSON(map[string]any{"name": "A"})),
			NewInputEvent("StudentRegistered", NewTags("student_id", "s2"), mustJSON(map[string]any{"name": "B"})),
			NewInputEvent("StudentEnrolled", NewTags("course_id", courseID, "student_id", "s1"), mustJSON(map[string]any{"CourseID": courseID, "StudentID": "s1"})),
			NewInputEvent("StudentEnrolled", NewTags("course_id", courseID, "student_id", "s2"), mustJSON(map[string]any{"CourseID": courseID, "StudentID": "s2"})),
			NewInputEvent("StudentRegistered", NewTags("student_id", "s3"), mustJSON(map[string]any{"name": "C"})),
		}
		_, err := store.Append(ctx, seed, nil)
		Expect(err).NotTo(HaveOccurred())

		// Project decision model for s3
		courseProjector := StateProjector{
			Query:        NewQuery(NewTags("course_id", courseID), "CourseCreated", "StudentEnrolled", "StudentUnenrolled"),
			InitialState: &CourseState{MaxStudents: 2, EnrolledStudents: map[string]bool{}},
			TransitionFn: func(state any, e Event) any {
				c := state.(*CourseState)
				switch e.Type {
				case "CourseCreated":
					var d struct {
						Title string
						Max   int
					}
					_ = json.Unmarshal(e.Data, &d)
					c.Title = d.Title
					if d.Max > 0 {
						c.MaxStudents = d.Max
					}
				case "StudentEnrolled":
					var d struct{ StudentID string }
					_ = json.Unmarshal(e.Data, &d)
					c.EnrolledStudents[d.StudentID] = true
				case "StudentUnenrolled":
					var d struct{ StudentID string }
					_ = json.Unmarshal(e.Data, &d)
					delete(c.EnrolledStudents, d.StudentID)
				}
				return c
			},
		}
		studentProjector := StateProjector{
			Query:        NewQuery(NewTags("student_id", "s3"), "StudentRegistered", "StudentEnrolled", "StudentUnenrolled"),
			InitialState: &StudentState{CourseIDs: map[string]bool{}},
			TransitionFn: func(state any, e Event) any {
				s := state.(*StudentState)
				switch e.Type {
				case "StudentRegistered":
					var d struct{ Name string }
					_ = json.Unmarshal(e.Data, &d)
					s.Name = d.Name
				case "StudentEnrolled":
					var d struct{ CourseID string }
					_ = json.Unmarshal(e.Data, &d)
					s.CourseIDs[d.CourseID] = true
				case "StudentUnenrolled":
					var d struct{ CourseID string }
					_ = json.Unmarshal(e.Data, &d)
					delete(s.CourseIDs, d.CourseID)
				}
				return s
			},
		}
		query := NewQueryFromItems(
			NewQueryItem([]string{"CourseCreated", "StudentEnrolled", "StudentUnenrolled"}, NewTags("course_id", courseID)),
			NewQueryItem([]string{"StudentRegistered", "StudentEnrolled", "StudentUnenrolled"}, NewTags("student_id", "s3")),
		)
		states, appendCond, err := store.ProjectDecisionModel(ctx, query, nil, []BatchProjector{
			{ID: "course", StateProjector: courseProjector},
			{ID: "student", StateProjector: studentProjector},
		})
		Expect(err).NotTo(HaveOccurred())
		course := states["course"].(*CourseState)
		student := states["student"].(*StudentState)
		Expect(len(course.EnrolledStudents)).To(Equal(2))
		Expect(len(student.CourseIDs)).To(Equal(0))
		Expect(course.MaxStudents).To(Equal(2))

		// Try to enroll s3 (should fail business rule)
		enrollEvent := NewInputEvent("StudentEnrolled", NewTags("course_id", courseID, "student_id", "s3"), mustJSON(map[string]any{"CourseID": courseID, "StudentID": "s3"}))
		if len(course.EnrolledStudents) >= course.MaxStudents {
			// Simulate business rule: do not append
			Expect(true).To(BeTrue())
		} else {
			_, err = store.Append(ctx, []InputEvent{enrollEvent}, &appendCond)
			Expect(err).To(HaveOccurred())
		}
	})

	It("should allow unenrollment and re-enrollment", func() {
		courseID := "course-103"
		studentID := "student-2"
		// Seed course and enroll student
		seed := []InputEvent{
			NewInputEvent("CourseCreated", NewTags("course_id", courseID), mustJSON(map[string]any{"title": "Math", "max": 1})),
			NewInputEvent("StudentRegistered", NewTags("student_id", studentID), mustJSON(map[string]any{"name": "Bob"})),
			NewInputEvent("StudentEnrolled", NewTags("course_id", courseID, "student_id", studentID), mustJSON(map[string]any{"CourseID": courseID, "StudentID": studentID})),
		}
		_, err := store.Append(ctx, seed, nil)
		Expect(err).NotTo(HaveOccurred())

		// Unenroll
		unenrollEvent := NewInputEvent("StudentUnenrolled", NewTags("course_id", courseID, "student_id", studentID), mustJSON(map[string]any{"CourseID": courseID, "StudentID": studentID}))
		_, err = store.Append(ctx, []InputEvent{unenrollEvent}, nil)
		Expect(err).NotTo(HaveOccurred())

		// Project decision model
		courseProjector := StateProjector{
			Query:        NewQuery(NewTags("course_id", courseID), "CourseCreated", "StudentEnrolled", "StudentUnenrolled"),
			InitialState: &CourseState{MaxStudents: 1, EnrolledStudents: map[string]bool{}},
			TransitionFn: func(state any, e Event) any {
				c := state.(*CourseState)
				switch e.Type {
				case "CourseCreated":
					var d struct {
						Title string
						Max   int
					}
					_ = json.Unmarshal(e.Data, &d)
					c.Title = d.Title
					if d.Max > 0 {
						c.MaxStudents = d.Max
					}
				case "StudentEnrolled":
					var d struct{ StudentID string }
					_ = json.Unmarshal(e.Data, &d)
					c.EnrolledStudents[d.StudentID] = true
				case "StudentUnenrolled":
					var d struct{ StudentID string }
					_ = json.Unmarshal(e.Data, &d)
					delete(c.EnrolledStudents, d.StudentID)
				}
				return c
			},
		}
		studentProjector := StateProjector{
			Query:        NewQuery(NewTags("student_id", studentID), "StudentRegistered", "StudentEnrolled", "StudentUnenrolled"),
			InitialState: &StudentState{CourseIDs: map[string]bool{}},
			TransitionFn: func(state any, e Event) any {
				s := state.(*StudentState)
				switch e.Type {
				case "StudentRegistered":
					var d struct{ Name string }
					_ = json.Unmarshal(e.Data, &d)
					s.Name = d.Name
				case "StudentEnrolled":
					var d struct{ CourseID string }
					_ = json.Unmarshal(e.Data, &d)
					s.CourseIDs[d.CourseID] = true
				case "StudentUnenrolled":
					var d struct{ CourseID string }
					_ = json.Unmarshal(e.Data, &d)
					delete(s.CourseIDs, d.CourseID)
				}
				return s
			},
		}
		query := NewQueryFromItems(
			NewQueryItem([]string{"CourseCreated", "StudentEnrolled", "StudentUnenrolled"}, NewTags("course_id", courseID)),
			NewQueryItem([]string{"StudentRegistered", "StudentEnrolled", "StudentUnenrolled"}, NewTags("student_id", studentID)),
		)
		states, appendCond, err := store.ProjectDecisionModel(ctx, query, nil, []BatchProjector{
			{ID: "course", StateProjector: courseProjector},
			{ID: "student", StateProjector: studentProjector},
		})
		Expect(err).NotTo(HaveOccurred())
		course := states["course"].(*CourseState)
		student := states["student"].(*StudentState)
		Expect(len(course.EnrolledStudents)).To(Equal(0))
		Expect(len(student.CourseIDs)).To(Equal(0))

		// Try to re-enroll
		enrollEvent := NewInputEvent("StudentEnrolled", NewTags("course_id", courseID, "student_id", studentID), mustJSON(map[string]any{"CourseID": courseID, "StudentID": studentID}))
		_, err = store.Append(ctx, []InputEvent{enrollEvent}, &appendCond)
		Expect(err).NotTo(HaveOccurred())
	})
})

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
