package command

import (
	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/agent"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// Service represents the command service with all its dependencies
type Service struct {
	HTTPHandler   *HTTPHandler
	SSEHandler    *SSEHandler
	CommandServer *CommandServer
	Processor     *CommandProcessor
}

// ServiceConfig contains configuration for the command service
type ServiceConfig struct {
	// Add any command-specific configuration here
}

// NewService creates a new command service with all dependencies
func NewService(
	agentManager *agent.Manager,
	sseManager *sse.Manager[*pb.CommandStreamResponse],
	config ServiceConfig,
) *Service {
	// Create processor
	processor := NewCommandProcessor()

	// Create gRPC server
	commandServer := NewCommandServer(agentManager)

	// Create HTTP handler
	httpHandler := NewHTTPHandler(agentManager)

	// Create SSE handler
	sseHandler := NewSSEHandler(agentManager, sseManager, processor)

	return &Service{
		HTTPHandler:   httpHandler,
		SSEHandler:    sseHandler,
		CommandServer: commandServer,
		Processor:     processor,
	}
}

// RegisterHTTPRoutes registers HTTP routes for the command service
func (s *Service) RegisterHTTPRoutes(router gin.RouterGroup) {
	s.HTTPHandler.RegisterRoutes(&router)
}

// RegisterSSERoutes registers SSE routes for the command service
func (s *Service) RegisterSSERoutes(router gin.RouterGroup) {
	s.SSEHandler.RegisterSSERoutes(&router)
}

// GetGRPCServer returns the gRPC server for registration
func (s *Service) GetGRPCServer() *CommandServer {
	return s.CommandServer
}
