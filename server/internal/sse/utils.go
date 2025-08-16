package sse

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// Utils provides common utilities for SSE handlers
type Utils struct {
	formatter *MessageFormatter
}

// NewUtils creates a new SSE utilities instance
func NewUtils() *Utils {
	return &Utils{
		formatter: NewMessageFormatter(),
	}
}

// SetupSSEHeaders sets up standard SSE headers for a Gin context
func (u *Utils) SetupSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
}

// GenerateClientID generates a unique client ID with optional prefix and components
func (u *Utils) GenerateClientID(prefix string, components ...string) string {
	if prefix == "" {
		return fmt.Sprintf("client_%d", time.Now().UnixNano())
	}

	if len(components) == 0 {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}

	// Build the client ID with prefix and components
	clientID := prefix
	for _, component := range components {
		if component != "" {
			clientID += "_" + component
		}
	}
	clientID += fmt.Sprintf("_%d", time.Now().UnixNano())

	return clientID
}

// GetClientIDFromQuery retrieves client ID from query parameter with fallback generation
func (u *Utils) GetClientIDFromQuery(c *gin.Context, prefix string, components ...string) string {
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = u.GenerateClientID(prefix, components...)
	}
	return clientID
}

// ConnectionHandler provides a reusable pattern for handling SSE connections
type ConnectionHandler struct {
	utils      *Utils
	sseManager common.SSEManager
}

// NewConnectionHandler creates a new SSE connection handler
func NewConnectionHandler(sseManager common.SSEManager) *ConnectionHandler {
	return &ConnectionHandler{
		utils:      NewUtils(),
		sseManager: sseManager,
	}
}

// HandleConnection manages the complete SSE connection lifecycle
func (h *ConnectionHandler) HandleConnection(c *gin.Context, config ConnectionConfig) error {
	// Setup headers
	h.utils.SetupSSEHeaders(c)

	// Generate or get client ID
	clientID := h.utils.GetClientIDFromQuery(c, config.ClientIDPrefix, config.ClientIDComponents...)

	// Add client to SSE manager
	client := h.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return fmt.Errorf("failed to create SSE client")
	}

	// Join rooms if specified
	for _, room := range config.Rooms {
		if err := h.sseManager.JoinRoom(clientID, room); err != nil {
			// Log error but continue
			fmt.Printf("Error joining room %s: %v\n", room, err)
		}
	}

	// Handle the connection lifecycle
	defer h.sseManager.RemoveClient(clientID)

	// Send initial messages if provided
	for _, msg := range config.InitialMessages {
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", msg); err != nil {
			return fmt.Errorf("error writing initial SSE message: %v", err)
		}
		c.Writer.Flush()
	}

	// Keep connection alive and send messages
	for {
		select {
		case msg := <-client.GetChannel():
			// Use custom formatter if provided, otherwise use default
			var formattedMsg string
			if config.MessageFormatter != nil {
				formattedMsg = config.MessageFormatter(msg)
			} else {
				formattedMsg = h.utils.formatter.FormatMessage(msg)
			}

			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", formattedMsg); err != nil {
				return fmt.Errorf("error writing SSE message: %v", err)
			}
			c.Writer.Flush()

		case <-c.Request.Context().Done():
			return nil
		case <-client.GetContext().Done():
			return nil
		}
	}
}

// ConnectionConfig holds configuration for SSE connections
type ConnectionConfig struct {
	ClientIDPrefix     string
	ClientIDComponents []string
	Rooms              []string
	InitialMessages    []string
	MessageFormatter   func(common.SSEMessage) string
}
