package command

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/agent"
)

// HTTPHandler handles HTTP requests for command execution
type HTTPHandler struct {
	agentManager *agent.Manager
}

// NewHTTPHandler creates a new HTTP handler for commands
func NewHTTPHandler(agentManager *agent.Manager) *HTTPHandler {
	return &HTTPHandler{
		agentManager: agentManager,
	}
}

// ExecuteCommandRequest represents an HTTP request for command execution
type ExecuteCommandRequest struct {
	AgentID          string            `json:"agent_id" binding:"required"`
	Command          string            `json:"command" binding:"required"`
	Environment      map[string]string `json:"environment,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	TimeoutSeconds   uint32            `json:"timeout_seconds,omitempty"`
}

// ExecuteCommandResponse represents an HTTP response for command execution
type ExecuteCommandResponse struct {
	RequestID       string `json:"request_id"`
	ExitCode        int32  `json:"exit_code"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	TimedOut        bool   `json:"timed_out"`
	ExecutionTimeMs int64  `json:"execution_time_ms"`
}

// ExecuteCommand handles HTTP POST requests for command execution
func (h *HTTPHandler) ExecuteCommand(c *gin.Context) {
	var req ExecuteCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if agent is connected
	if !h.agentManager.IsAgentConnected(req.AgentID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not connected"})
		return
	}

	// Create gRPC request (for future implementation)
	requestID := generateRequestID()
	_ = &pb.CommandRequest{
		Metadata: &pb.RequestMetadata{
			AgentId:   req.AgentID,
			RequestId: requestID,
			Timestamp: time.Now().Unix(),
		},
		Command:          req.Command,
		Environment:      req.Environment,
		WorkingDirectory: req.WorkingDirectory,
		TimeoutSeconds:   req.TimeoutSeconds,
	}

	// Set timeout
	ctx := c.Request.Context()
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	_ = ctx // Suppress unused variable warning	// Execute command
	// For now, return an error as command execution needs to be implemented
	// The HTTP handler should integrate with the command server or handle commands directly
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Command execution via HTTP not yet implemented",
		"note":  "Commands should be executed through the gRPC CommandServer or implement direct agent communication",
	})
}

// ListAgents handles HTTP GET requests for listing connected agents
func (h *HTTPHandler) ListAgents(c *gin.Context) {
	agents := h.agentManager.GetConnectedAgents()
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// GetAgentStatus handles HTTP GET requests for agent status
func (h *HTTPHandler) GetAgentStatus(c *gin.Context) {
	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	conn, exists := h.agentManager.GetAgentConnection(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	status := gin.H{
		"agent_id":     conn.AgentID,
		"status":       conn.Status.String(),
		"last_seen":    conn.LastSeen,
		"capabilities": conn.Capabilities,
		"metadata":     conn.Metadata,
	}

	c.JSON(http.StatusOK, status)
}

// RegisterRoutes registers HTTP routes for command operations
func (h *HTTPHandler) RegisterRoutes(router *gin.RouterGroup) {
	commands := router.Group("/commands")
	{
		commands.POST("/execute", h.ExecuteCommand)
	}

	agents := router.Group("/agents")
	{
		agents.GET("", h.ListAgents)
		agents.GET("/:agent_id/status", h.GetAgentStatus)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production you might want to use UUID
	return time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
