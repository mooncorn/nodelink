package events

import (
	"log"
	"sync"
	"time"

	"github.com/mooncorn/nodelink/server/pkg/interfaces"
)

// Bus implements the EventBus interface for publish-subscribe messaging
type Bus struct {
	mu        sync.RWMutex
	handlers  map[string][]interfaces.EventHandler
	asyncMode bool
}

// NewEventBus creates a new event bus
func NewEventBus(asyncMode bool) *Bus {
	return &Bus{
		handlers:  make(map[string][]interfaces.EventHandler),
		asyncMode: asyncMode,
	}
}

// Publish publishes an event to all subscribers of the event type
func (b *Bus) Publish(event interfaces.Event) {
	b.mu.RLock()
	handlers, exists := b.handlers[event.Type]
	if !exists {
		b.mu.RUnlock()
		return
	}

	// Make a copy of handlers to avoid holding the lock during execution
	handlersCopy := make([]interfaces.EventHandler, len(handlers))
	copy(handlersCopy, handlers)
	b.mu.RUnlock()

	// Execute handlers
	for _, handler := range handlersCopy {
		if b.asyncMode {
			go b.safeExecuteHandler(handler, event)
		} else {
			b.safeExecuteHandler(handler, event)
		}
	}
}

// Subscribe adds an event handler for a specific event type
func (b *Bus) Subscribe(eventType string, handler interfaces.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlers[eventType] == nil {
		b.handlers[eventType] = make([]interfaces.EventHandler, 0)
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Unsubscribe removes an event handler for a specific event type
func (b *Bus) Unsubscribe(eventType string, handler interfaces.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers, exists := b.handlers[eventType]
	if !exists {
		return
	}

	// Remove the handler (this is a simple implementation)
	// In a production system, you might want to use handler IDs
	for i, h := range handlers {
		if &h == &handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// safeExecuteHandler executes a handler with panic recovery
func (b *Bus) safeExecuteHandler(handler interfaces.EventHandler, event interfaces.Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Event handler panic for event type %s: %v", event.Type, r)
		}
	}()

	handler(event)
}

// CreateEvent is a helper function to create events with consistent structure
func CreateEvent(eventType, source string, data interface{}) interfaces.Event {
	return interfaces.Event{
		Type:      eventType,
		Data:      data,
		Source:    source,
		Timestamp: time.Now().Unix(),
	}
}
