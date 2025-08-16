package status

import (
	"encoding/json"
	"fmt"

	"github.com/mooncorn/nodelink/server/internal/common"
)

// StatusMessageFormatter handles status-specific message formatting
type StatusMessageFormatter struct{}

// NewStatusMessageFormatter creates a new status message formatter
func NewStatusMessageFormatter() *StatusMessageFormatter {
	return &StatusMessageFormatter{}
}

// FormatMessage formats a status SSE message
func (f *StatusMessageFormatter) FormatMessage(msg common.SSEMessage) string {
	data, err := json.Marshal(map[string]interface{}{
		"event": msg.EventType,
		"data":  msg.Data,
		"room":  msg.Room,
	})
	if err != nil {
		return fmt.Sprintf(`{"event":"%s","error":"failed to marshal data"}`, msg.EventType)
	}
	return string(data)
}

// FormatConnectionMessage formats the initial connection message
func (f *StatusMessageFormatter) FormatConnectionMessage() string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "connection",
		"data":  map[string]string{"status": "connected"},
	})
	return string(data)
}

// FormatAgentConnectionMessage formats the initial connection message for a specific agent
func (f *StatusMessageFormatter) FormatAgentConnectionMessage(agentID string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "agent_connection",
		"data": map[string]string{
			"status":   "connected",
			"agent_id": agentID,
		},
	})
	return string(data)
}

// FormatCurrentStatusMessage formats the current status message for an agent
func (f *StatusMessageFormatter) FormatCurrentStatusMessage(agent *common.AgentInfo) string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "current_status",
		"data":  agent,
	})
	return string(data)
}

// StatusRooms provides room naming utilities for status events
type StatusRooms struct{}

// NewStatusRooms creates a new status rooms utility
func NewStatusRooms() *StatusRooms {
	return &StatusRooms{}
}

// GetAllAgentsRoom returns the room name for all agent events
func (r *StatusRooms) GetAllAgentsRoom() string {
	return "agents"
}

// GetAgentSpecificRoom returns the room name for a specific agent
func (r *StatusRooms) GetAgentSpecificRoom(agentID string) string {
	return fmt.Sprintf("agent_%s", agentID)
}
