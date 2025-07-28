package main

import (
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
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	eventServer := grpclocal.NewEventServer()
	eventstream.RegisterEventServiceServer(grpcServer, eventServer)

	// Add a simple event listener that logs all events
	eventServer.AddListener(func(event *eventstream.NodeToServerEvent) {
		log.Printf("Server received event: %+v", event)
	})

	// Start gRPC server in background
	go func() {
		log.Println("gRPC Event Server starting on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Publish demo events periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		counter := 1

		var agentId = "agent1"

		for range ticker.C {
			eventId, err := eventServer.Send(&eventstream.ServerToNodeEvent{
				AgentId: agentId,
				Task: &eventstream.ServerToNodeEvent_LogMessage{
					LogMessage: &eventstream.LogMessage{
						Msg: "message to agent1",
					},
				},
			})
			if err != nil {
				log.Printf("\nFailed to send event: %v", err)
			}
			log.Printf("\nEvent %s sent to agent %s", eventId, agentId)
			counter++
		}
	}()

	// HTTP/SSE server setup
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

	router.POST("/agents/:agentId/shell", func(ctx *gin.Context) {
		// expected body
		// { cmd: "" }

		// expected response
		// { event_ref: "" }
		// event ref can be used for client to subscribe to incoming events from nodes

		// expected flow:
		// client sends a shell request with ls command
		// client recieves event_ref
		// client establishes sse connection with provided event_ref (subscription)
		// server relays events to interested clients using event_ref
	})

	log.Println("HTTP Server starting on :8080")
	router.Run()
}
