package command

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ExecuteRequest represents the HTTP request for command execution
type ExecuteRequest struct {
	AgentID    string            `json:"agent_id" binding:"required"`
	Command    string            `json:"command" binding:"required"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Timeout    int               `json:"timeout_seconds,omitempty"`
}

// ExecuteResponse represents the HTTP response for command execution
type ExecuteResponse struct {
	RequestID string `json:"request_id"`
	ExitCode  int32  `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Error     string `json:"error,omitempty"`
	Timeout   bool   `json:"timeout"`
}

// HTTPHandler handles HTTP requests for command execution
type HTTPHandler struct {
	commandHandler *Handler
}

// NewHTTPHandler creates a new HTTP handler for commands
func NewHTTPHandler(commandHandler *Handler) *HTTPHandler {
	return &HTTPHandler{
		commandHandler: commandHandler,
	}
}

// RegisterRoutes registers command-related routes
func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/commands", h.executeCommand)
	router.GET("/commands/pending", h.getPendingRequests)
}

// executeCommand handles POST /commands
func (h *HTTPHandler) executeCommand(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	// Convert timeout to duration
	timeout := time.Duration(req.Timeout) * time.Second

	// Execute command
	response, err := h.commandHandler.ExecuteCommand(
		c.Request.Context(),
		req.AgentID,
		req.Command,
		req.Args,
		req.Env,
		req.WorkingDir,
		timeout,
	)

	if err != nil {
		switch err {
		case ErrAgentNotConnected:
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent is not connected"})
		case ErrRequestTimeout:
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "Command execution timed out"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Convert protobuf response to HTTP response
	httpResponse := ExecuteResponse{
		RequestID: response.RequestId,
		ExitCode:  response.ExitCode,
		Stdout:    response.Stdout,
		Stderr:    response.Stderr,
		Error:     response.Error,
		Timeout:   response.Timeout,
	}

	c.JSON(http.StatusOK, httpResponse)
}

// getPendingRequests handles GET /commands/pending
func (h *HTTPHandler) getPendingRequests(c *gin.Context) {
	pending := h.commandHandler.GetPendingRequests()

	// Convert to response format
	type PendingRequest struct {
		ID         string            `json:"id"`
		AgentID    string            `json:"agent_id"`
		Command    string            `json:"command"`
		Args       []string          `json:"args"`
		Env        map[string]string `json:"env"`
		WorkingDir string            `json:"working_dir"`
		Timeout    string            `json:"timeout"`
		CreatedAt  time.Time         `json:"created_at"`
	}

	var requests []PendingRequest
	for _, req := range pending {
		requests = append(requests, PendingRequest{
			ID:         req.ID,
			AgentID:    req.AgentID,
			Command:    req.Command,
			Args:       req.Args,
			Env:        req.Env,
			WorkingDir: req.WorkingDir,
			Timeout:    req.Timeout.String(),
			CreatedAt:  req.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"pending_requests": requests,
		"count":            len(requests),
	})
}
