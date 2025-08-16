package status

import (
	"sync"
	"time"

	"github.com/mooncorn/nodelink/server/internal/common"
)

// Manager manages agent status and provides a centralized status tracking system
type Manager struct {
	mu        sync.RWMutex
	agents    map[string]*common.AgentInfo
	listeners []common.StatusChangeListener
}

// NewManager creates a new status manager
func NewManager() *Manager {
	return &Manager{
		agents:    make(map[string]*common.AgentInfo),
		listeners: make([]common.StatusChangeListener, 0),
	}
}

// RegisterAgent registers a new agent with the status manager
func (m *Manager) RegisterAgent(agentID string, metadata map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Don't overwrite existing agent if it already exists
	if _, exists := m.agents[agentID]; exists {
		return
	}

	agent := &common.AgentInfo{
		AgentID:   agentID,
		Status:    common.AgentStatusOffline,
		LastSeen:  now,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.agents[agentID] = agent
}

// SetAgentOnline marks an agent as online
func (m *Manager) SetAgentOnline(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		// Auto-register if agent doesn't exist
		now := time.Now()
		agent = &common.AgentInfo{
			AgentID:   agentID,
			Status:    common.AgentStatusOffline,
			LastSeen:  now,
			CreatedAt: now,
			UpdatedAt: now,
		}
		m.agents[agentID] = agent
	}

	oldStatus := agent.Status
	now := time.Now()

	agent.Status = common.AgentStatusOnline
	agent.LastSeen = now
	agent.UpdatedAt = now

	if agent.ConnectedAt == nil {
		agent.ConnectedAt = &now
	}

	// Notify listeners if status changed
	if oldStatus != common.AgentStatusOnline {
		event := common.StatusChangeEvent{
			AgentID:   agentID,
			OldStatus: oldStatus,
			NewStatus: common.AgentStatusOnline,
			Timestamp: now,
			Agent:     agent,
		}
		m.notifyStatusChange(event)
	}
}

// SetAgentOffline marks an agent as offline
func (m *Manager) SetAgentOffline(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists || agent.Status == common.AgentStatusOffline {
		return
	}

	oldStatus := agent.Status
	now := time.Now()

	agent.Status = common.AgentStatusOffline
	agent.UpdatedAt = now

	event := common.StatusChangeEvent{
		AgentID:   agentID,
		OldStatus: oldStatus,
		NewStatus: common.AgentStatusOffline,
		Timestamp: now,
		Agent:     agent,
	}
	m.notifyStatusChange(event)
}

// UpdateLastSeen updates the last seen timestamp for an agent
func (m *Manager) UpdateLastSeen(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent, exists := m.agents[agentID]; exists {
		agent.LastSeen = time.Now()
		agent.UpdatedAt = time.Now()
	}
}

// IsAgentOnline checks if an agent is currently online
func (m *Manager) IsAgentOnline(agentID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	return exists && agent.Status == common.AgentStatusOnline
}

// GetAgent returns information about a specific agent
func (m *Manager) GetAgent(agentID string) (*common.AgentInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	agentCopy := *agent
	return &agentCopy, true
}

// GetAllAgents returns information about all agents
func (m *Manager) GetAllAgents() []*common.AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*common.AgentInfo, 0, len(m.agents))
	for _, agent := range m.agents {
		agentCopy := *agent
		agents = append(agents, &agentCopy)
	}
	return agents
}

// GetOnlineAgents returns all currently online agents
func (m *Manager) GetOnlineAgents() []*common.AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var onlineAgents []*common.AgentInfo
	for _, agent := range m.agents {
		if agent.Status == common.AgentStatusOnline {
			agentCopy := *agent
			onlineAgents = append(onlineAgents, &agentCopy)
		}
	}
	return onlineAgents
}

// AddListener adds a status change listener
func (m *Manager) AddListener(listener common.StatusChangeListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// CleanupStaleAgents removes agents that haven't been seen since the cutoff time
func (m *Manager) CleanupStaleAgents(cutoff time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var toDelete []string
	for agentID, agent := range m.agents {
		if agent.LastSeen.Before(cutoff) && agent.Status == common.AgentStatusOffline {
			toDelete = append(toDelete, agentID)
		}
	}

	for _, agentID := range toDelete {
		delete(m.agents, agentID)
	}

	return len(toDelete)
}

// notifyStatusChange notifies all listeners about a status change
func (m *Manager) notifyStatusChange(event common.StatusChangeEvent) {
	for _, listener := range m.listeners {
		go listener.OnStatusChange(event)
	}
}
