package setup

import (
	"encoding/json"

	"go-crablet/pkg/dcb"
)

// CourseState represents the projected state of a course
type CourseState struct {
	CourseID      string `json:"courseId"`
	Name          string `json:"name"`
	Capacity      int    `json:"capacity"`
	Instructor    string `json:"instructor"`
	EnrolledCount int    `json:"enrolledCount"`
	Exists        bool   `json:"exists"`
}

// StudentState represents the projected state of a student
type StudentState struct {
	StudentID       string   `json:"studentId"`
	Name            string   `json:"name"`
	Email           string   `json:"email"`
	EnrolledCourses []string `json:"enrolledCourses"`
	Exists          bool     `json:"exists"`
}

// EnrollmentState represents the enrollment state between a student and course
type EnrollmentState struct {
	StudentID  string `json:"studentId"`
	CourseID   string `json:"courseId"`
	IsEnrolled bool   `json:"isEnrolled"`
}

// CourseDefined represents the data in a CourseDefined event
type CourseDefined struct {
	CourseID   string `json:"courseId"`
	Name       string `json:"name"`
	Capacity   int    `json:"capacity"`
	Instructor string `json:"instructor"`
}

// StudentRegistered represents the data in a StudentRegistered event
type StudentRegistered struct {
	StudentID string `json:"studentId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// StudentEnrolledInCourse represents the data in a StudentEnrolledInCourse event
type StudentEnrolledInCourse struct {
	StudentID  string `json:"studentId"`
	CourseID   string `json:"courseId"`
	EnrolledAt string `json:"enrolledAt"`
}

// StudentDroppedFromCourse represents the data in a StudentDroppedFromCourse event
type StudentDroppedFromCourse struct {
	StudentID string `json:"studentId"`
	CourseID  string `json:"courseId"`
	DroppedAt string `json:"droppedAt"`
}

// CourseCapacityChanged represents the data in a CourseCapacityChanged event
type CourseCapacityChanged struct {
	CourseID    string `json:"courseId"`
	NewCapacity int    `json:"newCapacity"`
}

// CreateCourseExistsProjector creates a projector that tracks if a course exists
func CreateCourseExistsProjector(courseID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "courseExists",
		Query:        dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
		InitialState: false,
		TransitionFn: func(state any, event dcb.Event) any {
			return true
		},
	}
}

// CreateCourseStateProjector creates a projector that tracks the complete state of a course
func CreateCourseStateProjector(courseID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "courseState",
		Query:        dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined", "CourseCapacityChanged"),
		InitialState: &CourseState{CourseID: courseID, Exists: false},
		TransitionFn: func(state any, event dcb.Event) any {
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
			}
			return course
		},
	}
}

// CreateCourseEnrollmentCountProjector creates a projector that counts enrollments for a course
func CreateCourseEnrollmentCountProjector(courseID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "courseEnrollmentCount",
		Query:        dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
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

// CreateStudentExistsProjector creates a projector that tracks if a student exists
func CreateStudentExistsProjector(studentID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "studentExists",
		Query:        dcb.NewQuery(dcb.NewTags("student_id", studentID), "StudentRegistered"),
		InitialState: false,
		TransitionFn: func(state any, event dcb.Event) any {
			return true
		},
	}
}

// CreateStudentStateProjector creates a projector that tracks the complete state of a student
func CreateStudentStateProjector(studentID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "studentState",
		Query:        dcb.NewQuery(dcb.NewTags("student_id", studentID), "StudentRegistered"),
		InitialState: &StudentState{StudentID: studentID, Exists: false},
		TransitionFn: func(state any, event dcb.Event) any {
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

// CreateStudentEnrollmentCountProjector creates a projector that counts enrollments for a student
func CreateStudentEnrollmentCountProjector(studentID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "studentEnrollmentCount",
		Query:        dcb.NewQuery(dcb.NewTags("student_id", studentID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
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

// CreateStudentEnrollmentStateProjector creates a projector that tracks enrollment state between a student and course
func CreateStudentEnrollmentStateProjector(studentID, courseID string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "studentEnrollmentState",
		Query:        dcb.NewQuery(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentEnrolledInCourse", "StudentDroppedFromCourse"),
		InitialState: &EnrollmentState{StudentID: studentID, CourseID: courseID, IsEnrolled: false},
		TransitionFn: func(state any, event dcb.Event) any {
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

// CreateSimpleCountProjector creates a simple count projector for any event type
func CreateSimpleCountProjector(eventType string, tagKey, tagValue string) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "simpleCount",
		Query:        dcb.NewQuery(dcb.NewTags(tagKey, tagValue), eventType),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}
}

// CreateMultiTagCountProjector creates a count projector for events matching multiple tags
func CreateMultiTagCountProjector(eventType string, tags []dcb.Tag) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "multiTagCount",
		Query:        dcb.NewQuery(tags, eventType),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}
}

// CreateValueProjector creates a projector that extracts a value from event data
func CreateValueProjector(eventType string, tagKey, tagValue string, valueExtractor func(dcb.Event) any) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "valueProjector",
		Query:        dcb.NewQuery(dcb.NewTags(tagKey, tagValue), eventType),
		InitialState: nil,
		TransitionFn: func(state any, event dcb.Event) any {
			return valueExtractor(event)
		},
	}
}

// CreateBooleanProjector creates a projector that tracks a boolean state
func CreateBooleanProjector(eventType string, tagKey, tagValue string, setToTrue bool) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "booleanProjector",
		Query:        dcb.NewQuery(dcb.NewTags(tagKey, tagValue), eventType),
		InitialState: false,
		TransitionFn: func(state any, event dcb.Event) any {
			return setToTrue
		},
	}
}

// CreateMapProjector creates a projector that maintains a map of entities
func CreateMapProjector(eventType string, tagKey, tagValue string, keyExtractor func(dcb.Event) string, valueExtractor func(dcb.Event) any) dcb.StateProjector {
	return dcb.StateProjector{
		ID:           "mapProjector",
		Query:        dcb.NewQuery(dcb.NewTags(tagKey, tagValue), eventType),
		InitialState: make(map[string]any),
		TransitionFn: func(state any, event dcb.Event) any {
			m := state.(map[string]any)
			key := keyExtractor(event)
			m[key] = valueExtractor(event)
			return m
		},
	}
}

// CreateBenchmarkProjectors creates a set of projectors for benchmark testing
func CreateBenchmarkProjectors(dataset *Dataset) []dcb.StateProjector {
	projectors := []dcb.StateProjector{
		CreateSimpleCountProjector("CourseDefined", "", ""),
		CreateSimpleCountProjector("StudentRegistered", "", ""),
		CreateSimpleCountProjector("StudentEnrolledInCourse", "", ""),
		CreateSimpleCountProjector("StudentDroppedFromCourse", "", ""),
	}

	// Add a few specific course and student projectors
	if len(dataset.Courses) > 0 {
		courseID := dataset.Courses[0].ID
		projectors = append(projectors, CreateCourseStateProjector(courseID))
		projectors = append(projectors, CreateCourseEnrollmentCountProjector(courseID))
	}

	if len(dataset.Students) > 0 {
		studentID := dataset.Students[0].ID
		projectors = append(projectors, CreateStudentStateProjector(studentID))
		projectors = append(projectors, CreateStudentEnrollmentCountProjector(studentID))
	}

	return projectors
}
