package sse

import (
	"context"
	"sync"
)

// ClientID represents a unique identifier for a client
type ClientID string

// Message represents a generic message that can be sent to clients
type Message[T any] struct {
	Data      T        `json:"data"`
	EventType string   `json:"event_type,omitempty"`
	To        ClientID `json:"to,omitempty"`   // Specific client ID, empty for broadcast
	From      ClientID `json:"from,omitempty"` // Sender client ID
	Room      string   `json:"room,omitempty"` // Room/group identifier
}

// Client represents a connected SSE client
type Client[T any] struct {
	ID       ClientID               `json:"id"`
	Channel  chan Message[T]        `json:"-"`
	Context  context.Context        `json:"-"`
	Cancel   context.CancelFunc     `json:"-"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Rooms    map[string]bool        `json:"rooms,omitempty"`
	mu       sync.RWMutex
}

// ClientInfo provides read-only information about a client
type ClientInfo struct {
	ID       ClientID               `json:"id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Rooms    []string               `json:"rooms,omitempty"`
}

// EventHandler defines callback functions for client events
type EventHandler[T any] struct {
	OnConnect    func(client *Client[T])
	OnDisconnect func(client *Client[T])
	OnMessage    func(message Message[T])
	OnError      func(clientID ClientID, err error)
}

// ManagerConfig holds configuration for the SSE manager
type ManagerConfig struct {
	BufferSize     int  `json:"buffer_size"`
	EnableRooms    bool `json:"enable_rooms"`
	EnableMetadata bool `json:"enable_metadata"`
}
