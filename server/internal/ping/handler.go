package ping

import (
	"context"
	"log"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/status"
)

// Config contains configuration for the ping handler
type Config struct {
	PingInterval    time.Duration
	OfflineTimeout  time.Duration
	CleanupInterval time.Duration
	StaleAgentTTL   time.Duration
}

// DefaultConfig returns a default ping configuration
func DefaultConfig() Config {
	return Config{
		PingInterval:    3 * time.Second,
		OfflineTimeout:  6 * time.Second,
		CleanupInterval: 1 * time.Hour,
		StaleAgentTTL:   24 * time.Hour,
	}
}

// Handler manages ping/pong communication and heartbeat monitoring
type Handler struct {
	mu            sync.RWMutex
	config        Config
	statusManager *status.Manager
	streamSenders map[string]common.StreamSender
	offlineTimers map[string]*time.Timer

	// Background context and cleanup
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHandler creates a new ping handler
func NewHandler(statusManager *status.Manager, config Config) *Handler {
	return &Handler{
		config:        config,
		statusManager: statusManager,
		streamSenders: make(map[string]common.StreamSender),
		offlineTimers: make(map[string]*time.Timer),
	}
}

// Start begins the ping handler and background cleanup tasks
func (h *Handler) Start(ctx context.Context) {
	h.ctx, h.cancel = context.WithCancel(ctx)

	// Start cleanup goroutine for stale agents
	if h.config.CleanupInterval > 0 && h.config.StaleAgentTTL > 0 {
		h.wg.Add(1)
		go h.cleanupStaleAgents()
	}
}

// Stop gracefully shuts down the ping handler
func (h *Handler) Stop() {
	if h.cancel != nil {
		h.cancel()
	}

	h.mu.Lock()

	// Stop all offline timers
	for _, timer := range h.offlineTimers {
		timer.Stop()
	}
	h.offlineTimers = make(map[string]*time.Timer)

	h.mu.Unlock()

	// Wait for background tasks to complete
	h.wg.Wait()
}

// RegisterAgent registers an agent for ping monitoring
func (h *Handler) RegisterAgent(agentID string, sender common.StreamSender) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Store the stream sender
	h.streamSenders[agentID] = sender

	// Stop existing offline timer if present
	if timer, exists := h.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(h.offlineTimers, agentID)
	}

	// Mark agent as online in status manager
	h.statusManager.SetAgentOnline(agentID)

	log.Printf("Agent %s registered for ping monitoring", agentID)
}

// UnregisterAgent unregisters an agent from ping monitoring
func (h *Handler) UnregisterAgent(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove stream sender
	delete(h.streamSenders, agentID)

	// Stop offline timer
	if timer, exists := h.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(h.offlineTimers, agentID)
	}

	// Mark agent as offline in status manager
	h.statusManager.SetAgentOffline(agentID)

	log.Printf("Agent %s unregistered from ping monitoring", agentID)
}

// HandlePong processes a pong message from an agent
func (h *Handler) HandlePong(agentID string, pong *pb.Pong) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Verify agent is registered
	if _, exists := h.streamSenders[agentID]; !exists {
		return nil // Agent not registered, ignore pong
	}

	// Update last seen time in status manager
	h.statusManager.UpdateLastSeen(agentID)

	// Stop existing offline timer
	if timer, exists := h.offlineTimers[agentID]; exists {
		timer.Stop()
		delete(h.offlineTimers, agentID)
	}

	// Create new offline timer
	h.offlineTimers[agentID] = time.AfterFunc(h.config.OfflineTimeout, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		// Double-check agent still exists before marking offline
		if _, exists := h.streamSenders[agentID]; exists {
			log.Printf("Agent %s marked as offline due to timeout", agentID)
			h.statusManager.SetAgentOffline(agentID)
		}
	})

	return nil
}

// SendPing sends a ping message to a specific agent
func (h *Handler) SendPing(agentID string) error {
	h.mu.RLock()
	sender, exists := h.streamSenders[agentID]
	h.mu.RUnlock()

	if !exists {
		return nil // Agent not registered
	}

	ping := &pb.ServerMessage{
		Message: &pb.ServerMessage_Ping{
			Ping: &pb.Ping{
				Timestamp: time.Now().UTC().Unix(),
			},
		},
	}

	return sender.SendToAgent(agentID, ping)
}

// StartPingLoop starts a periodic ping loop for an agent
func (h *Handler) StartPingLoop(agentID string) {
	h.wg.Add(1)
	go h.pingLoop(agentID)
}

// pingLoop sends periodic pings to an agent
func (h *Handler) pingLoop(agentID string) {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			log.Printf("Stopping ping loop for agent %s", agentID)
			return
		case <-ticker.C:
			h.mu.RLock()
			_, exists := h.streamSenders[agentID]
			h.mu.RUnlock()

			if !exists {
				log.Printf("Agent %s disconnected, stopping ping loop", agentID)
				return
			}

			if err := h.SendPing(agentID); err != nil {
				log.Printf("Error sending ping to agent %s: %v", agentID, err)
				return
			}
		}
	}
}

// cleanupStaleAgents periodically cleans up stale agent records
func (h *Handler) cleanupStaleAgents() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			log.Println("Stopping stale agent cleanup")
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-h.config.StaleAgentTTL)
			count := h.statusManager.CleanupStaleAgents(cutoff)
			if count > 0 {
				log.Printf("Cleaned up %d stale agent records", count)
			}
		}
	}
}
