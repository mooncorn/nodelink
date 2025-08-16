package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/auth"
	"github.com/mooncorn/nodelink/server/internal/comm"
	"github.com/mooncorn/nodelink/server/internal/command"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/ping"
	"github.com/mooncorn/nodelink/server/internal/sse"
	"github.com/mooncorn/nodelink/server/internal/status"
	"github.com/mooncorn/nodelink/server/internal/terminal"
	"google.golang.org/grpc"
)

// AgentStatusLogger implements the StatusChangeListener interface
type AgentStatusLogger struct{}

// OnStatusChange logs agent status changes
func (l *AgentStatusLogger) OnStatusChange(event common.StatusChangeEvent) {
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

	auth := auth.NewDefaultAuthenticator(defaultAgents)

	// Create status manager (replaces agentRepo)
	statusManager := status.NewManager()

	// Create status change logger
	logger := &AgentStatusLogger{}
	statusManager.AddListener(logger)

	// Create ping handler
	pingHandler := ping.NewHandler(statusManager, ping.DefaultConfig())

	// Create command handler with status manager
	commandHandler := command.NewHandler(statusManager)

	// Create terminal session manager and handlers
	terminalSessionManager := terminal.NewSessionManager()
	defer terminalSessionManager.Stop()

	// Create and start SSE manager (before terminal handlers need it)
	sseManager := sse.NewManager()
	sseManager.Start()
	defer sseManager.Stop()

	terminalHandler := terminal.NewHandler(terminalSessionManager, statusManager, sseManager)

	// Create communication server with all dependencies
	commServer := comm.NewCommunicationServer(comm.CommunicationConfig{
		StatusManager:   statusManager,
		PingHandler:     pingHandler,
		CommandHandler:  commandHandler,
		TerminalHandler: terminalHandler,
		Authenticator:   auth,
	})

	// Start all services
	pingHandler.Start(context.Background())
	defer pingHandler.Stop()

	commServer.Start(context.Background())
	defer commServer.Stop()

	grpcServer := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcServer, commServer)

	// Create HTTP and SSE handlers for status management
	statusHTTPHandler := status.NewHTTPHandler(statusManager)

	statusSSEHandler := status.NewSSEHandler(statusManager, sseManager)
	defer statusSSEHandler.Stop()

	// Create command HTTP handler
	commandHTTPHandler := command.NewHTTPHandler(commandHandler)

	// Create terminal HTTP and SSE handlers
	terminalHTTPHandler := terminal.NewHTTPHandler(terminalHandler)
	terminalSSEHandler := terminal.NewSSEHandler(terminalHandler, sseManager)

	router := gin.Default()

	// Register status routes (replaces agent routes)
	statusHTTPHandler.RegisterRoutes(router)
	statusSSEHandler.RegisterRoutes(router)

	// Register command routes
	commandHTTPHandler.RegisterRoutes(router)

	// Register terminal routes
	terminalHTTPHandler.RegisterRoutes(router)
	terminalSSEHandler.RegisterRoutes(router)

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
