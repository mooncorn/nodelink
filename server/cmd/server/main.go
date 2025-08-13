package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/agent"
	"google.golang.org/grpc"
)

// AgentStatusLogger implements the StatusChangeListener interface
type AgentStatusLogger struct{}

// OnStatusChange logs agent status changes
func (l *AgentStatusLogger) OnStatusChange(event agent.StatusChangeEvent) {
	log.Printf("Agent %s status changed: %s -> %s at %s",
		event.AgentID,
		event.OldStatus,
		event.NewStatus,
		event.Timestamp.Format(time.RFC3339))
}

func main() {
	defaultAgents := map[string]string{
		"agent1": "secret_token1",
		"agent2": "secret_token2",
	}

	auth := agent.NewDefaultAuthenticator(defaultAgents)

	agentRepo := agent.NewRepository()

	logger := &AgentStatusLogger{}
	agentRepo.AddListener(logger)

	pingPongServer := agent.NewPingPongServer(agent.PingPongServerConfig{
		OfflineTimeout:  6 * time.Second,
		PingInterval:    3 * time.Second,
		CleanupInterval: 12 * time.Second,
		StaleAgentTTL:   24 * time.Second,
		AgentRepo:       agentRepo,
		Authenticator:   auth,
	})

	// Start ping/pong server
	pingPongServer.Start(context.Background())
	defer pingPongServer.Stop()

	grpcServer := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcServer, pingPongServer)

	// Create HTTP and SSE handlers for agent management
	agentHTTPHandler := agent.NewHTTPHandler(agentRepo)
	agentSSEHandler := agent.NewSSEHandler(agentRepo)
	defer agentSSEHandler.Stop()

	router := gin.Default()

	// Register agent routes
	agentHTTPHandler.RegisterRoutes(router)
	agentSSEHandler.RegisterRoutes(router)

	// Start gRPC server in background
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		log.Println("gRPC Event Server starting on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	log.Println("HTTP Server starting on :8080")
	router.Run()
}
