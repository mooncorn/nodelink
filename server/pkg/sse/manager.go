package sse

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Manager manages SSE connections and message broadcasting
type Manager[T any] struct {
	clients      map[ClientID]*Client[T]
	rooms        map[string]map[ClientID]*Client[T]
	clientsMu    sync.RWMutex
	roomsMu      sync.RWMutex
	config       ManagerConfig
	eventHandler EventHandler[T]
	messageQueue chan Message[T]
	running      atomic.Bool
	done         chan struct{}
}

// NewManager creates a new SSE manager
func NewManager[T any](config ManagerConfig, handler EventHandler[T]) *Manager[T] {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}

	manager := &Manager[T]{
		clients:      make(map[ClientID]*Client[T]),
		rooms:        make(map[string]map[ClientID]*Client[T]),
		config:       config,
		eventHandler: handler,
		messageQueue: make(chan Message[T], config.BufferSize*10),
		done:         make(chan struct{}),
	}

	return manager
}

// Start starts the message processing goroutine
func (m *Manager[T]) Start() {
	if m.running.Swap(true) {
		return // Already running
	}

	go m.processMessages()
}

// Stop stops the manager and closes all connections
func (m *Manager[T]) Stop() {
	if !m.running.Swap(false) {
		return // Already stopped
	}

	close(m.done)

	// Close all client connections
	m.clientsMu.Lock()
	for _, client := range m.clients {
		client.Close()
	}
	m.clients = make(map[ClientID]*Client[T])
	m.clientsMu.Unlock()

	// Clear rooms
	m.roomsMu.Lock()
	m.rooms = make(map[string]map[ClientID]*Client[T])
	m.roomsMu.Unlock()
}

// AddClient adds a new client to the manager
func (m *Manager[T]) AddClient(clientID ClientID) *Client[T] {
	if clientID == "" {
		// Simple client ID generation using timestamp and atomic counter
		timestamp := time.Now().UnixNano()
		clientID = ClientID("client_" + strconv.FormatInt(timestamp, 36))
	}

	client := NewClient[T](clientID, m.config.BufferSize)

	m.clientsMu.Lock()
	m.clients[clientID] = client
	m.clientsMu.Unlock()

	if m.eventHandler.OnConnect != nil {
		m.eventHandler.OnConnect(client)
	}

	return client
}

// RemoveClient removes a client from the manager
func (m *Manager[T]) RemoveClient(clientID ClientID) {
	m.clientsMu.Lock()
	client, exists := m.clients[clientID]
	if exists {
		delete(m.clients, clientID)
	}
	m.clientsMu.Unlock()

	if exists {
		// Remove from all rooms
		if m.config.EnableRooms {
			m.removeClientFromAllRooms(clientID)
		}

		if m.eventHandler.OnDisconnect != nil {
			m.eventHandler.OnDisconnect(client)
		}

		client.Close()
	}
}

// GetClient returns a client by ID
func (m *Manager[T]) GetClient(clientID ClientID) (*Client[T], bool) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	client, exists := m.clients[clientID]
	return client, exists
}

// GetAllClients returns information about all connected clients
func (m *Manager[T]) GetAllClients() []ClientInfo {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	clients := make([]ClientInfo, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client.GetInfo())
	}
	return clients
}

// GetClientCount returns the number of connected clients
func (m *Manager[T]) GetClientCount() int {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	return len(m.clients)
}

