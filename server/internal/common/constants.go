package common

import (
	"errors"
	"time"
)

var (
	// Common error definitions
	ErrAgentNotFound         = errors.New("agent not found")
	ErrAgentNotConnected     = errors.New("agent is not connected")
	ErrAgentAlreadyConnected = errors.New("agent is already connected")
	ErrRequestTimeout        = errors.New("request timed out")

	// Authentication error definitions
	ErrMissingMetadata    = errors.New("missing metadata")
	ErrMissingAgentID     = errors.New("missing agent_id")
	ErrMissingAgentToken  = errors.New("missing agent_token")
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Terminal-specific errors
	ErrTerminalSessionNotFound    = errors.New("terminal session not found")
	ErrTerminalSessionExists      = errors.New("terminal session already exists")
	ErrMaxTerminalSessionsReached = errors.New("maximum terminal sessions reached for user")
	ErrTerminalSessionClosed      = errors.New("terminal session is closed")
	ErrUnauthorizedTerminalAccess = errors.New("unauthorized access to terminal session")
)

const (
	// Ping/Pong defaults
	DefaultPingInterval    = 3 * time.Second
	DefaultOfflineTimeout  = 6 * time.Second
	DefaultCleanupInterval = 10 * time.Second
	DefaultStaleAgentTTL   = 30 * time.Second

	// Command execution constants
	DefaultCommandTimeout = 30 * time.Second
	MaxCommandTimeout     = 5 * time.Minute

	// Terminal session constants
	DefaultTerminalTimeout     = 30 * time.Minute
	DefaultTerminalShell       = "bash"
	MaxTerminalSessionsPerUser = 10
	TerminalCleanupInterval    = 5 * time.Minute
)
