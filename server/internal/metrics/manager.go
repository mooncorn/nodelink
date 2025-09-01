package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mooncorn/nodelink/server/internal/common"
	pb "github.com/mooncorn/nodelink/server/internal/proto"
)

// StreamingManager manages continuous metrics collection and distribution
type StreamingManager struct {
	handler       *Handler
	statusManager common.StatusManager
	sseManager    common.SSEManager

	mu              sync.RWMutex
	agentMetrics    map[string]*pb.SystemMetrics
	agentSystemInfo map[string]*pb.SystemInfo

	// Configuration
	metricsInterval time.Duration
	sysInfoInterval time.Duration

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Status change listener
	statusListener *metricsStatusListener

	// SSE Handler for broadcasting
	sseHandler *SSEHandler
}

// metricsStatusListener listens for agent status changes
type metricsStatusListener struct {
	manager *StreamingManager
}

// OnStatusChange handles agent status changes
func (l *metricsStatusListener) OnStatusChange(event common.StatusChangeEvent) {
	switch event.NewStatus {
	case common.AgentStatusOnline:
		// Start polling when agent comes online
		go l.manager.startAgentPolling(event.AgentID)
	case common.AgentStatusOffline:
		// Clean up when agent goes offline
		l.manager.cleanupAgent(event.AgentID)
	}
}

// NewStreamingManager creates a new metrics streaming manager
func NewStreamingManager(handler *Handler, statusManager common.StatusManager, sseManager common.SSEManager) *StreamingManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &StreamingManager{
		handler:         handler,
		statusManager:   statusManager,
		sseManager:      sseManager,
		agentMetrics:    make(map[string]*pb.SystemMetrics),
		agentSystemInfo: make(map[string]*pb.SystemInfo),
		metricsInterval: 5 * time.Second,  // Poll metrics every 5 seconds
		sysInfoInterval: 60 * time.Second, // Poll system info every 60 seconds
		ctx:             ctx,
		cancel:          cancel,
		sseHandler:      nil, // Will be set by the SSE handler when it's created
	}

	manager.statusListener = &metricsStatusListener{manager: manager}

	return manager
}

// Start starts the streaming manager
func (m *StreamingManager) Start() {
	log.Println("Starting metrics streaming manager")

	// Listen for agent status changes
	m.statusManager.AddListener(m.statusListener)

	// Start polling for already online agents
	agents := m.statusManager.GetAllAgents()
	for _, agent := range agents {
		if agent.Status == common.AgentStatusOnline {
			go m.startAgentPolling(agent.AgentID)
		}
	}
}

// Stop stops the streaming manager
func (m *StreamingManager) Stop() {
	log.Println("Stopping metrics streaming manager")
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

// startAgentPolling starts continuous polling for a specific agent
func (m *StreamingManager) startAgentPolling(agentID string) {
	log.Printf("Starting metrics polling for agent %s", agentID)

	m.wg.Add(1)
	defer m.wg.Done()

	metricsTicker := time.NewTicker(m.metricsInterval)
	defer metricsTicker.Stop()

	sysInfoTicker := time.NewTicker(m.sysInfoInterval)
	defer sysInfoTicker.Stop()

	// Collect initial system info
	m.collectSystemInfo(agentID)

	// Collect initial metrics
	m.collectMetrics(agentID)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-metricsTicker.C:
			if !m.statusManager.IsAgentOnline(agentID) {
				log.Printf("Agent %s went offline, stopping metrics polling", agentID)
				return
			}
			m.collectMetrics(agentID)
		case <-sysInfoTicker.C:
			if !m.statusManager.IsAgentOnline(agentID) {
				log.Printf("Agent %s went offline, stopping system info polling", agentID)
				return
			}
			m.collectSystemInfo(agentID)
		}
	}
}

// collectMetrics collects metrics from an agent and broadcasts to clients
func (m *StreamingManager) collectMetrics(agentID string) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	metrics, err := m.handler.RequestMetrics(ctx, agentID)
	if err != nil {
		log.Printf("Error collecting metrics from agent %s: %v", agentID, err)
		// Broadcast error to clients
		m.broadcastMetricsError(agentID, err.Error())
		return
	}

	// Update cached metrics
	m.mu.Lock()
	m.agentMetrics[agentID] = metrics
	m.mu.Unlock()

	// Broadcast to interested clients
	m.broadcastMetrics(agentID, metrics)
}

// collectSystemInfo collects system info from an agent
func (m *StreamingManager) collectSystemInfo(agentID string) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	systemInfo, err := m.handler.RequestSystemInfo(ctx, agentID)
	if err != nil {
		log.Printf("Error collecting system info from agent %s: %v", agentID, err)
		return
	}

	// Update cached system info
	m.mu.Lock()
	m.agentSystemInfo[agentID] = systemInfo
	m.mu.Unlock()
}

// broadcastMetrics broadcasts metrics to all clients interested in this agent
func (m *StreamingManager) broadcastMetrics(agentID string, metrics *pb.SystemMetrics) {
	if m.sseHandler != nil {
		m.sseHandler.BroadcastMetrics(agentID, metrics)
	}
}

// broadcastMetricsError broadcasts an error to all clients interested in this agent
func (m *StreamingManager) broadcastMetricsError(agentID string, errorMsg string) {
	if m.sseHandler != nil {
		m.sseHandler.BroadcastMetricsError(agentID, errorMsg)
	}
}

// cleanupAgent removes cached data for an offline agent
func (m *StreamingManager) cleanupAgent(agentID string) {
	m.mu.Lock()
	delete(m.agentMetrics, agentID)
	delete(m.agentSystemInfo, agentID)
	m.mu.Unlock()

	// Broadcast offline status to clients
	if m.sseHandler != nil {
		m.sseHandler.BroadcastAgentOffline(agentID)
	}
}

// SetSSEHandler sets the SSE handler for broadcasting
func (m *StreamingManager) SetSSEHandler(handler *SSEHandler) {
	m.sseHandler = handler
}

// GetCachedMetrics returns cached metrics for an agent
func (m *StreamingManager) GetCachedMetrics(agentID string) (*pb.SystemMetrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, exists := m.agentMetrics[agentID]
	return metrics, exists
}

// GetCachedSystemInfo returns cached system info for an agent
func (m *StreamingManager) GetCachedSystemInfo(agentID string) (*pb.SystemInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	systemInfo, exists := m.agentSystemInfo[agentID]
	return systemInfo, exists
}

// SetMetricsInterval sets the metrics polling interval
func (m *StreamingManager) SetMetricsInterval(interval time.Duration) {
	m.metricsInterval = interval
}

// SetSystemInfoInterval sets the system info polling interval
func (m *StreamingManager) SetSystemInfoInterval(interval time.Duration) {
	m.sysInfoInterval = interval
}
