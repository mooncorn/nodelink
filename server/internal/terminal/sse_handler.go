package terminal

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles SSE streaming for terminal output
type SSEHandler struct {
	terminalHandler   *Handler
	sessionManager    common.TerminalSessionManager
	sseManager        common.SSEManager
	connectionHandler *sse.ConnectionHandler
	formatter         *TerminalMessageFormatter
	rooms             *TerminalRooms
}

// NewSSEHandler creates a new SSE handler for terminal streaming
func NewSSEHandler(terminalHandler *Handler, sseManager common.SSEManager) *SSEHandler {
	return &SSEHandler{
		terminalHandler:   terminalHandler,
		sessionManager:    terminalHandler.sessionManager,
		sseManager:        sseManager,
		connectionHandler: sse.NewConnectionHandler(sseManager),
		formatter:         NewTerminalMessageFormatter(),
		rooms:             NewTerminalRooms(),
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

	// Setup SSE connection configuration
	config := sse.ConnectionConfig{
		ClientIDPrefix:     "terminal",
		ClientIDComponents: []string{userID, sessionID},
		Rooms:              []string{h.rooms.GetTerminalSessionRoom(sessionID)},
		InitialMessages:    []string{h.formatter.FormatTerminalConnectionMessage(sessionID)},
		MessageFormatter:   h.customTerminalFormatter,
	}

	// Update last activity before starting the connection
	h.sessionManager.UpdateLastActivity(sessionID)

	if err := h.connectionHandler.HandleConnection(c, config); err != nil {
		log.Printf("Error handling terminal SSE connection: %v", err)
	}
}

// customTerminalFormatter provides terminal-specific message formatting
func (h *SSEHandler) customTerminalFormatter(msg common.SSEMessage) string {
	// For terminal messages, we use a simpler format compatible with c.SSEvent
	return h.formatter.FormatMessage(msg)
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
