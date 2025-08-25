package comm

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"

	pb "github.com/mooncorn/nodelink/server/internal/proto"
	"github.com/mooncorn/nodelink/server/internal/auth"
	"github.com/mooncorn/nodelink/server/internal/command"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/metrics"
	"github.com/mooncorn/nodelink/server/internal/ping"
	"github.com/mooncorn/nodelink/server/internal/status"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// CommunicationServer handles bidirectional communication with agents
type CommunicationServer struct {
	pb.UnimplementedAgentServiceServer

	mu            sync.RWMutex
	activeStreams map[string]pb.AgentService_StreamCommunicationServer

	// Dependencies
	statusManager   *status.Manager
	pingHandler     *ping.Handler
	commandHandler  *command.Handler
	terminalHandler common.TerminalResponseHandler
	metricsHandler  *metrics.Handler
	auth            auth.Authenticator

	// Background context and cleanup
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// CommunicationConfig contains configuration for the communication server
type CommunicationConfig struct {
	StatusManager   *status.Manager
	PingHandler     *ping.Handler
	CommandHandler  *command.Handler
	TerminalHandler common.TerminalResponseHandler
	MetricsHandler  *metrics.Handler
	Authenticator   auth.Authenticator
}

// NewCommunicationServer creates a new communication server
func NewCommunicationServer(config CommunicationConfig) *CommunicationServer {
	ctx, cancel := context.WithCancel(context.Background())

	server := &CommunicationServer{
		activeStreams:   make(map[string]pb.AgentService_StreamCommunicationServer),
		statusManager:   config.StatusManager,
		pingHandler:     config.PingHandler,
		commandHandler:  config.CommandHandler,
		terminalHandler: config.TerminalHandler,
		metricsHandler:  config.MetricsHandler,
		auth:            config.Authenticator,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Set stream sender for handlers that need it
	if config.CommandHandler != nil {
		config.CommandHandler.SetStreamSender(server)
	}
	if config.TerminalHandler != nil {
		config.TerminalHandler.SetStreamSender(server)
	}
	if config.MetricsHandler != nil {
		config.MetricsHandler.SetStreamSender(server)
	}

	return server
}

// Start begins the communication server
func (s *CommunicationServer) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)
}

// Stop gracefully shuts down the communication server
func (s *CommunicationServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	s.mu.Lock()
	// Clear active streams
	for agentID := range s.activeStreams {
		log.Printf("Closing stream for agent %s during shutdown", agentID)
		delete(s.activeStreams, agentID)
	}
	s.mu.Unlock()

	// Wait for background tasks to complete
	s.wg.Wait()
}

// StreamCommunication implements the gRPC bidirectional streaming
func (s *CommunicationServer) StreamCommunication(stream pb.AgentService_StreamCommunicationServer) error {
	// Authenticate agent from stream context
	agentID, err := s.auth.Authenticate(stream.Context())
	if err != nil {
		return grpcstatus.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	log.Printf("Agent %s connected via communication stream", agentID)

	// Check if agent is already connected
	s.mu.Lock()
	if _, exists := s.activeStreams[agentID]; exists {
		s.mu.Unlock()
		return grpcstatus.Errorf(codes.AlreadyExists, "agent %s is already connected", agentID)
	}
	s.activeStreams[agentID] = stream
	s.mu.Unlock()

	// Register with ping handler
	s.pingHandler.RegisterAgent(agentID, s)

	// Start ping loop for this agent
	s.pingHandler.StartPingLoop(agentID)

	// Cleanup on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.activeStreams, agentID)
		s.mu.Unlock()

		s.pingHandler.UnregisterAgent(agentID)
		log.Printf("Agent %s disconnected", agentID)
	}()

	// Create a context that cancels when the stream is done
	streamCtx, streamCancel := context.WithCancel(stream.Context())
	defer streamCancel()

	// Handle incoming messages from agent
	for {
		// Check if context is cancelled
		if streamCtx.Err() != nil {
			if errors.Is(streamCtx.Err(), context.Canceled) {
				log.Printf("Stream context cancelled for agent %s", agentID)
			} else {
				log.Printf("Stream context error for agent %s: %v", agentID, streamCtx.Err())
			}
			return streamCtx.Err()
		}

		agentMsg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("Agent %s closed communication stream", agentID)
				break
			}
			log.Printf("Error receiving message from agent %s: %v", agentID, err)
			return err
		}

		// Handle different message types
		switch msg := agentMsg.Message.(type) {
		case *pb.AgentMessage_Pong:
			// Process pong message through ping handler
			if err := s.pingHandler.HandlePong(agentID, msg.Pong); err != nil {
				log.Printf("Error processing pong from agent %s: %v", agentID, err)
			}
		case *pb.AgentMessage_CommandResponse:
			// Process command response through command handler
			if s.commandHandler != nil {
				if err := s.commandHandler.HandleCommandResponse(msg.CommandResponse); err != nil {
					log.Printf("Error processing command response from agent %s: %v", agentID, err)
				}
			}
		case *pb.AgentMessage_TerminalCreateResponse:
			// Process terminal create response through terminal handler
			if s.terminalHandler != nil {
				if err := s.terminalHandler.HandleTerminalCreateResponse(msg.TerminalCreateResponse); err != nil {
					log.Printf("Error processing terminal create response from agent %s: %v", agentID, err)
				}
			}
		case *pb.AgentMessage_TerminalCommandResponse:
			// Process terminal command response through terminal handler
			if s.terminalHandler != nil {
				if err := s.terminalHandler.HandleTerminalCommandResponse(msg.TerminalCommandResponse); err != nil {
					log.Printf("Error processing terminal command response from agent %s: %v", agentID, err)
				}
			}
		case *pb.AgentMessage_TerminalCloseResponse:
			// Process terminal close response through terminal handler
			if s.terminalHandler != nil {
				if err := s.terminalHandler.HandleTerminalCloseResponse(msg.TerminalCloseResponse); err != nil {
					log.Printf("Error processing terminal close response from agent %s: %v", agentID, err)
				}
			}
		case *pb.AgentMessage_MetricsResponse:
			// Process metrics response through metrics handler
			if s.metricsHandler != nil {
				s.metricsHandler.HandleMetricsResponse(msg.MetricsResponse)
			}
		case *pb.AgentMessage_SystemInfoResponse:
			// Process system info response through metrics handler
			if s.metricsHandler != nil {
				s.metricsHandler.HandleSystemInfoResponse(msg.SystemInfoResponse)
			}
		default:
			log.Printf("Unknown message type received from agent %s: %T", agentID, msg)
		}
	}

	return nil
}

// SendToAgent implements the StreamSender interface for sending messages to agents
func (s *CommunicationServer) SendToAgent(agentID string, message *pb.ServerMessage) error {
	s.mu.RLock()
	stream, exists := s.activeStreams[agentID]
	s.mu.RUnlock()

	if !exists {
		return errors.New("agent not connected or stream not found")
	}

	return stream.Send(message)
}
