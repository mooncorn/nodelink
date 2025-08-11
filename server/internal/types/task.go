package types

import (
	"context"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
)

// TaskStatus represents the current state of a task
type TaskStatus int

const (
	TaskStatusCreated TaskStatus = iota
	TaskStatusSent
	TaskStatusInProgress
	TaskStatusCompleted
	TaskStatusFailed
	TaskStatusTimeout
	TaskStatusCancelled
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusCreated:
		return "created"
	case TaskStatusSent:
		return "sent"
	case TaskStatusInProgress:
		return "in_progress"
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusTimeout:
		return "timeout"
	case TaskStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Task represents a task in the system
type Task struct {
	ID        string
	Status    TaskStatus
	Request   *pb.TaskRequest
	Response  *pb.TaskResponse // last response
	CreatedAt time.Time
	UpdatedAt time.Time
	Timeout   time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
}

// TaskEventListener defines a function that processes task events
type TaskEventListener func(task *Task)
