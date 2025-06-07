// Package dcb provides domain-specific types and helpers for the course domain.
package dcb

import (
	"encoding/json"
)

// CourseState represents the state of a course
type CourseState struct {
	Title           string
	EnrollmentCount int
	EventCount      int
}

// CourseUserState represents the state of a course and its user interactions
type CourseUserState struct {
	CourseTitle      string
	UserName         string
	EnrollmentStatus string
	UserLevel        string
	EventCount       int
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

// CourseUserProjector creates a projector for course and user events
func CourseUserProjector(courseID string) StateProjector {
	return StateProjector{
		Query: NewQuery(
			NewTags("course_id", courseID),
			"CourseLaunched", "CourseUpdated", "UserRegistered", "UserProfileUpdated",
			"EnrollmentStarted", "EnrollmentCompleted",
		),
		InitialState: &CourseUserState{},
		TransitionFn: func(state any, e Event) any {
			s := state.(*CourseUserState)
			s.EventCount++

			var data map[string]string
			_ = json.Unmarshal(e.Data, &data)

			switch e.Type {
			case "CourseLaunched", "CourseUpdated":
				s.CourseTitle = data["title"]
			case "UserRegistered":
				s.UserName = data["name"]
			case "UserProfileUpdated":
				s.UserLevel = data["level"]
			case "EnrollmentStarted", "EnrollmentCompleted":
				s.EnrollmentStatus = data["status"]
			}
			return s
		},
	}
}
