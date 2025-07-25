package sse

import (
	"log"
)

// DefaultEventHandler provides a basic implementation of EventHandler
type DefaultEventHandler[T any] struct {
	EnableLogging bool
}

// OnConnect handles client connection events
func (h *DefaultEventHandler[T]) OnConnect(client *Client[T]) {
	if h.EnableLogging {
		log.Printf("Client connected: %s", client.ID)
	}
}

// OnDisconnect handles client disconnection events
func (h *DefaultEventHandler[T]) OnDisconnect(client *Client[T]) {
	if h.EnableLogging {
		log.Printf("Client disconnected: %s", client.ID)
	}
}

// OnMessage handles message events
func (h *DefaultEventHandler[T]) OnMessage(message Message[T]) {
	if h.EnableLogging {
		log.Printf("Message: %+v", message)
	}
}

// OnError handles error events
func (h *DefaultEventHandler[T]) OnError(clientID ClientID, err error) {
	if h.EnableLogging {
		log.Printf("Error for client %s: %v", clientID, err)
	}
}

// NewDefaultEventHandler creates a new default event handler
func NewDefaultEventHandler[T any](enableLogging bool) EventHandler[T] {
	handler := &DefaultEventHandler[T]{EnableLogging: enableLogging}
	return EventHandler[T]{
		OnConnect:    handler.OnConnect,
		OnDisconnect: handler.OnDisconnect,
		OnMessage:    handler.OnMessage,
		OnError:      handler.OnError,
	}
}
