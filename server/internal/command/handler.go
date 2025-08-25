package command

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/server/internal/proto"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/status"
)

var (
	ErrAgentNotConnected = common.ErrAgentNotConnected
	ErrRequestTimeout    = common.ErrRequestTimeout
)

// Request represents a pending command request
type Request struct {
	ID         string
	AgentID    string
	Command    string
	Args       []string
	Env        map[string]string
	WorkingDir string
	Timeout    time.Duration
	Response   chan *pb.CommandResponse
	CreatedAt  time.Time
}

// Handler manages command execution requests to agents
type Handler struct {
	mu              sync.RWMutex
	pendingRequests map[string]*Request
	statusManager   *status.Manager
	streamSender    common.StreamSender
	defaultTimeout  time.Duration
	maxTimeout      time.Duration
	requestCounter  int64
}

// NewHandler creates a new command handler
func NewHandler(statusManager *status.Manager) *Handler {
	return &Handler{
		pendingRequests: make(map[string]*Request),
		statusManager:   statusManager,
		defaultTimeout:  30 * time.Second,
		maxTimeout:      5 * time.Minute,
	}
}

// SetStreamSender sets the stream sender for the handler
func (h *Handler) SetStreamSender(sender common.StreamSender) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streamSender = sender
}

// ExecuteCommand sends a command to an agent and waits for the response
func (h *Handler) ExecuteCommand(ctx context.Context, agentID, command string, args []string, env map[string]string, workingDir string, timeout time.Duration) (*pb.CommandResponse, error) {
	// Validate agent is connected using status manager
	if !h.statusManager.IsAgentOnline(agentID) {
		return nil, ErrAgentNotConnected
	}

	// Validate and set timeout
	if timeout <= 0 {
		timeout = h.defaultTimeout
	}
	if timeout > h.maxTimeout {
		timeout = h.maxTimeout
	}

	// Generate request ID
	requestID := h.generateRequestID()

	// Create request
	request := &Request{
		ID:         requestID,
		AgentID:    agentID,
		Command:    command,
		Args:       args,
		Env:        env,
		WorkingDir: workingDir,
		Timeout:    timeout,
		Response:   make(chan *pb.CommandResponse, 1),
		CreatedAt:  time.Now(),
	}

	// Store pending request
	h.mu.Lock()
	h.pendingRequests[requestID] = request
	h.mu.Unlock()

	// Cleanup on completion
	defer func() {
		h.mu.Lock()
		delete(h.pendingRequests, requestID)
		h.mu.Unlock()
	}()

	// Create command request message
	cmdReq := &pb.CommandRequest{
		RequestId:      requestID,
		Command:        command,
		Args:           args,
		Env:            env,
		WorkingDir:     workingDir,
		TimeoutSeconds: int32(timeout.Seconds()),
	}

	// Send command request to agent
	if err := h.sendCommandToAgent(agentID, cmdReq); err != nil {
		return nil, fmt.Errorf("failed to send command to agent: %w", err)
	}

	// Wait for response or timeout
	select {
	case response := <-request.Response:
		return response, nil
	case <-time.After(timeout + 5*time.Second): // Add buffer for network latency
		return nil, ErrRequestTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// HandleCommandResponse processes a command response from an agent
func (h *Handler) HandleCommandResponse(response *pb.CommandResponse) error {
	h.mu.RLock()
	request, exists := h.pendingRequests[response.RequestId]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no pending request found for ID: %s", response.RequestId)
	}

	// Send response to waiting goroutine
	select {
	case request.Response <- response:
		return nil
	default:
		return fmt.Errorf("failed to deliver response for request ID: %s", response.RequestId)
	}
}

// GetPendingRequests returns all pending requests for monitoring
func (h *Handler) GetPendingRequests() map[string]*Request {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*Request)
	for k, v := range h.pendingRequests {
		result[k] = v
	}
	return result
}

// generateRequestID generates a unique request ID
func (h *Handler) generateRequestID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requestCounter++
	return fmt.Sprintf("cmd_%d_%d", time.Now().Unix(), h.requestCounter)
}

// sendCommandToAgent sends a command request to an agent
func (h *Handler) sendCommandToAgent(agentID string, cmdReq *pb.CommandRequest) error {
	h.mu.RLock()
	sender := h.streamSender
	h.mu.RUnlock()

	if sender == nil {
		return fmt.Errorf("stream sender not configured")
	}

	serverMsg := &pb.ServerMessage{
		Message: &pb.ServerMessage_CommandRequest{
			CommandRequest: cmdReq,
		},
	}

	return sender.SendToAgent(agentID, serverMsg)
}
