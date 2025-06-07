// Package dcb provides domain-specific types and helpers for the workflow domain.
package dcb

import (
	"encoding/json"
	"go-crablet/pkg/dcb"
)

// WorkflowState represents the state of a workflow
type WorkflowState struct {
	CurrentStep    int
	CompletedTasks []string
	FailedTasks    map[string]string
	RetryCount     map[string]int
	IsComplete     bool
}

// WorkflowStartedEvent represents when a workflow starts
type WorkflowStartedEvent struct {
	Step int `json:"step"`
}

// TaskAssignedEvent represents when a task is assigned
type TaskAssignedEvent struct {
	Task string `json:"task"`
}

// TaskCompletedEvent represents when a task is completed
type TaskCompletedEvent struct {
	Task string `json:"task"`
}

// TaskFailedEvent represents when a task fails
type TaskFailedEvent struct {
	Task  string `json:"task"`
	Error string `json:"error"`
}

// TaskRetriedEvent represents when a task is retried
type TaskRetriedEvent struct {
	Task string `json:"task"`
}

// WorkflowCompletedEvent represents when a workflow completes
type WorkflowCompletedEvent struct {
	Step int `json:"step"`
}

// NewWorkflowStartedEvent creates a new workflow started event
func NewWorkflowStartedEvent(step int, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(WorkflowStartedEvent{Step: step})
	return dcb.InputEvent{
		Type: "WorkflowStarted",
		Tags: tags,
		Data: data,
	}
}

// NewTaskAssignedEvent creates a new task assigned event
func NewTaskAssignedEvent(task string, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(TaskAssignedEvent{Task: task})
	return dcb.InputEvent{
		Type: "TaskAssigned",
		Tags: tags,
		Data: data,
	}
}

// NewTaskCompletedEvent creates a new task completed event
func NewTaskCompletedEvent(task string, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(TaskCompletedEvent{Task: task})
	return dcb.InputEvent{
		Type: "TaskCompleted",
		Tags: tags,
		Data: data,
	}
}

// NewTaskFailedEvent creates a new task failed event
func NewTaskFailedEvent(task string, error string, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(TaskFailedEvent{Task: task, Error: error})
	return dcb.InputEvent{
		Type: "TaskFailed",
		Tags: tags,
		Data: data,
	}
}

// NewTaskRetriedEvent creates a new task retried event
func NewTaskRetriedEvent(task string, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(TaskRetriedEvent{Task: task})
	return dcb.InputEvent{
		Type: "TaskRetried",
		Tags: tags,
		Data: data,
	}
}

// NewWorkflowCompletedEvent creates a new workflow completed event
func NewWorkflowCompletedEvent(step int, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(WorkflowCompletedEvent{Step: step})
	return dcb.InputEvent{
		Type: "WorkflowCompleted",
		Tags: tags,
		Data: data,
	}
}

// WorkflowProjector creates a projector for workflow events
func WorkflowProjector(workflowID string) dcb.StateProjector {
	return dcb.StateProjector{
		Query: NewQuery(
			NewTags("workflow_id", workflowID),
			"TaskAssigned", "TaskCompleted", "TaskFailed", "TaskRetried",
		),
		InitialState: &WorkflowState{
			FailedTasks: make(map[string]string),
			RetryCount:  make(map[string]int),
		},
		TransitionFn: func(state any, e dcb.Event) any {
			s := state.(*WorkflowState)
			var data map[string]string
			_ = json.Unmarshal(e.Data, &data)
			taskID := data["task"]

			switch e.Type {
			case "TaskAssigned":
				// No state changes needed
			case "TaskCompleted":
				s.CompletedTasks = append(s.CompletedTasks, taskID)
				delete(s.FailedTasks, taskID)
			case "TaskFailed":
				s.FailedTasks[taskID] = data["error"]
			case "TaskRetried":
				s.RetryCount[taskID]++
				delete(s.FailedTasks, taskID)
			}
			return s
		},
	}
}
