package agent

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidPong           = errors.New("received non-pong message")
	ErrUnregisteredAgent     = errors.New("received message from unregistered agent")
	ErrAgentAlreadyConnected = errors.New("agent is already connected")
)

type AgentPingPong struct {
	LastPong time.Time
}

type PingPongServer struct {
	pb.UnimplementedAgentServiceServer

	mu            sync.RWMutex
	agents        map[string]*AgentPingPong
	offlineTimers map[string]*time.Timer
	activeStreams map[string]context.CancelFunc

	config PingPongServerConfig

	// Dependencies
	agentRepo *Repository
	auth      Authenticator

	// Background context and cleanup
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// PingPongServerConfig contains configuration for the ping/pong server
type PingPongServerConfig struct {
	OfflineTimeout  time.Duration
	PingInterval    time.Duration
	CleanupInterval time.Duration
	StaleAgentTTL   time.Duration
	AgentRepo       *Repository
	Authenticator   Authenticator
}

// NewPingPongServer creates a new ping/pong server with the provided configuration.
// It merges the provided config with defaults and validates required fields.
func NewPingPongServer(config PingPongServerConfig) *PingPongServer {
	// Merge with defaults
	if config.OfflineTimeout == 0 {
		config.OfflineTimeout = 6 * time.Second
	}
	if config.PingInterval == 0 {
		config.PingInterval = 3 * time.Second
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}
	if config.StaleAgentTTL == 0 {
		config.StaleAgentTTL = 24 * time.Hour
	}

	return &PingPongServer{
		agents:        make(map[string]*AgentPingPong),
		offlineTimers: make(map[string]*time.Timer),
		activeStreams: make(map[string]context.CancelFunc),
		config:        config,
		agentRepo:     config.AgentRepo,
		auth:          config.Authenticator,
	}
}

// Start begins the ping/pong server and background cleanup tasks.
// It should be called after creating the server and before handling streams.
func (s *PingPongServer) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start cleanup goroutine for stale agents
	if s.config.CleanupInterval > 0 && s.config.StaleAgentTTL > 0 {
		s.wg.Add(1)
		go s.cleanupStaleAgents()
	}
}

// Stop gracefully shuts down the ping/pong server and waits for background tasks to complete.
func (s *PingPongServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	s.mu.Lock()

	// Cancel all active streams
	for agentID, cancelFunc := range s.activeStreams {
		log.Printf("Cancelling stream for agent %s during shutdown", agentID)
		cancelFunc()
	}
	s.activeStreams = make(map[string]context.CancelFunc)

	// Stop all offline timers
	for _, timer := range s.offlineTimers {
		timer.Stop()
	}
	s.offlineTimers = make(map[string]*time.Timer)

	s.mu.Unlock()

	// Wait for background tasks to complete
	s.wg.Wait()
}

// HeartbeatStream implements the gRPC bidirectional streaming for agent ping/pong.
// It handles agent authentication, registration, and implements a ping/pong mechanism
// where the server sends pings and agents respond with pongs for connection monitoring.
func (s *PingPongServer) StreamPingPong(stream pb.AgentService_StreamPingPongServer) error {
	// Authenticate agent from stream context
	agentID, err := s.auth.Authenticate(stream.Context())
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	log.Printf("Agent %s connected via ping/pong stream", agentID)

	// Register agent connection (this will handle disconnecting any existing agent with same ID)
	err = s.registerAgent(agentID)
	if err != nil {
		return status.Errorf(codes.AlreadyExists, "agent %s is already connected: %v", agentID, err)
	}
	defer s.unregisterAgent(agentID)

	// Create a context that cancels when the stream is done
	streamCtx, streamCancel := context.WithCancel(stream.Context())
	defer streamCancel()

	// Store stream cancel function for graceful shutdown
	s.mu.Lock()
	s.activeStreams[agentID] = streamCancel
	s.mu.Unlock()

	// Start ping sender goroutine
	s.wg.Add(1)
	go s.sendPings(streamCtx, stream, agentID)

	// Handle incoming pong messages
	for {
		// Check if context is cancelled (more explicit than select with default)
		if streamCtx.Err() != nil {
			if errors.Is(streamCtx.Err(), context.Canceled) {
				log.Printf("Stream context cancelled for agent %s", agentID)
			} else {
				log.Printf("Stream context error for agent %s: %v", agentID, streamCtx.Err())
			}
			return streamCtx.Err()
		}

		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("Agent %s closed ping/pong stream", agentID)
				break
			}
			log.Printf("Error receiving pong from agent %s: %v", agentID, err)
			return err
		}

		// Process pong message
		if err := s.processPong(agentID, msg); err != nil {
			log.Printf("Error processing pong from agent %s: %v", agentID, err)
			continue
		}
	}

	return nil
}

