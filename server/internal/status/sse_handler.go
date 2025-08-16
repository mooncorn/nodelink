package status

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles SSE streaming for agent status updates
type SSEHandler struct {
	manager           *Manager
	sseManager        common.SSEManager
	connectionHandler *sse.ConnectionHandler
	formatter         *StatusMessageFormatter
	rooms             *StatusRooms
}

// NewSSEHandler creates a new SSE handler for agent status updates
func NewSSEHandler(manager *Manager, sseManager common.SSEManager) *SSEHandler {
	handler := &SSEHandler{
		manager:           manager,
		sseManager:        sseManager,
		connectionHandler: sse.NewConnectionHandler(sseManager),
		formatter:         NewStatusMessageFormatter(),
		rooms:             NewStatusRooms(),
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
	config := sse.ConnectionConfig{
		ClientIDPrefix:     "client",
		ClientIDComponents: []string{c.Request.RemoteAddr},
		Rooms:              []string{h.rooms.GetAllAgentsRoom()},
		InitialMessages:    []string{h.formatter.FormatConnectionMessage()},
		MessageFormatter:   h.formatter.FormatMessage,
	}

	if err := h.connectionHandler.HandleConnection(c, config); err != nil {
		log.Printf("Error handling SSE connection: %v", err)
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

	// Prepare initial messages
	initialMessages := []string{h.formatter.FormatAgentConnectionMessage(agentID)}

	// Add current agent status if available
	if agent, exists := h.manager.GetAgent(agentID); exists {
		initialMessages = append(initialMessages, h.formatter.FormatCurrentStatusMessage(agent))
	}

	config := sse.ConnectionConfig{
		ClientIDPrefix:     "client",
		ClientIDComponents: []string{agentID, c.Request.RemoteAddr},
		Rooms:              []string{h.rooms.GetAgentSpecificRoom(agentID)},
		InitialMessages:    initialMessages,
		MessageFormatter:   h.formatter.FormatMessage,
	}

	if err := h.connectionHandler.HandleConnection(c, config); err != nil {
		log.Printf("Error handling SSE connection: %v", err)
	}
}

// OnStatusChange implements StatusChangeListener interface
func (h *SSEHandler) OnStatusChange(event common.StatusChangeEvent) {
	// Broadcast status change event to all connected SSE clients in the general agents room
	if err := h.sseManager.SendToRoom(h.rooms.GetAllAgentsRoom(), event, "agent_status_change"); err != nil {
		log.Printf("Failed to broadcast agent status change to agents room: %v", err)
	}

	// Also send to agent-specific room if someone is listening to specific agent
	agentRoom := h.rooms.GetAgentSpecificRoom(event.AgentID)
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