// SendToClient sends a message to a specific client
func (m *Manager[T]) SendToClient(clientID ClientID, data T, eventType string) error {
	message := Message[T]{
		Data:      data,
		EventType: eventType,
		To:        clientID,
	}

	select {
	case m.messageQueue <- message:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// Broadcast sends a message to all connected clients
func (m *Manager[T]) Broadcast(data T, eventType string) error {
	message := Message[T]{
		Data:      data,
		EventType: eventType,
	}

	select {
	case m.messageQueue <- message:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// SendToRoom sends a message to all clients in a specific room
func (m *Manager[T]) SendToRoom(room string, data T, eventType string) error {
	if !m.config.EnableRooms {
		return fmt.Errorf("rooms are not enabled")
	}

	message := Message[T]{
		Data:      data,
		EventType: eventType,
		Room:      room,
	}

	select {
	case m.messageQueue <- message:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// JoinRoom adds a client to a room
func (m *Manager[T]) JoinRoom(clientID ClientID, room string) error {
	if !m.config.EnableRooms {
		return fmt.Errorf("rooms are not enabled")
	}

	client, exists := m.GetClient(clientID)
	if !exists {
		return fmt.Errorf("client not found")
	}

	client.JoinRoom(room)

	m.roomsMu.Lock()
	if m.rooms[room] == nil {
		m.rooms[room] = make(map[ClientID]*Client[T])
	}
	m.rooms[room][clientID] = client
	m.roomsMu.Unlock()

	return nil
}

// LeaveRoom removes a client from a room
func (m *Manager[T]) LeaveRoom(clientID ClientID, room string) error {
	if !m.config.EnableRooms {
		return fmt.Errorf("rooms are not enabled")
	}

	client, exists := m.GetClient(clientID)
	if !exists {
		return fmt.Errorf("client not found")
	}

	client.LeaveRoom(room)

	m.roomsMu.Lock()
	if m.rooms[room] != nil {
		delete(m.rooms[room], clientID)
		if len(m.rooms[room]) == 0 {
			delete(m.rooms, room)
		}
	}
	m.roomsMu.Unlock()

	return nil
}

// processMessages processes queued messages
func (m *Manager[T]) processMessages() {
	for {
		select {
		case message := <-m.messageQueue:
			m.handleMessage(message)
		case <-m.done:
			return
		}
	}
}

// handleMessage handles a single message
func (m *Manager[T]) handleMessage(message Message[T]) {
	if m.eventHandler.OnMessage != nil {
		m.eventHandler.OnMessage(message)
	}

	// Route message based on type
	if message.To != "" {
		// Send to specific client
		m.sendToSpecificClient(message)
	} else if message.Room != "" {
		// Send to room
		m.sendToRoomClients(message)
	} else {
		// Broadcast to all
		m.broadcastToAll(message)
	}
}

// sendToSpecificClient sends message to a specific client
func (m *Manager[T]) sendToSpecificClient(message Message[T]) {
	m.clientsMu.RLock()
	client, exists := m.clients[message.To]
	m.clientsMu.RUnlock()

	if !exists {
		if m.eventHandler.OnError != nil {
			m.eventHandler.OnError(message.To, fmt.Errorf("client not found"))
		}
		return
	}

	select {
	case client.Channel <- message:
	case <-client.Context.Done():
		// Client disconnected, remove it
		m.RemoveClient(message.To)
	default:
		// Channel is full, client might be slow
		if m.eventHandler.OnError != nil {
			m.eventHandler.OnError(message.To, fmt.Errorf("client channel is full"))
		}
	}
}

// sendToRoomClients sends message to all clients in a room
func (m *Manager[T]) sendToRoomClients(message Message[T]) {
	m.roomsMu.RLock()
	roomClients, exists := m.rooms[message.Room]
	if !exists {
		m.roomsMu.RUnlock()
		return
	}

	// Create a copy to avoid holding the lock while sending
	clients := make(map[ClientID]*Client[T])
	for id, client := range roomClients {
		clients[id] = client
	}
	m.roomsMu.RUnlock()

	for clientID, client := range clients {
		select {
		case client.Channel <- message:
		case <-client.Context.Done():
			// Client disconnected, remove it
			m.RemoveClient(clientID)
		default:
			// Channel is full, skip this client
			if m.eventHandler.OnError != nil {
				m.eventHandler.OnError(clientID, fmt.Errorf("client channel is full"))
			}
		}
	}
}

// broadcastToAll sends message to all connected clients
func (m *Manager[T]) broadcastToAll(message Message[T]) {
	m.clientsMu.RLock()
	// Create a copy to avoid holding the lock while sending
	clients := make(map[ClientID]*Client[T])
	for id, client := range m.clients {
		clients[id] = client
	}
	m.clientsMu.RUnlock()

	for clientID, client := range clients {
		select {
		case client.Channel <- message:
		case <-client.Context.Done():
			// Client disconnected, remove it
			m.RemoveClient(clientID)
		default:
			// Channel is full, skip this client
			if m.eventHandler.OnError != nil {
				m.eventHandler.OnError(clientID, fmt.Errorf("client channel is full"))
			}
		}
	}
}

// removeClientFromAllRooms removes a client from all rooms
func (m *Manager[T]) removeClientFromAllRooms(clientID ClientID) {
	m.roomsMu.Lock()
	defer m.roomsMu.Unlock()

	for room, clients := range m.rooms {
		delete(clients, clientID)
		if len(clients) == 0 {
			delete(m.rooms, room)
		}
	}
}
