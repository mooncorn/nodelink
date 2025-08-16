package terminal

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// HTTPHandler handles HTTP endpoints for terminal operations
type HTTPHandler struct {
	terminalHandler *Handler
}

// NewHTTPHandler creates a new HTTP handler for terminal operations
func NewHTTPHandler(terminalHandler *Handler) *HTTPHandler {
	return &HTTPHandler{
		terminalHandler: terminalHandler,
	}
}

// RegisterRoutes registers terminal HTTP routes
func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	terminals := router.Group("/terminals")
	{
		terminals.POST("", h.createTerminalSession)
		terminals.GET("", h.getUserTerminalSessions)
		terminals.POST("/:sessionId/command", h.executeCommand)
		terminals.DELETE("/:sessionId", h.closeTerminalSession)
	}
}

// CreateSessionRequest represents the request to create a terminal session
type CreateSessionRequest struct {
	AgentID    string            `json:"agent_id" binding:"required"`
	Shell      string            `json:"shell,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// CreateSessionResponse represents the response for terminal session creation
type CreateSessionResponse struct {
	SessionID  string `json:"session_id"`
	AgentID    string `json:"agent_id"`
	Shell      string `json:"shell"`
	WorkingDir string `json:"working_dir"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

// ExecuteCommandRequest represents the request to execute a command
type ExecuteCommandRequest struct {
	Command string `json:"command" binding:"required"`
}

// ExecuteCommandResponse represents the response for command execution
type ExecuteCommandResponse struct {
	CommandID string `json:"command_id"`
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

// createTerminalSession handles POST /terminals
func (h *HTTPHandler) createTerminalSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	// Get user ID from context (should be set by auth middleware)
	userID := h.getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Create terminal session
	session, err := h.terminalHandler.CreateTerminalSession(
		c.Request.Context(),
		userID,
		req.AgentID,
		req.Shell,
		req.WorkingDir,
		req.Env,
	)

	if err != nil {
		switch err {
		case common.ErrAgentNotConnected:
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent is not connected"})
		case common.ErrMaxTerminalSessionsReached:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Maximum terminal sessions reached"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Convert to response
	response := CreateSessionResponse{
		SessionID:  session.SessionID,
		AgentID:    session.AgentID,
		Shell:      session.Shell,
		WorkingDir: session.WorkingDir,
		Status:     string(session.Status),
		CreatedAt:  session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusCreated, response)
}

// getUserTerminalSessions handles GET /terminals
func (h *HTTPHandler) getUserTerminalSessions(c *gin.Context) {
	// Get user ID from context
	userID := h.getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Get user sessions
	sessions := h.terminalHandler.GetUserSessions(userID)

	// Convert to response format
	response := make([]CreateSessionResponse, len(sessions))
	for i, session := range sessions {
		response[i] = CreateSessionResponse{
			SessionID:  session.SessionID,
			AgentID:    session.AgentID,
			Shell:      session.Shell,
			WorkingDir: session.WorkingDir,
			Status:     string(session.Status),
			CreatedAt:  session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"sessions": response})
}

// executeCommand handles POST /terminals/:sessionId/command
func (h *HTTPHandler) executeCommand(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req ExecuteCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	// Get user ID from context
	userID := h.getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Execute command
	commandID, err := h.terminalHandler.ExecuteCommand(
		c.Request.Context(),
		sessionID,
		userID,
		req.Command,
	)

	if err != nil {
		switch err {
		case common.ErrTerminalSessionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Terminal session not found"})
		case common.ErrUnauthorizedTerminalAccess:
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to terminal session"})
		case common.ErrTerminalSessionClosed:
			c.JSON(http.StatusGone, gin.H{"error": "Terminal session is closed"})
		case common.ErrAgentNotConnected:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent is not connected"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Return command ID for tracking
	response := ExecuteCommandResponse{
		CommandID: commandID,
		SessionID: sessionID,
		Status:    "executing",
	}

	c.JSON(http.StatusOK, response)
}

// closeTerminalSession handles DELETE /terminals/:sessionId
func (h *HTTPHandler) closeTerminalSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Get user ID from context
	userID := h.getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Close terminal session
	err := h.terminalHandler.CloseTerminalSession(
		c.Request.Context(),
		sessionID,
		userID,
	)

	if err != nil {
		switch err {
		case common.ErrTerminalSessionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Terminal session not found"})
		case common.ErrUnauthorizedTerminalAccess:
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to terminal session"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terminal session closed successfully"})
}

// getUserIDFromContext extracts user ID from the request context
// This should be set by authentication middleware
func (h *HTTPHandler) getUserIDFromContext(c *gin.Context) string {
	// For now, we'll use a simple approach with a header
	// In a real implementation, this would come from JWT token or session
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		// Fallback to query parameter for testing
		userID = c.Query("user_id")
	}

	// Clean and validate user ID
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}

	return userID
}
