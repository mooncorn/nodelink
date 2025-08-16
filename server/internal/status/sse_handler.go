package status

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// SSEHandler handles SSE streaming for agent status updates
type SSEHandler struct {
	manager    *Manager
	sseManager common.SSEManager
}

// NewSSEHandler creates a new SSE handler for agent status updates
func NewSSEHandler(manager *Manager, sseManager common.SSEManager) *SSEHandler {
	handler := &SSEHandler{
		manager:    manager,
		sseManager: sseManager,
	}

	// Register as a status change listener with the manager
	manager.AddListener(handler)

	return handler
}

// RegisterRoutes registers SSE routes for agent status updates
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/agents/events", h.handleAllAgentStatusEvents)
	router.GET("/agents/:agentId/events", h.handleSpecificAgentStatusEvents)
}

// handleAllAgentStatusEvents handles SSE connections for all agent status events
func (h *SSEHandler) handleAllAgentStatusEvents(c *gin.Context) {
	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate a unique client ID
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("client_%s_%d", c.Request.RemoteAddr, time.Now().UnixNano())
	}

	// Add client to SSE manager
	client := h.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return
	}

	// Join the default room for all agent events
	room := "agents"
	if err := h.sseManager.JoinRoom(clientID, room); err != nil {
		log.Printf("Error joining room %s: %v", room, err)
	}

	// Handle the SSE connection
	defer h.sseManager.RemoveClient(clientID)

	// Send initial connection message
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", formatConnectionMessage()); err != nil {
		log.Printf("Error writing initial SSE message: %v", err)
		return
	}
	c.Writer.Flush()

	// Keep connection alive and send messages
	for {
		select {
		case msg := <-client.GetChannel():
			// Format and send SSE message
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", formatMessage(msg)); err != nil {
				log.Printf("Error writing SSE message: %v", err)
				return
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		case <-client.GetContext().Done():
			return
		}
	}
}

// handleSpecificAgentStatusEvents handles SSE connections for specific agent status events
func (h *SSEHandler) handleSpecificAgentStatusEvents(c *gin.Context) {
	// Get agent ID from URL parameter
	agentID := c.Param("agentId")
	if agentID == "" {
		c.JSON(400, gin.H{"error": "agent_id is required"})
		return
	}

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate a unique client ID
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("client_%s_%s_%d", agentID, c.Request.RemoteAddr, time.Now().UnixNano())
	}

	// Add client to SSE manager
	client := h.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return
	}

	// Join the agent-specific room
	agentRoom := fmt.Sprintf("agent_%s", agentID)
	if err := h.sseManager.JoinRoom(clientID, agentRoom); err != nil {
		log.Printf("Error joining room %s: %v", agentRoom, err)
	}

	// Handle the SSE connection
	defer h.sseManager.RemoveClient(clientID)

	// Send initial connection message with agent info
	connectionMsg := formatAgentConnectionMessage(agentID)
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", connectionMsg); err != nil {
		log.Printf("Error writing initial SSE message: %v", err)
		return
	}
	c.Writer.Flush()

	// Send current agent status if available
	if agent, exists := h.manager.GetAgent(agentID); exists {
		currentStatusMsg := formatCurrentStatusMessage(agent)
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", currentStatusMsg); err != nil {
			log.Printf("Error writing current status message: %v", err)
			return
		}
		c.Writer.Flush()
	}

	// Keep connection alive and send messages
	for {
		select {
		case msg := <-client.GetChannel():
			// Format and send SSE message
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", formatMessage(msg)); err != nil {
				log.Printf("Error writing SSE message: %v", err)
				return
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		case <-client.GetContext().Done():
			return
		}
	}
}

// formatMessage formats a message for SSE transmission
func formatMessage(msg common.SSEMessage) string {
	data, err := json.Marshal(map[string]interface{}{
		"event": msg.EventType,
		"data":  msg.Data,
		"room":  msg.Room,
	})
	if err != nil {
		log.Printf("Error marshaling SSE message: %v", err)
		return fmt.Sprintf(`{"event":"%s","error":"failed to marshal data"}`, msg.EventType)
	}
	return string(data)
}

// formatConnectionMessage formats the initial connection message
func formatConnectionMessage() string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "connection",
		"data":  map[string]string{"status": "connected"},
	})
	return string(data)
}

// formatAgentConnectionMessage formats the initial connection message for a specific agent
func formatAgentConnectionMessage(agentID string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "agent_connection",
		"data": map[string]string{
			"status":   "connected",
			"agent_id": agentID,
		},
	})
	return string(data)
}

// formatCurrentStatusMessage formats the current status message for an agent
func formatCurrentStatusMessage(agent *common.AgentInfo) string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "current_status",
		"data":  agent,
	})
	return string(data)
}

// OnStatusChange implements StatusChangeListener interface
func (h *SSEHandler) OnStatusChange(event common.StatusChangeEvent) {
	// Broadcast status change event to all connected SSE clients in the general agents room
	if err := h.sseManager.SendToRoom("agents", event, "agent_status_change"); err != nil {
		log.Printf("Failed to broadcast agent status change to agents room: %v", err)
	}

	// Also send to agent-specific room if someone is listening to specific agent
	agentRoom := fmt.Sprintf("agent_%s", event.AgentID)
	if err := h.sseManager.SendToRoom(agentRoom, event, "status_change"); err != nil {
		log.Printf("Failed to broadcast to agent room %s: %v", agentRoom, err)
	}
}

// Stop gracefully stops the SSE handler
func (h *SSEHandler) Stop() {
	if h.sseManager != nil {
		h.sseManager.Stop()
	}
}
