package sse

import (
	"context"
)

// NewClient creates a new client instance
func NewClient[T any](id ClientID, bufferSize int) *Client[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client[T]{
		ID:       id,
		Channel:  make(chan Message[T], bufferSize),
		Context:  ctx,
		Cancel:   cancel,
		Metadata: make(map[string]interface{}),
		Rooms:    make(map[string]bool),
	}
}

// SetMetadata sets metadata for the client (thread-safe)
func (c *Client[T]) SetMetadata(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Metadata[key] = value
}

// GetMetadata gets metadata for the client (thread-safe)
func (c *Client[T]) GetMetadata(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.Metadata[key]
	return value, exists
}

// JoinRoom adds the client to a room (thread-safe)
func (c *Client[T]) JoinRoom(room string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rooms[room] = true
}

// LeaveRoom removes the client from a room (thread-safe)
func (c *Client[T]) LeaveRoom(room string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Rooms, room)
}

// IsInRoom checks if the client is in a room (thread-safe)
func (c *Client[T]) IsInRoom(room string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Rooms[room]
}

// GetRooms returns a copy of all rooms the client is in (thread-safe)
func (c *Client[T]) GetRooms() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rooms := make([]string, 0, len(c.Rooms))
	for room := range c.Rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// GetInfo returns read-only information about the client
func (c *Client[T]) GetInfo() ClientInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metadata := make(map[string]interface{})
	for k, v := range c.Metadata {
		metadata[k] = v
	}

	rooms := make([]string, 0, len(c.Rooms))
	for room := range c.Rooms {
		rooms = append(rooms, room)
	}

	return ClientInfo{
		ID:       c.ID,
		Metadata: metadata,
		Rooms:    rooms,
	}
}

// Close closes the client's channel and cancels its context
func (c *Client[T]) Close() {
	c.Cancel()
	close(c.Channel)
}
