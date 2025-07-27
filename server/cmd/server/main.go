package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	eventstream "github.com/mooncorn/nodelink/proto"
	grpclocal "github.com/mooncorn/nodelink/server/pkg/grpc"
	"github.com/mooncorn/nodelink/server/pkg/sse"
	"google.golang.org/grpc"
)

// Data to be broadcasted to a client.
type Data struct {
	Message  string `json:"message"`
	ClientId string `json:"clientId"`
}

func main() {
	// Create gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	eventServer := grpclocal.NewEventServer()
	eventstream.RegisterEventServiceServer(grpcServer, eventServer)

	// Add a simple event listener that logs all events
	eventServer.AddListener(func(event *eventstream.Event) {
		log.Printf("Server received event: %+v", event)
	})

	// Start gRPC server in background
	go func() {
		log.Println("gRPC Event Server starting on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Publish demo events periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		counter := 1

		for range ticker.C {
			taskID := fmt.Sprintf("demo-task-%d", counter)
			eventServer.Broadcast(&eventstream.Event{
				Payload: &eventstream.Event_TaskAssigned{
					TaskAssigned: &eventstream.TaskAssigned{
						TaskId: taskID}}})
			counter++
		}
	}()

	// HTTP/SSE server setup (existing code)
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

		// Send the data
		manager.Broadcast(data, "message")

		// Handle the stream
		sse.HandleSSEStream[Data](c)
	})

	log.Println("HTTP Server starting on :8080")
	router.Run()
}
