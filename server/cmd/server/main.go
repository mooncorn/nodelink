package main

import (
	"log"
	"net"
	"net/http"

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

	config := sse.ManagerConfig{
		BufferSize:     100,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	eventHandler := sse.NewDefaultEventHandler[eventstream.NodeToServerEvent_EventResponse](true)
	manager := sse.NewManager(config, eventHandler)
	manager.Start()
	defer manager.Stop()

	// Listener that sends event responses to appropriate rooms
	eventServer.AddListener(func(event *eventstream.NodeToServerEvent) {
		log.Printf("Server received event: %+v", event)

		switch event := event.Event.(type) {
		case *eventstream.NodeToServerEvent_EventResponse:
			manager.EnableRoomBuffering(event.EventResponse.EventRef, 3)
			manager.SendToRoom(event.EventResponse.EventRef, *event, "response")
		}
	})

	// Start gRPC server in background
	go func() {
		log.Println("gRPC Event Server starting on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// HTTP/SSE server setup
	router := gin.Default()

	// Subscribe client to event reference
	// TODO: add auth
	router.GET("/stream", sse.SSEHeaders(), sse.SSEConnection(manager), func(c *gin.Context) {
		client, ok := sse.GetClientFromContext[eventstream.NodeToServerEvent_EventResponse](c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}

		eventRef := c.Query("ref")
		if eventRef != "" {
			manager.JoinRoom(client.ID, eventRef)
		}

		// Handle the stream
		sse.HandleSSEStream[eventstream.NodeToServerEvent_EventResponse](c)
	})

	router.POST("/agents/:agentId/shell", func(ctx *gin.Context) {
		agentId := ctx.Param("agentId")
		var req struct {
			Cmd string `json:"cmd"`
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		eventId, err := eventServer.Send(&eventstream.ServerToNodeEvent{
			AgentId: agentId,
			Task: &eventstream.ServerToNodeEvent_ShellExecute{
				ShellExecute: &eventstream.ShellExecute{
					Cmd: req.Cmd,
				},
			},
		})
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"ref": eventId})
	})

	log.Println("HTTP Server starting on :8080")
	router.Run()
}
