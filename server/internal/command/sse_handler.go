package command

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/agent"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles Server-Sent Events for command streaming
type SSEHandler struct {
	agentManager *agent.Manager
	sseManager   *sse.Manager[*pb.CommandStreamResponse]
	processor    *CommandProcessor
}

// NewSSEHandler creates a new SSE handler for command streaming
func NewSSEHandler(agentManager *agent.Manager, sseManager *sse.Manager[*pb.CommandStreamResponse], processor *CommandProcessor) *SSEHandler {
	return &SSEHandler{
		agentManager: agentManager,
		sseManager:   sseManager,
		processor:    processor,
	}
}

// StreamCommandRequest represents a request to start command streaming
type StreamCommandRequest struct {
	AgentID          string            `json:"agent_id" binding:"required"`
	Command          string            `json:"command" binding:"required"`
	Environment      map[string]string `json:"environment,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	TimeoutSeconds   uint32            `json:"timeout_seconds,omitempty"`
}

// StreamCommand handles SSE connections for command streaming
func (h *SSEHandler) StreamCommand(c *gin.Context) {
	var req StreamCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if agent is connected
	if !h.agentManager.IsAgentConnected(req.AgentID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not connected"})
		return
	}

	// Create unique room for this command stream
	requestID := generateRequestID()
	room := "command_stream_" + requestID

	// Enable buffering for this room
	h.sseManager.EnableRoomBuffering(room, 100)
	defer h.sseManager.DisableRoomBuffering(room)

	// Create gRPC request (for future implementation)
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
	_ = ctx // Suppress unused variable warning

	// Start streaming command execution
	// For now, this is a placeholder - needs integration with command execution
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Streaming command execution via SSE not yet implemented",
		"note":  "This requires implementing command execution pipeline",
	})
}

// forwardToSSE forwards command stream responses to SSE manager
func (h *SSEHandler) forwardToSSE(ctx context.Context, room string, responseStream <-chan *pb.CommandStreamResponse) {
	for {
		select {
		case response, ok := <-responseStream:
			if !ok {
				// Stream closed, send final event
				h.sseManager.SendToRoom(room, &pb.CommandStreamResponse{
					Type:    pb.CommandStreamResponse_EXIT,
					Data:    "Stream completed",
					IsFinal: true,
				}, "command_complete")
				return
			}

			// Send response to room
			if err := h.sseManager.SendToRoom(room, response, "command_output"); err != nil {
				log.Printf("Failed to send command output to room %s: %v", room, err)
			}

			if response.IsFinal {
				return
			}

		case <-ctx.Done():
			// Context cancelled, send cancellation event
			h.sseManager.SendToRoom(room, &pb.CommandStreamResponse{
				Type:    pb.CommandStreamResponse_ERROR,
				Data:    "Command cancelled",
				IsFinal: true,
			}, "command_cancelled")
			return
		}
	}
}

// formatSSEMessage formats an SSE message for transmission
func formatSSEMessage(message sse.Message[*pb.CommandStreamResponse]) []byte {
	// Simple JSON formatting - you might want to use a proper JSON encoder
	data := `{"type":"` + message.EventType + `","data":`

	if response := message.Data; response != nil {
		data += `{"stream_type":"`
		switch response.Type {
		case pb.CommandStreamResponse_STDOUT:
			data += "stdout"
		case pb.CommandStreamResponse_STDERR:
			data += "stderr"
		case pb.CommandStreamResponse_EXIT:
			data += "exit"
		case pb.CommandStreamResponse_ERROR:
			data += "error"
		}
		data += `","content":"` + response.Data + `","exit_code":` + string(rune(response.ExitCode)) + `,"is_final":` + boolToString(response.IsFinal) + "}"
	} else {
		data += "null"
	}

	data += "}\n\n"
	return []byte("data: " + data)
}

// boolToString converts boolean to string
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// RegisterSSERoutes registers SSE routes for command streaming
func (h *SSEHandler) RegisterSSERoutes(router *gin.RouterGroup) {
	commands := router.Group("/commands")
	{
		commands.POST("/stream", h.StreamCommand)
	}
}
