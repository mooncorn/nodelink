package common

import (
	"context"

	pb "github.com/mooncorn/nodelink/proto"
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
	UpdateAgentStatus(agentID string, status AgentStatus) error
	GetAgent(agentID string) (*AgentInfo, error)
	GetAllAgents() map[string]*AgentInfo
	IsAgentOnline(agentID string) bool
	AddListener(listener StatusChangeListener)
	RemoveListener(listener StatusChangeListener)
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
