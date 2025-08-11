package events

import (
	"log"
	"sync"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/interfaces"
	"github.com/mooncorn/nodelink/server/internal/metrics"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// EventRouter handles centralized event processing
type EventRouter struct {
	mu         sync.RWMutex
	processors map[string]interfaces.EventProcessor
	sseManager *sse.Manager[*pb.TaskResponse]
	metrics    *metrics.MetricsStore
}

// NewEventRouter creates a new event router
func NewEventRouter(sseManager *sse.Manager[*pb.TaskResponse], metricsStore *metrics.MetricsStore) *EventRouter {
	router := &EventRouter{
		processors: make(map[string]interfaces.EventProcessor),
		sseManager: sseManager,
		metrics:    metricsStore,
	}

	return router
}

// RegisterProcessor registers an event processor
func (er *EventRouter) RegisterProcessor(processor interfaces.EventProcessor) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.processors[processor.GetEventType()] = processor
}

// ProcessAndRelay processes an event and relays it to appropriate channels
func (er *EventRouter) ProcessAndRelay(response *pb.TaskResponse) {
	er.mu.RLock()
	eventType := er.getEventType(response)
	processor, exists := er.processors[eventType]
	er.mu.RUnlock()

	if !exists {
		// Default: relay as-is
		er.relayRaw(response)
		return
	}

	processed, err := processor.ProcessEvent(response)
	if err != nil {
		log.Printf("Event processing failed for type %s: %v", eventType, err)
		er.relayRaw(response) // Fallback to raw relay
		return
	}

	if processed.ShouldRelay {
		er.relayProcessed(processed)
	}
}

// getEventType determines the event type from a TaskResponse
func (er *EventRouter) getEventType(response *pb.TaskResponse) string {
	if response.GetShellExecute() != nil {
		return "shell_output"
	}
	if response.GetMetricsResponse() != nil {
		return "metrics"
	}
	if response.GetTaskCancel() != nil {
		return "task_cancel"
	}

	return "unknown"
}

// relayRaw sends the raw response to SSE without processing
func (er *EventRouter) relayRaw(response *pb.TaskResponse) {
	// Send to task-specific room
	er.sseManager.SendToRoom(response.TaskId, response, "response")
}

// relayProcessed sends a processed event to appropriate channels
func (er *EventRouter) relayProcessed(processed *interfaces.ProcessedEvent) {
	// Send processed data to target room
	if processed.TargetRoom != "" {
		er.sseManager.SendToRoom(processed.TargetRoom, processed.OriginalEvent, processed.EventType)
	}

	// Also send to the original task room for compatibility
	er.sseManager.SendToRoom(processed.OriginalEvent.TaskId, processed.OriginalEvent, "response")
}