// registerAgent registers an agent connection and updates connection status.
func (s *PingPongServer) registerAgent(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if there's already an active agent with this ID
	if _, exists := s.agents[agentID]; exists {
		return ErrAgentAlreadyConnected
	}

	// Stop existing offline timer if present (additional safety check)
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Register agent with explicit UTC timestamp
	now := time.Now().UTC()
	s.agents[agentID] = &AgentPingPong{
		LastPong: now,
	}

	// Start initial offline timer
	s.offlineTimers[agentID] = time.AfterFunc(s.config.OfflineTimeout, func() {
		go s.markAgentOffline(agentID)
	})

	// Update agent repository
	s.agentRepo.RegisterAgent(agentID)
	s.agentRepo.MarkConnected(agentID)
	s.agentRepo.UpdateLastSeen(agentID, now)

	log.Printf("Registered agent %s for ping/pong monitoring", agentID)
	return nil
}

// unregisterAgent removes an agent and marks it offline
func (s *PingPongServer) unregisterAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.agents, agentID)

	// Stop offline timer if it exists
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Remove from active streams
	if cancelFunc, exists := s.activeStreams[agentID]; exists {
		cancelFunc()
		delete(s.activeStreams, agentID)
	}

	// Update agent repository directly (we already have the mutex)
	s.agentRepo.MarkDisconnected(agentID)

	log.Printf("Unregistered agent %s from ping/pong monitoring", agentID)
}

// processPong processes a pong message from an agent
func (s *PingPongServer) processPong(agentID string, pong *pb.Pong) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.agents[agentID]
	if !exists {
		return ErrUnregisteredAgent
	}

	diff := time.Now().UTC().Sub(time.UnixMicro(pong.PingTimestamp))

	log.Printf("Received pong from agent %s (roundtrip: %v)", agentID, diff)

	// Reset offline timer
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Start new offline timer
	s.offlineTimers[agentID] = time.AfterFunc(s.config.OfflineTimeout, func() {
		go s.markAgentOffline(agentID)
	})

	now := time.Now().UTC()

	// Update agent repository
	s.agentRepo.MarkConnected(agentID)
	s.agentRepo.UpdateLastSeen(agentID, now)

	return nil
}

// markAgentOffline marks an agent as offline due to timeout and performs cleanup
func (s *PingPongServer) markAgentOffline(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove offline timer
	if timer, exists := s.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(s.offlineTimers, agentID)
	}

	// Remove agent from active tracking (fully evict timed-out agents)
	delete(s.agents, agentID)

	// Cancel stream if still active
	if cancelFunc, exists := s.activeStreams[agentID]; exists {
		cancelFunc()
		delete(s.activeStreams, agentID)
	}

	// Update agent repository
	s.agentRepo.MarkDisconnected(agentID)

	log.Printf("Agent %s marked as offline and evicted due to timeout", agentID)
}

// sendPings sends periodic ping messages to an agent for connection monitoring.
// This goroutine runs alongside the PingPongStream to proactively detect connection issues.
func (s *PingPongServer) sendPings(ctx context.Context, stream pb.AgentService_StreamPingPongServer, agentID string) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping ping goroutine for agent %s: %v", agentID, ctx.Err())
			return
		case <-s.ctx.Done():
			log.Printf("Stopping ping goroutine for agent %s due to server shutdown", agentID)
			return
		case <-ticker.C:
			// Send ping message to agent
			pingMsg := &pb.Ping{
				Timestamp: time.Now().UTC().UnixMicro(),
			}

			if err := stream.Send(pingMsg); err != nil {
				log.Printf("Failed to send ping to agent %s: %v", agentID, err)
				// Connection is likely broken, let the main stream handler deal with it
				return
			}
		}
	}
}

// cleanupStaleAgents periodically removes agents that have been disconnected for too long.
// This prevents memory bloat from accumulating phantom agents over time.
func (s *PingPongServer) cleanupStaleAgents() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Stopping stale agent cleanup due to server shutdown")
			return
		case <-ticker.C:
			s.performStaleAgentCleanup()
		}
	}
}

// performStaleAgentCleanup removes agents from the repository that have been disconnected
// for longer than the configured TTL.
func (s *PingPongServer) performStaleAgentCleanup() {
	// Calculate the cutoff time for stale agents
	cutoffTime := time.Now().UTC().Add(-s.config.StaleAgentTTL)

	// Get all disconnected agents from the repository
	disconnectedAgents := s.agentRepo.GetDisconnectedAgents()
	cleanedCount := 0

	for _, agentID := range disconnectedAgents {
		// Get detailed agent info to check disconnection time
		agentInfo, exists := s.agentRepo.GetAgent(agentID)
		if !exists {
			continue
		}

		// Check if agent has been disconnected long enough to be considered stale
		if agentInfo.DisconnectedAt != nil && agentInfo.DisconnectedAt.Before(cutoffTime) {
			s.agentRepo.UnregisterAgent(agentID)
			cleanedCount++
			log.Printf("Cleaned up stale agent %s (disconnected at %v, TTL: %v)",
				agentID, agentInfo.DisconnectedAt.Format(time.RFC3339), s.config.StaleAgentTTL)
		}
	}

	if cleanedCount > 0 {
		log.Printf("Cleaned up %d stale agents (disconnected for more than %v)", cleanedCount, s.config.StaleAgentTTL)
	}
}

// GetActiveAgents returns a list of currently active agent IDs
func (s *PingPongServer) GetActiveAgents() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]string, 0, len(s.agents))
	for agentID := range s.agents {
		agents = append(agents, agentID)
	}
	return agents
}
