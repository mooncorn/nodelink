package common

import (
	"context"
	"time"

	pb "github.com/mooncorn/nodelink/server/internal/proto"
)

// SSEMessage represents a message to be sent via SSE
type SSEMessage struct {
	EventType string `json:"event_type"`
	Data      any    `json:"data"`
	Room      string `json:"room,omitempty"`
}

// SSEClient represents a connected SSE client
type SSEClient interface {
	GetChannel() <-chan SSEMessage
	GetContext() context.Context
}

// StreamSender interface for sending messages to agents
// This is the unified interface used by all components
type StreamSender interface {
	SendToAgent(agentID string, message *pb.ServerMessage) error
}

// CommandResponseHandler interface for handling command responses
type CommandResponseHandler interface {
	HandleCommandResponse(response *pb.CommandResponse) error
	SetStreamSender(sender StreamSender)
}

// Authenticator interface for agent authentication
type Authenticator interface {
	Authenticate(ctx context.Context) (string, error)
}

// StatusManager interface for managing agent status
type StatusManager interface {
	GetAgent(agentID string) (*AgentInfo, bool)
	GetAllAgents() []*AgentInfo
	IsAgentOnline(agentID string) bool
	AddListener(listener StatusChangeListener)
}

// SSEManager interface for managing Server-Sent Events
type SSEManager interface {
	Start()
	Stop()
	AddClient(clientID string) SSEClient
	RemoveClient(clientID string)
	JoinRoom(clientID, room string) error
	SendToRoom(room string, data any, eventType string) error
}

// TerminalSessionManager interface for managing terminal sessions
type TerminalSessionManager interface {
	CreateSession(userID, agentID, shell, workingDir string, env map[string]string) (*TerminalSession, error)
	GetSession(sessionID string) (*TerminalSession, error)
	GetUserSessions(userID string) []*TerminalSession
	CloseSession(sessionID string) error
	UpdateLastActivity(sessionID string) error
	CleanupInactiveSessions(maxInactivity time.Duration) int
	ValidateSessionAccess(sessionID, userID string) error
}

// TerminalResponseHandler interface for handling terminal command responses
type TerminalResponseHandler interface {
	HandleTerminalCreateResponse(response *pb.TerminalCreateResponse) error
	HandleTerminalCommandResponse(response *pb.TerminalCommandResponse) error
	HandleTerminalCloseResponse(response *pb.TerminalCloseResponse) error
	SetStreamSender(sender StreamSender)
}
