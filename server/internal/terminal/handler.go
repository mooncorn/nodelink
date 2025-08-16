package terminal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// Handler manages terminal operations and command execution
type Handler struct {
	sessionManager common.TerminalSessionManager
	statusManager  common.StatusManager
	streamSender   common.StreamSender
	sseManager     common.SSEManager

	// Track pending commands and responses
	pendingCommands map[string]*TerminalCommand
	mu              sync.RWMutex
}

// TerminalCommand represents a command in execution
type TerminalCommand struct {
	CommandID string
	SessionID string
	UserID    string
	Command   string
	CreatedAt time.Time
	Response  chan *pb.TerminalCommandResponse
}

// NewHandler creates a new terminal handler
func NewHandler(sessionManager common.TerminalSessionManager, statusManager common.StatusManager, sseManager common.SSEManager) *Handler {
	return &Handler{
		sessionManager:  sessionManager,
		statusManager:   statusManager,
		sseManager:      sseManager,
		pendingCommands: make(map[string]*TerminalCommand),
	}
}

// SetStreamSender sets the stream sender for communicating with agents
func (h *Handler) SetStreamSender(sender common.StreamSender) {
	h.streamSender = sender
}

// CreateTerminalSession creates a new terminal session on an agent
func (h *Handler) CreateTerminalSession(ctx context.Context, userID, agentID, shell, workingDir string, env map[string]string) (*common.TerminalSession, error) {
	// Validate agent is connected
	if !h.statusManager.IsAgentOnline(agentID) {
		return nil, common.ErrAgentNotConnected
	}

	// Create session in memory
	session, err := h.sessionManager.CreateSession(userID, agentID, shell, workingDir, env)
	if err != nil {
		return nil, err
	}

	// Send create request to agent
	request := &pb.TerminalCreateRequest{
		SessionId:  session.SessionID,
		Shell:      session.Shell,
		WorkingDir: session.WorkingDir,
		Env:        session.Env,
	}

	message := &pb.ServerMessage{
		Message: &pb.ServerMessage_TerminalCreateRequest{
			TerminalCreateRequest: request,
		},
	}

	if err := h.streamSender.SendToAgent(agentID, message); err != nil {
		// Cleanup session if agent communication fails
		h.sessionManager.CloseSession(session.SessionID)
		return nil, fmt.Errorf("failed to send terminal create request to agent: %w", err)
	}

	return session, nil
}

// ExecuteCommand sends a command to a terminal session
func (h *Handler) ExecuteCommand(ctx context.Context, sessionID, userID, command string) (string, error) {
	// Validate session access
	if err := h.sessionManager.ValidateSessionAccess(sessionID, userID); err != nil {
		return "", err
	}

	// Get session details
	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	// Validate agent is still connected
	if !h.statusManager.IsAgentOnline(session.AgentID) {
		return "", common.ErrAgentNotConnected
	}

	// Generate command ID
	commandID := uuid.New().String()

	// Create command tracking
	termCmd := &TerminalCommand{
		CommandID: commandID,
		SessionID: sessionID,
		UserID:    userID,
		Command:   command,
		CreatedAt: time.Now(),
		Response:  make(chan *pb.TerminalCommandResponse, 100), // Buffer for streaming responses
	}

	// Store pending command
	h.mu.Lock()
	h.pendingCommands[commandID] = termCmd
	h.mu.Unlock()

	// Cleanup on completion
	defer func() {
		h.mu.Lock()
		delete(h.pendingCommands, commandID)
		h.mu.Unlock()
	}()

	// Send command request to agent
	request := &pb.TerminalCommandRequest{
		SessionId: sessionID,
		CommandId: commandID,
		Command:   command,
	}

	message := &pb.ServerMessage{
		Message: &pb.ServerMessage_TerminalCommandRequest{
			TerminalCommandRequest: request,
		},
	}

	if err := h.streamSender.SendToAgent(session.AgentID, message); err != nil {
		return "", fmt.Errorf("failed to send terminal command to agent: %w", err)
	}

	// Update last activity
	h.sessionManager.UpdateLastActivity(sessionID)

	return commandID, nil
}

// CloseTerminalSession closes a terminal session
func (h *Handler) CloseTerminalSession(ctx context.Context, sessionID, userID string) error {
	// Validate session access
	if err := h.sessionManager.ValidateSessionAccess(sessionID, userID); err != nil {
		return err
	}

	// Get session details
	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Send close request to agent if still connected
	if h.statusManager.IsAgentOnline(session.AgentID) {
		request := &pb.TerminalCloseRequest{
			SessionId: sessionID,
		}

		message := &pb.ServerMessage{
			Message: &pb.ServerMessage_TerminalCloseRequest{
				TerminalCloseRequest: request,
			},
		}

		// Send close request (ignore errors since we're closing anyway)
		h.streamSender.SendToAgent(session.AgentID, message)
	}

	// Close session locally
	return h.sessionManager.CloseSession(sessionID)
}

// GetUserSessions returns all terminal sessions for a user
func (h *Handler) GetUserSessions(userID string) []*common.TerminalSession {
	return h.sessionManager.GetUserSessions(userID)
}

// HandleTerminalCreateResponse handles responses from agent for terminal creation
func (h *Handler) HandleTerminalCreateResponse(response *pb.TerminalCreateResponse) error {
	// If terminal creation failed on agent, cleanup local session
	if !response.Success {
		log.Printf("Terminal creation failed on agent for session %s: %s", response.SessionId, response.Error)
		h.sessionManager.CloseSession(response.SessionId)
		return fmt.Errorf("terminal creation failed: %s", response.Error)
	}

	log.Printf("Terminal session %s created successfully on agent", response.SessionId)
	return nil
}

// HandleTerminalCommandResponse handles streaming command responses from agent
func (h *Handler) HandleTerminalCommandResponse(response *pb.TerminalCommandResponse) error {
	// Handle both command-specific responses and session output
	if response.CommandId != "" {
		// This is a response to a specific command
		h.mu.RLock()
		_, exists := h.pendingCommands[response.CommandId]
		h.mu.RUnlock()

		if !exists {
			log.Printf("Received response for unknown command: %s", response.CommandId)
			// Still continue to broadcast the output via SSE
		}
	}

	// Send output via SSE to client (regardless of command ID)
	roomID := fmt.Sprintf("terminal_%s", response.SessionId)

	outputData := map[string]interface{}{
		"session_id": response.SessionId,
		"command_id": response.CommandId,
		"output":     response.Output,
		"error":      response.Error,
		"is_final":   response.IsFinal,
	}

	if response.IsFinal {
		outputData["exit_code"] = response.ExitCode
	}

	// Send to SSE room
	if err := h.sseManager.SendToRoom(roomID, outputData, "terminal_output"); err != nil {
		log.Printf("Failed to send terminal output to SSE room %s: %v", roomID, err)
	}

	// Update last activity
	h.sessionManager.UpdateLastActivity(response.SessionId)

	return nil
}

// HandleTerminalCloseResponse handles responses from agent for terminal closure
func (h *Handler) HandleTerminalCloseResponse(response *pb.TerminalCloseResponse) error {
	if !response.Success {
		log.Printf("Terminal close failed on agent for session %s: %s", response.SessionId, response.Error)
	} else {
		log.Printf("Terminal session %s closed successfully on agent", response.SessionId)
	}

	// Always cleanup local session
	return h.sessionManager.CloseSession(response.SessionId)
}

// GetTerminalSessionRoom returns the SSE room name for a terminal session
func GetTerminalSessionRoom(sessionID string) string {
	return fmt.Sprintf("terminal_%s", sessionID)
}
