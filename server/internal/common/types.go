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
