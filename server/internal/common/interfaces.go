package common

import (
	"context"

	pb "github.com/mooncorn/nodelink/proto"
)

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
