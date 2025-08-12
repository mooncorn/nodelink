package heartbeat

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AgentRepository interface {
	RegisterAgent(agentID string, metadata ...map[string]string)
	MarkConnected(agentID string)
	MarkDisconnected(agentID string)
}

type AgentHeartbeat struct {
	AgentID       string
	LastHeartbeat time.Time
}

type HeartbeatServer struct {
	pb.UnimplementedAgentServiceServer

	mu            sync.RWMutex
	agents        map[string]*AgentHeartbeat
	offlineTimers map[string]*time.Timer

	// Configuration
	offlineTimeout time.Duration

	// Dependencies
	agentRepo AgentRepository

	// Background context
	ctx    context.Context
	cancel context.CancelFunc
}

// HeartbeatServerConfig contains configuration for the heartbeat server
type HeartbeatServerConfig struct {
	OfflineTimeout time.Duration
	AgentRepo      AgentRepository
}

// AllowedAgents defines valid agent credentials (in production, use a proper auth service)
var AllowedAgents map[string]string = map[string]string{
	"agent1": "secret_token1",
	"agent2": "secret_token2",
}

// NewHeartbeatServer creates a new heartbeat server
func NewHeartbeatServer(config HeartbeatServerConfig) *HeartbeatServer {
	return &HeartbeatServer{
		agents:         make(map[string]*AgentHeartbeat),
		offlineTimers:  make(map[string]*time.Timer),
		offlineTimeout: config.OfflineTimeout,
		agentRepo:      config.AgentRepo,
	}
}

// Start begins the heartbeat server
func (s *HeartbeatServer) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)
}

// Stop gracefully shuts down the heartbeat server
func (s *HeartbeatServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop all offline timers
	for _, timer := range s.offlineTimers {
		timer.Stop()
	}
	s.offlineTimers = make(map[string]*time.Timer)
}

// HeartbeatStream implements the gRPC bidirectional streaming for agent heartbeats
func (s *HeartbeatServer) HeartbeatStream(stream pb.AgentService_HeartbeatStreamServer) error {
	// Authenticate agent from stream context
	agentID, _, err := s.authenticateAgent(stream.Context())
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	log.Printf("Agent %s connected via heartbeat stream", agentID)

	// Register agent connection
	s.registerAgent(agentID)
	defer s.unregisterAgent(agentID)

	// Create a context that cancels when the stream is done
	streamCtx, streamCancel := context.WithCancel(stream.Context())
	defer streamCancel()

	// Handle incoming heartbeat messages
	for {
		select {
		case <-streamCtx.Done():
			log.Printf("Stream context cancelled for agent %s", agentID)
			return streamCtx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("Agent %s closed heartbeat stream", agentID)
				break
			}
			log.Printf("Error receiving heartbeat from agent %s: %v", agentID, err)
			// Agent will be unregistered by defer, but ensure it's marked offline immediately
			s.markAgentOfflineImmediate(agentID)
			return err
		}

		// Process heartbeat message
		if err := s.processHeartbeat(agentID, msg); err != nil {
			log.Printf("Error processing heartbeat from agent %s: %v", agentID, err)
			continue
		}
	}

	return nil
}

// authenticateAgent extracts and validates agent credentials from gRPC metadata
func (s *HeartbeatServer) authenticateAgent(ctx context.Context) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	agentIDs := md.Get("agent_id")
	if len(agentIDs) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing agent_id")
	}

	agentTokens := md.Get("agent_token")
	if len(agentTokens) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing agent_token")
	}

	agentID := agentIDs[0]
	agentToken := agentTokens[0]

	// Validate credentials
	expectedToken, exists := AllowedAgents[agentID]
	if !exists || expectedToken != agentToken {
		return "", "", status.Errorf(codes.Unauthenticated, "invalid credentials for agent %s", agentID)
	}

	return agentID, agentToken, nil
}

// registerAgent registers an agent connection and updates connection status
func (s *HeartbeatServer) registerAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing offline timer if present
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Register agent
	s.agents[agentID] = &AgentHeartbeat{
		AgentID:       agentID,
		LastHeartbeat: time.Now(),
	}

	// Start initial offline timer
	s.offlineTimers[agentID] = time.AfterFunc(s.offlineTimeout, func() {
		go s.markAgentOffline(agentID)
	})

	// Update agent repository
	if s.agentRepo != nil {
		s.agentRepo.RegisterAgent(agentID)
		s.agentRepo.MarkConnected(agentID)
	}

	log.Printf("Registered agent %s for heartbeat monitoring", agentID)
}

// unregisterAgent removes an agent and marks it offline
func (s *HeartbeatServer) unregisterAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.agents, agentID)

	// Stop offline timer if it exists
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Update agent repository directly (we already have the mutex)
	if s.agentRepo != nil {
		s.agentRepo.MarkDisconnected(agentID)
	}

	log.Printf("Unregistered agent %s from heartbeat monitoring", agentID)
}

// processHeartbeat processes a heartbeat message from an agent
func (s *HeartbeatServer) processHeartbeat(agentID string, msg *pb.AgentMessage) error {
	// Verify message is a heartbeat or pong
	heartbeat := msg.GetHeartbeat()

	if heartbeat == nil {
		return fmt.Errorf("received non-heartbeat message from agent %s", agentID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return fmt.Errorf("received message from unregistered agent: %s", agentID)
	}

	// Update last heartbeat time
	agent.LastHeartbeat = time.Now()

	// Reset offline timer
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Start new offline timer
	s.offlineTimers[agentID] = time.AfterFunc(s.offlineTimeout, func() {
		go s.markAgentOffline(agentID)
	})

	// Update agent repository
	if s.agentRepo != nil {
		s.agentRepo.MarkConnected(agentID)
	}

	return nil
}

func (s *HeartbeatServer) markAgentOffline(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove offline timer
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Update agent repository
	if s.agentRepo != nil {
		s.agentRepo.MarkDisconnected(agentID)
	}

	log.Printf("Agent %s marked as offline due to timeout", agentID)
}

// markAgentOfflineImmediate marks an agent as offline immediately without mutex locking
// This is used when we need to mark an agent offline from within a function that may already hold the mutex
func (s *HeartbeatServer) markAgentOfflineImmediate(agentID string) {
	// Use a goroutine to avoid potential deadlock issues
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Remove offline timer
		if timer, exists := s.offlineTimers[agentID]; exists {
			timer.Stop()
			delete(s.offlineTimers, agentID)
		}

		// Update agent repository
		if s.agentRepo != nil {
			s.agentRepo.MarkDisconnected(agentID)
		}

		log.Printf("Agent %s marked as offline due to stream error", agentID)
	}()
}
