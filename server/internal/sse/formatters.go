package sse

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mooncorn/nodelink/server/internal/common"
)

// MessageFormatter provides generic message formatting functions for SSE
type MessageFormatter struct{}

// NewMessageFormatter creates a new message formatter instance
func NewMessageFormatter() *MessageFormatter {
	return &MessageFormatter{}
}

// FormatMessage formats a message for SSE transmission with error handling
func (f *MessageFormatter) FormatMessage(msg common.SSEMessage) string {
	data, err := json.Marshal(map[string]interface{}{
		"event": msg.EventType,
		"data":  msg.Data,
		"room":  msg.Room,
	})
	if err != nil {
		log.Printf("Error marshaling SSE message: %v", err)
		return fmt.Sprintf(`{"event":"%s","error":"failed to marshal data"}`, msg.EventType)
	}
	return string(data)
}

// FormatSimpleMessage formats a simple message with event type and data
func (f *MessageFormatter) FormatSimpleMessage(eventType string, data interface{}) string {
	result, _ := json.Marshal(map[string]interface{}{
		"event": eventType,
		"data":  data,
	})
	return string(result)
}

// FormatErrorMessage formats an error message for SSE transmission
func (f *MessageFormatter) FormatErrorMessage(eventType, errorMessage string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"event": eventType,
		"error": errorMessage,
	})
	return string(data)
}
