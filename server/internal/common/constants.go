package common

import (
	"errors"
	"time"
)

// Common error definitions
var (
	ErrAgentNotFound         = errors.New("agent not found")
	ErrAgentNotConnected     = errors.New("agent is not connected")
	ErrAgentAlreadyConnected = errors.New("agent is already connected")
	ErrRequestTimeout        = errors.New("request timed out")
	ErrMissingMetadata       = errors.New("missing metadata")
	ErrMissingAgentID        = errors.New("missing agent_id")
	ErrMissingAgentToken     = errors.New("missing agent_token")
	ErrInvalidCredentials    = errors.New("invalid credentials")
)

// Default timeout and interval constants
const (
	DefaultPingInterval    = 3 * time.Second
	DefaultOfflineTimeout  = 6 * time.Second
	DefaultCleanupInterval = 10 * time.Second
	DefaultStaleAgentTTL   = 30 * time.Second
	DefaultCommandTimeout  = 30 * time.Second
	MaxCommandTimeout      = 5 * time.Minute
)
