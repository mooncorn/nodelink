package terminal

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// SSEHandler handles SSE streaming for terminal output
type SSEHandler struct {
	terminalHandler *Handler
	sessionManager  common.TerminalSessionManager
	sseManager      common.SSEManager
}

// NewSSEHandler creates a new SSE handler for terminal streaming
func NewSSEHandler(terminalHandler *Handler, sseManager common.SSEManager) *SSEHandler {
	return &SSEHandler{
		terminalHandler: terminalHandler,
		sessionManager:  terminalHandler.sessionManager,
		sseManager:      sseManager,
	}
}

// RegisterRoutes registers SSE routes for terminal streaming
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/terminals/:sessionId/stream", h.handleTerminalStream)
}

// handleTerminalStream handles SSE connections for terminal output
func (h *SSEHandler) handleTerminalStream(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(400, gin.H{"error": "Session ID is required"})
		return
	}

	// Get user ID from context
	userID := h.getUserIDFromContext(c)
	if userID == "" {
		c.JSON(401, gin.H{"error": "User authentication required"})
		return
	}

	// Validate session access
	if err := h.sessionManager.ValidateSessionAccess(sessionID, userID); err != nil {
		switch err {
		case common.ErrTerminalSessionNotFound:
			c.JSON(404, gin.H{"error": "Terminal session not found"})
		case common.ErrUnauthorizedTerminalAccess:
			c.JSON(403, gin.H{"error": "Unauthorized access to terminal session"})
		case common.ErrTerminalSessionClosed:
			c.JSON(410, gin.H{"error": "Terminal session is closed"})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate a unique client ID for this connection
	clientID := fmt.Sprintf("terminal_%s_%s", userID, sessionID)

	// Add client to SSE manager
	client := h.sseManager.AddClient(clientID)
	defer h.sseManager.RemoveClient(clientID)

	// Join the terminal session room
	roomID := GetTerminalSessionRoom(sessionID)
	if err := h.sseManager.JoinRoom(clientID, roomID); err != nil {
		log.Printf("Failed to join terminal room %s: %v", roomID, err)
		c.JSON(500, gin.H{"error": "Failed to join terminal stream"})
		return
	}

	// Send initial connection message
	h.sseManager.SendToRoom(roomID, map[string]interface{}{
		"session_id": sessionID,
		"message":    "Connected to terminal stream",
	}, "terminal_connected")

	// Update last activity
	h.sessionManager.UpdateLastActivity(sessionID)

	// Keep connection alive and stream messages
	for {
		select {
		case msg := <-client.GetChannel():
			// Format SSE message
			if msg.EventType != "" {
				c.SSEvent(msg.EventType, msg.Data)
			} else {
				c.SSEvent("message", msg.Data)
			}
			c.Writer.Flush()

		case <-client.GetContext().Done():
			// Client disconnected
			log.Printf("Client %s disconnected from terminal stream %s", clientID, sessionID)
			return
		}
	}
}

// getUserIDFromContext extracts user ID from the request context
func (h *SSEHandler) getUserIDFromContext(c *gin.Context) string {
	// For now, we'll use a simple approach with a header
	// In a real implementation, this would come from JWT token or session
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		// Fallback to query parameter for testing
		userID = c.Query("user_id")
	}

	return userID
}
