package common

import (
	"time"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
)

// AgentInfo contains information about an agent
type AgentInfo struct {
	AgentID     string            `json:"agent_id"`
	Status      AgentStatus       `json:"status"`
	LastSeen    time.Time         `json:"last_seen"`
	ConnectedAt *time.Time        `json:"connected_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// StatusChangeEvent represents a status change notification
type StatusChangeEvent struct {
	AgentID   string      `json:"agent_id"`
	OldStatus AgentStatus `json:"old_status"`
	NewStatus AgentStatus `json:"new_status"`
	Timestamp time.Time   `json:"timestamp"`
	Agent     *AgentInfo  `json:"agent"`
}

// StatusChangeListener defines the interface for status change notifications
type StatusChangeListener interface {
	OnStatusChange(event StatusChangeEvent)
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	SessionID    string            `json:"session_id"`
	UserID       string            `json:"user_id"`
	AgentID      string            `json:"agent_id"`
	Shell        string            `json:"shell"`
	WorkingDir   string            `json:"working_dir"`
	Status       TerminalStatus    `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActivity time.Time         `json:"last_activity"`
	Env          map[string]string `json:"env,omitempty"`
}

// TerminalStatus represents the status of a terminal session
type TerminalStatus string

const (
	TerminalStatusActive   TerminalStatus = "active"
	TerminalStatusInactive TerminalStatus = "inactive"
	TerminalStatusClosed   TerminalStatus = "closed"
)

// TerminalCommand represents a command being executed in a terminal session
type TerminalCommand struct {
	CommandID string    `json:"command_id"`
	SessionID string    `json:"session_id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
