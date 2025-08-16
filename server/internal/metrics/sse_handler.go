package metrics

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles metrics SSE endpoints
type SSEHandler struct {
	handler           *Handler
	streamingManager  *StreamingManager
	sseManager        common.SSEManager
	formatter         *MetricsMessageFormatter
	rooms             *MetricsRooms
	connectionHandler *sse.ConnectionHandler
}

// NewSSEHandler creates a new SSE handler for metrics streaming
func NewSSEHandler(handler *Handler, streamingManager *StreamingManager, sseManager common.SSEManager) *SSEHandler {
	return &SSEHandler{
		handler:           handler,
		streamingManager:  streamingManager,
		sseManager:        sseManager,
		formatter:         NewMetricsMessageFormatter(),
		rooms:             NewMetricsRooms(),
		connectionHandler: sse.NewConnectionHandler(sseManager),
	}
}

// RegisterRoutes registers SSE routes for metrics streaming
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/metrics/:agentID/stream", h.handleMetricsStream)
}

// handleMetricsStream handles SSE connections for metrics streaming
func (h *SSEHandler) handleMetricsStream(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(400, gin.H{"error": "Agent ID is required"})
		return
	}

	// Check if agent exists and is online
	if !h.handler.statusManager.IsAgentOnline(agentID) {
		c.JSON(404, gin.H{"error": "Agent not found or offline"})
		return
	}

	// Get user ID from context (for now, we'll use a default)
	userID := h.getUserIDFromContext(c)
	if userID == "" {
		userID = "default-user" // For now, until we implement proper auth
	}

	// Validate access (agent must be accessible to user)
	if err := h.validateAgentAccess(agentID, userID); err != nil {
		c.JSON(403, gin.H{"error": "Access denied to agent metrics"})
		return
	}

	// Prepare initial messages with cached data
	initialMessages := h.getInitialMessages(agentID)

	// Setup connection configuration
	config := sse.ConnectionConfig{
		ClientIDPrefix:     "metrics-client",
		ClientIDComponents: []string{agentID},
		Rooms:              []string{h.rooms.GetMetricsRoomName(agentID)},
		InitialMessages:    initialMessages,
		MessageFormatter:   h.formatSSEMessage,
	}

	// Handle the SSE connection using the reusable connection handler
	if err := h.connectionHandler.HandleConnection(c, config); err != nil {
		log.Printf("Error handling metrics SSE connection for agent %s: %v", agentID, err)
	}
}

// getInitialMessages prepares initial messages with cached data
func (h *SSEHandler) getInitialMessages(agentID string) []string {
	var messages []string

	// Add cached metrics if available
	if metrics, exists := h.streamingManager.GetCachedMetrics(agentID); exists {
		if data, err := json.Marshal(h.formatter.FormatMetricsMessage(agentID, metrics)); err == nil {
			messages = append(messages, string(data))
		}
	}

	// Add cached system info if available
	if systemInfo, exists := h.streamingManager.GetCachedSystemInfo(agentID); exists {
		if data, err := json.Marshal(h.formatter.FormatSystemInfoMessage(agentID, systemInfo)); err == nil {
			messages = append(messages, string(data))
		}
	}

	return messages
}

// formatSSEMessage formats SSE messages using the metrics formatter
func (h *SSEHandler) formatSSEMessage(msg common.SSEMessage) string {
	// For metrics SSE messages, we can pass them through as-is since they're already formatted
	if data, err := json.Marshal(msg.Data); err == nil {
		return string(data)
	}
	return "{}"
}

// sendInitialDataViaSSE sends cached data directly via SSE to the client
func (h *SSEHandler) sendInitialDataViaSSE(c *gin.Context, agentID string) {
	// Send cached metrics if available
	if metrics, exists := h.streamingManager.GetCachedMetrics(agentID); exists {
		formattedMsg := h.formatter.FormatMetricsMessage(agentID, metrics)
		c.SSEvent("metrics", formattedMsg)
		c.Writer.Flush()
	}

	// Send cached system info if available
	if systemInfo, exists := h.streamingManager.GetCachedSystemInfo(agentID); exists {
		formattedMsg := h.formatter.FormatSystemInfoMessage(agentID, systemInfo)
		c.SSEvent("system_info", formattedMsg)
		c.Writer.Flush()
	}
}

// getUserIDFromContext extracts user ID from gin context
func (h *SSEHandler) getUserIDFromContext(c *gin.Context) string {
	// For now, return a default user ID
	// In a real implementation, this would extract from JWT token or session
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return "default-user"
}

// validateAgentAccess validates if a user can access an agent's metrics
func (h *SSEHandler) validateAgentAccess(agentID, userID string) error {
	// For now, allow all access
	// In a real implementation, this would check permissions/ACLs
	return nil
}
