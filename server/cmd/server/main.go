package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/pkg/sse"
)

// Data to be broadcasted to a client.
type Data struct {
	Message  string `json:"message"`
	ClientId string `json:"clientId"`
}

func main() {
	router := gin.Default()

	config := sse.ManagerConfig{
		BufferSize:     100,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	eventHandler := sse.NewDefaultEventHandler[Data](true)
	manager := sse.NewManager(config, eventHandler)
	manager.Start()
	defer manager.Stop()

	router.GET("/stream", sse.SSEHeaders(), sse.SSEConnection(manager), func(c *gin.Context) {
		client, ok := sse.GetClientFromContext[Data](c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}

		// Send welcome message
		data := Data{
			Message:  "New Client in town",
			ClientId: string(client.ID),
		}

		// Send the data (this replaces your manual goroutine)
		manager.Broadcast(data, "message")

		// Handle the stream
		sse.HandleSSEStream[Data](c)
	})

	router.Run()
}

func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Next()
	}
}
