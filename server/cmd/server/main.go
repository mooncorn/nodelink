package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/agent"
	"github.com/mooncorn/nodelink/server/internal/heartbeat"
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
	agentRepo := agent.NewRepository()

	logger := &AgentStatusLogger{}
	agentRepo.AddListener(logger)

	heartbeatServer := heartbeat.NewHeartbeatServer(heartbeat.HeartbeatServerConfig{
		OfflineTimeout: 6 * time.Second,
		AgentRepo:      agentRepo,
	})

	// Start heartbeat server
	heartbeatServer.Start(context.Background())
	defer heartbeatServer.Stop()

	grpcServer := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcServer, heartbeatServer)

	router := gin.Default()

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
