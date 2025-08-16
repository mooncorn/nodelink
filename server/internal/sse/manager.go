package sse

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mooncorn/nodelink/server/internal/common"
)

// Client represents a connected SSE client
type Client struct {
	ID      string
	Channel chan common.SSEMessage
	Context context.Context
	Cancel  context.CancelFunc
	Rooms   []string
	mu      sync.RWMutex
}

// GetChannel implements common.SSEClient interface
func (c *Client) GetChannel() <-chan common.SSEMessage {
	return c.Channel
}

// GetContext implements common.SSEClient interface
func (c *Client) GetContext() context.Context {
	return c.Context
}

// Manager manages SSE connections and message broadcasting
type Manager struct {
	clients   map[string]*Client
	rooms     map[string]map[string]*Client
	clientsMu sync.RWMutex
	roomsMu   sync.RWMutex
	running   bool
	stopCh    chan struct{}
}

// NewManager creates a new SSE manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		rooms:   make(map[string]map[string]*Client),
		stopCh:  make(chan struct{}),
	}
}

// Start starts the SSE manager
func (m *Manager) Start() {
	m.running = true
}

// Stop stops the SSE manager and closes all connections
func (m *Manager) Stop() {
	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false

	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	for _, client := range m.clients {
		client.Cancel()
		close(client.Channel)
	}

	m.clients = make(map[string]*Client)
	m.rooms = make(map[string]map[string]*Client)
}

// AddClient adds a new SSE client
func (m *Manager) AddClient(clientID string) common.SSEClient {
	if clientID == "" {
		clientID = fmt.Sprintf("client_%d", time.Now().UnixNano())
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		ID:      clientID,
		Channel: make(chan common.SSEMessage, 100),
		Context: ctx,
		Cancel:  cancel,
		Rooms:   make([]string, 0),
	}

	m.clientsMu.Lock()
	m.clients[clientID] = client
	m.clientsMu.Unlock()

	log.Printf("SSE Client connected: %s", clientID)
	return client
}

// RemoveClient removes an SSE client
func (m *Manager) RemoveClient(clientID string) {
	m.clientsMu.Lock()
	client, exists := m.clients[clientID]
	if !exists {
		m.clientsMu.Unlock()
		return
	}

	// Remove from all rooms
	client.mu.RLock()
	rooms := make([]string, len(client.Rooms))
	copy(rooms, client.Rooms)
	client.mu.RUnlock()

	delete(m.clients, clientID)
	m.clientsMu.Unlock()

	// Remove from rooms
	m.roomsMu.Lock()
	for _, room := range rooms {
		if roomClients, exists := m.rooms[room]; exists {
			delete(roomClients, clientID)
			if len(roomClients) == 0 {
				delete(m.rooms, room)
			}
		}
	}
	m.roomsMu.Unlock()

	// Cancel and close
	client.Cancel()
	close(client.Channel)

	log.Printf("SSE Client disconnected: %s", clientID)
}

// JoinRoom adds a client to a room
func (m *Manager) JoinRoom(clientID, room string) error {
	m.clientsMu.RLock()
	client, exists := m.clients[clientID]
	m.clientsMu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	client.mu.Lock()
	client.Rooms = append(client.Rooms, room)
	client.mu.Unlock()

	m.roomsMu.Lock()
	if m.rooms[room] == nil {
		m.rooms[room] = make(map[string]*Client)
	}
	m.rooms[room][clientID] = client
	m.roomsMu.Unlock()

	return nil
}

// Broadcast sends a message to all clients
func (m *Manager) Broadcast(data any, eventType string) error {
	message := common.SSEMessage{
		EventType: eventType,
		Data:      data,
	}

	m.clientsMu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.clientsMu.RUnlock()

	for _, client := range clients {
		select {
		case client.Channel <- message:
		case <-client.Context.Done():
			// Client disconnected, remove it
			go m.RemoveClient(client.ID)
		default:
			// Channel is full, skip this client
			log.Printf("SSE channel full for client %s", client.ID)
		}
	}

	return nil
}

// SendToRoom sends a message to all clients in a room
func (m *Manager) SendToRoom(room string, data any, eventType string) error {
	message := common.SSEMessage{
		EventType: eventType,
		Data:      data,
		Room:      room,
	}

	m.roomsMu.RLock()
	roomClients, exists := m.rooms[room]
	if !exists {
		m.roomsMu.RUnlock()
		return nil // Room doesn't exist, no clients to send to
	}

	clients := make([]*Client, 0, len(roomClients))
	for _, client := range roomClients {
		clients = append(clients, client)
	}
	m.roomsMu.RUnlock()

	for _, client := range clients {
		select {
		case client.Channel <- message:
		case <-client.Context.Done():
			// Client disconnected, remove it
			go m.RemoveClient(client.ID)
		default:
			// Channel is full, skip this client
			log.Printf("SSE channel full for client %s", client.ID)
		}
	}

	return nil
}
