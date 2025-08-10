package interfaces

import (
	"errors"

	pb "github.com/mooncorn/nodelink/proto"
)

// Errors
var (
	ErrStreamTypeNotFound = errors.New("stream type not found")
	ErrStreamNotFound     = errors.New("stream not found")
	ErrInvalidData        = errors.New("invalid data type")
)

// Event represents a system event
type Event struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Source    string      `json:"source"`
	Timestamp int64       `json:"timestamp"`
}

// EventHandler handles events of specific types
type EventHandler func(event Event)

// EventBus interface for publish-subscribe messaging
type EventBus interface {
	Publish(event Event)
	Subscribe(eventType string, handler EventHandler)
	Unsubscribe(eventType string, handler EventHandler)
}

// TaskSender interface for sending tasks to agents
// This interface breaks the circular dependency between TaskManager and TaskServer
type TaskSender interface {
	SendTask(request *pb.TaskRequest) error
}

// ResponseReceiver interface for handling task responses
type ResponseReceiver interface {
	HandleResponse(response *pb.TaskResponse)
}

// EventProcessor interface for processing different types of events
type EventProcessor interface {
	ProcessEvent(event *pb.TaskResponse) (*ProcessedEvent, error)
	GetEventType() string
}

// ProcessedEvent represents the result of event processing
type ProcessedEvent struct {
	OriginalEvent *pb.TaskResponse `json:"original_event"`
	ProcessedData interface{}      `json:"processed_data"`
	ShouldRelay   bool             `json:"should_relay"`
	TargetRoom    string           `json:"target_room"`
	EventType     string           `json:"event_type"`
}

// StreamManager interface for managing different stream types
type StreamManager interface {
	CreateStream(streamType, resourceID string) error
	CloseStream(streamType, resourceID string) error
	SendToStream(streamType, resourceID string, data interface{}) error
}
