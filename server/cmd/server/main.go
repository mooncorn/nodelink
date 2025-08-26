package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/auth"
	"github.com/mooncorn/nodelink/server/internal/comm"
	"github.com/mooncorn/nodelink/server/internal/command"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/metrics"
	"github.com/mooncorn/nodelink/server/internal/ping"
	pb "github.com/mooncorn/nodelink/server/internal/proto"
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
	// Get ports from environment variables
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

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

	// Create metrics handler
	metricsHandler := metrics.NewHandler(statusManager)

	// Create metrics streaming manager
	metricsStreamingManager := metrics.NewStreamingManager(metricsHandler, statusManager, sseManager)

	// Create communication server with all dependencies
	commServer := comm.NewCommunicationServer(comm.CommunicationConfig{
		StatusManager:   statusManager,
		PingHandler:     pingHandler,
		CommandHandler:  commandHandler,
		TerminalHandler: terminalHandler,
		MetricsHandler:  metricsHandler,
		Authenticator:   auth,
	})

	// Start all services
	pingHandler.Start(context.Background())
	defer pingHandler.Stop()

	// Start metrics handler cleanup routine
	go metricsHandler.Start(context.Background())

	// Start metrics streaming manager
	metricsStreamingManager.Start()
	defer metricsStreamingManager.Stop()

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

	// Create metrics HTTP handler
	metricsHTTPHandler := metrics.NewHTTPHandler(metricsHandler, sseManager, metricsStreamingManager)

	// Create metrics SSE handler
	metricsSSEHandler := metrics.NewSSEHandler(metricsHandler, metricsStreamingManager, sseManager)

	router := gin.Default()

	// Configure CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://mooncorn.github.io/nodelink/",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "Cache-Control"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	// Register status routes (replaces agent routes)
	statusHTTPHandler.RegisterRoutes(router)
	statusSSEHandler.RegisterRoutes(router)

	// Register command routes
	commandHTTPHandler.RegisterRoutes(router)

	// Register terminal routes
	terminalHTTPHandler.RegisterRoutes(router)
	terminalSSEHandler.RegisterRoutes(router)

	// Register metrics routes
	metricsHTTPHandler.RegisterRoutes(router)
	metricsSSEHandler.RegisterRoutes(router)

	// Start gRPC server in background
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		log.Printf("gRPC Event Server starting on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	log.Printf("HTTP Server starting on :%s", httpPort)
	router.Run(":" + httpPort)
}
