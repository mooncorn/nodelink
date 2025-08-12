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

// EventProcessor interface for processing different types of events
type EventProcessor interface {
	ProcessEvent(event *pb.TaskResponse) (*ProcessedEvent, error)
	GetEventType() string
}

// ProcessedEvent represents the result of event processing
type ProcessedEvent struct {
	OriginalEvent *pb.TaskResponse `json:"original_event"`
	ProcessedData *pb.TaskResponse `json:"processed_data"`
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
