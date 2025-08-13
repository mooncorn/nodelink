package agent

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles SSE streaming for agent status updates
type SSEHandler struct {
	repo       *Repository
	sseManager *sse.Manager[StatusChangeEvent]
}

// NewSSEHandler creates a new SSE handler for agent status updates
func NewSSEHandler(repo *Repository) *SSEHandler {
	// Create SSE manager for agent status events
	config := sse.ManagerConfig{
		BufferSize:     100,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	handler := &SSEHandler{
		repo: repo,
	}

	// Create custom event handler
	eventHandler := sse.EventHandler[StatusChangeEvent]{
		OnConnect: func(client *sse.Client[StatusChangeEvent]) {
			log.Printf("Agent SSE Client connected: %s", client.ID)
		},
		OnDisconnect: func(client *sse.Client[StatusChangeEvent]) {
			log.Printf("Agent SSE Client disconnected: %s", client.ID)
		},
		OnMessage: func(message sse.Message[StatusChangeEvent]) {
			// Default message handling
		},
		OnError: func(clientID sse.ClientID, err error) {
			log.Printf("Agent SSE Error for client %s: %v", clientID, err)
		},
	}

	handler.sseManager = sse.NewManager(config, eventHandler)
	handler.sseManager.Start()

	// Register as a status change listener with the repository
	repo.AddListener(handler)

	return handler
}

// RegisterRoutes registers SSE routes with the given router
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	// SSE endpoint for agent status updates
	router.GET("/agents/events", h.HandleSSE)

	// SSE endpoint for specific agent status updates
	router.GET("/agents/:agentId/events", h.HandleAgentSSE)
}

// HandleSSE handles SSE connections for all agent status updates
func (h *SSEHandler) HandleSSE(c *gin.Context) {
	// Add the client to the SSE manager
	client := h.sseManager.AddClient("")

	// Join the general agent status room
	if err := h.sseManager.JoinRoom(client.ID, "agent_status"); err != nil {
		log.Printf("Failed to join agent_status room: %v", err)
		c.JSON(500, gin.H{"error": "Failed to setup SSE connection"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Start the SSE stream
	h.handleSSEStream(c, client)
}

// HandleAgentSSE handles SSE connections for specific agent status updates
func (h *SSEHandler) HandleAgentSSE(c *gin.Context) {
	agentID := c.Param("agentId")
	if agentID == "" {
		c.JSON(400, gin.H{"error": "agent_id is required"})
		return
	}

	// Add the client to the SSE manager
	client := h.sseManager.AddClient("")

	// Create agent-specific room
	roomName := fmt.Sprintf("status_%s", agentID)

	// Join the agent-specific room
	if err := h.sseManager.JoinRoom(client.ID, roomName); err != nil {
		log.Printf("Failed to join room %s: %v", roomName, err)
		c.JSON(500, gin.H{"error": "Failed to setup SSE connection"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Start the SSE stream
	h.handleSSEStream(c, client)
}

// handleSSEStream handles the actual SSE streaming
func (h *SSEHandler) handleSSEStream(c *gin.Context, client *sse.Client[StatusChangeEvent]) {
	// Handle client cleanup when the connection closes
	defer func() {
		h.sseManager.RemoveClient(client.ID)
	}()

	// Send initial connection event
	c.SSEvent("connection", "connected")
	c.Writer.Flush()

	// Stream events from the client channel
	for {
		select {
		case message := <-client.Channel:
			// Send the status change event
			eventType := fmt.Sprintf("status_%s", message.Data.AgentID)
			c.SSEvent(eventType, message.Data)
			c.Writer.Flush()
		case <-client.Context.Done():
			// Client disconnected
			return
		case <-c.Request.Context().Done():
			// Request cancelled
			return
		}
	}
}

// OnStatusChange implements StatusChangeListener interface
// This method gets called whenever an agent's status changes
func (h *SSEHandler) OnStatusChange(event StatusChangeEvent) {
	// Send to general agent status room
	if err := h.sseManager.SendToRoom("agent_status", event, "agent_status_change"); err != nil {
		log.Printf("Failed to send status change to agent_status room: %v", err)
	}

	// Send to agent-specific room
	roomName := fmt.Sprintf("status_%s", event.AgentID)
	if err := h.sseManager.SendToRoom(roomName, event, "status_change"); err != nil {
		log.Printf("Failed to send status change to room %s: %v", roomName, err)
	}
}

// Stop stops the SSE handler
func (h *SSEHandler) Stop() {
	if h.sseManager != nil {
		h.sseManager.Stop()
	}
}
