package terminal

import (
	"encoding/json"
	"fmt"

	"github.com/mooncorn/nodelink/server/internal/common"
)

// TerminalMessageFormatter handles terminal-specific message formatting
type TerminalMessageFormatter struct{}

// NewTerminalMessageFormatter creates a new terminal message formatter
func NewTerminalMessageFormatter() *TerminalMessageFormatter {
	return &TerminalMessageFormatter{}
}

// FormatMessage formats a terminal SSE message
func (f *TerminalMessageFormatter) FormatMessage(msg common.SSEMessage) string {
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

// FormatTerminalConnectionMessage formats the initial connection message for terminal
func (f *TerminalMessageFormatter) FormatTerminalConnectionMessage(sessionID string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "terminal_connected",
		"data": map[string]interface{}{
			"session_id": sessionID,
			"message":    "Connected to terminal stream",
		},
	})
	return string(data)
}

// TerminalRooms provides room naming utilities for terminal events
type TerminalRooms struct{}

// NewTerminalRooms creates a new terminal rooms utility
func NewTerminalRooms() *TerminalRooms {
	return &TerminalRooms{}
}

// GetTerminalSessionRoom returns the room name for a terminal session
func (r *TerminalRooms) GetTerminalSessionRoom(sessionID string) string {
	return fmt.Sprintf("terminal_%s", sessionID)
}

// GetUserTerminalRoom returns the room name for all terminal sessions of a user
func (r *TerminalRooms) GetUserTerminalRoom(userID string) string {
	return fmt.Sprintf("user_terminals_%s", userID)
}
