package agent

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrMissingMetadata    = errors.New("missing metadata")
	ErrMissingAgentID     = errors.New("missing agent_id")
	ErrMissingAgentToken  = errors.New("missing agent_token")
)

// Authenticator interface for agent authentication
type Authenticator interface {
	Authenticate(ctx context.Context) (string, error)
}

// DefaultAuthenticator uses a static map for authentication (development/testing only)
type DefaultAuthenticator struct {
	allowedAgents map[string]string
}

// NewDefaultAuthenticator creates a new default authenticator
func NewDefaultAuthenticator(allowedAgents map[string]string) *DefaultAuthenticator {
	return &DefaultAuthenticator{
		allowedAgents: allowedAgents,
	}
}

// Authenticate validates agent credentials
func (a *DefaultAuthenticator) Authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, ErrMissingMetadata.Error())
	}

	agentIDs := md.Get("agent_id")
	if len(agentIDs) == 0 {
		return "", status.Error(codes.Unauthenticated, ErrMissingAgentID.Error())
	}

	agentTokens := md.Get("agent_token")
	if len(agentTokens) == 0 {
		return "", status.Error(codes.Unauthenticated, ErrMissingAgentToken.Error())
	}

	agentID := agentIDs[0]
	agentToken := agentTokens[0]

	expectedToken, exists := a.allowedAgents[agentID]
	if !exists || expectedToken != agentToken {
		return "", ErrInvalidCredentials
	}
	return agentID, nil
}
