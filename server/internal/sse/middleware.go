package sse

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SSEHeaders sets the required headers for SSE
func SSEHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
		c.Next()
	}
}

// SSEConnection creates middleware for handling SSE connections
func SSEConnection[T any](manager *Manager[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or generate client ID
		clientIDParam := c.Query("client_id")
		var clientID ClientID
		if clientIDParam != "" {
			clientID = ClientID(clientIDParam)
		} else {
			// Simple client ID generation without uuid dependency
			clientID = ClientID("client_" + c.Request.RemoteAddr + "_" + c.Request.Header.Get("User-Agent"))
		}

		// Add client to manager
		client := manager.AddClient(clientID)

		// Set client in context
		c.Set("client", client)
		c.Set("client_id", clientID)

		// Handle client disconnect
		defer func() {
			manager.RemoveClient(clientID)
		}()

		c.Next()
	}
}

// HandleSSEStream handles the SSE streaming for a client
func HandleSSEStream[T any](c *gin.Context) {
	clientInterface, exists := c.Get("client")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "client not found in context"})
		return
	}

	client, ok := clientInterface.(*Client[T])
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid client type"})
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case message, ok := <-client.Channel:
			if !ok {
				return false
			}

			// Serialize the message data
			data, err := json.Marshal(message.Data)
			if err != nil {
				return false
			}

			// Send SSE event
			eventType := message.EventType
			if eventType == "" {
				eventType = "message"
			}

			c.SSEvent(eventType, string(data))
			return true

		case <-client.Context.Done():
			return false

		case <-c.Request.Context().Done():
			return false
		}
	})
}

// GetClientFromContext retrieves the client from gin context
func GetClientFromContext[T any](c *gin.Context) (*Client[T], bool) {
	clientInterface, exists := c.Get("client")
	if !exists {
		return nil, false
	}

	client, ok := clientInterface.(*Client[T])
	return client, ok
}

// GetClientIDFromContext retrieves the client ID from gin context
func GetClientIDFromContext(c *gin.Context) (ClientID, bool) {
	clientIDInterface, exists := c.Get("client_id")
	if !exists {
		return "", false
	}

	clientID, ok := clientIDInterface.(ClientID)
	return clientID, ok
}
