package agent

import (
	"sync"
	"time"
)

type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusUnknown      ConnectionStatus = "unknown"
)

// AgentInfo represents comprehensive information about an agent
type AgentInfo struct {
	AgentID        string            `json:"agent_id"`
	Status         ConnectionStatus  `json:"status"`
	LastSeen       time.Time         `json:"last_seen"`
	ConnectedAt    *time.Time        `json:"connected_at,omitempty"`
	DisconnectedAt *time.Time        `json:"disconnected_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ConnectionAddr string            `json:"connection_addr,omitempty"`
	Version        string            `json:"version,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// StatusChangeEvent represents a status change notification
type StatusChangeEvent struct {
	AgentID   string           `json:"agent_id"`
	OldStatus ConnectionStatus `json:"old_status"`
	NewStatus ConnectionStatus `json:"new_status"`
	Timestamp time.Time        `json:"timestamp"`
	Agent     *AgentInfo       `json:"agent"`
}

// StatusChangeListener defines the interface for status change notifications
type StatusChangeListener interface {
	OnStatusChange(event StatusChangeEvent)
}

// Repository manages agent information and status tracking with listener notifications
type Repository struct {
	mu        sync.RWMutex
	agents    map[string]*AgentInfo
	listeners []StatusChangeListener
}

// NewRepository creates a new agent repository
func NewRepository() *Repository {
	repo := &Repository{
		agents:    make(map[string]*AgentInfo),
		listeners: make([]StatusChangeListener, 0),
	}

	return repo
}

// AddListener adds a status change listener
func (r *Repository) AddListener(listener StatusChangeListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners = append(r.listeners, listener)
}

// notifyStatusChange notifies all listeners and callbacks about status changes
func (r *Repository) notifyStatusChange(agentID string, oldStatus, newStatus ConnectionStatus, agent *AgentInfo) {
	if oldStatus == newStatus {
		return
	}

	event := StatusChangeEvent{
		AgentID:   agentID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: time.Now(),
		Agent:     agent,
	}

	// Notify listeners
	for _, listener := range r.listeners {
		go listener.OnStatusChange(event)
	}
}

// RegisterAgent registers a new agent with comprehensive information
func (r *Repository) RegisterAgent(agentID string, metadata ...map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Don't overwrite existing agent if it already exists
	if _, exists := r.agents[agentID]; exists {
		return
	}

	now := time.Now()
	agent := &AgentInfo{
		AgentID:   agentID,
		Status:    StatusUnknown,
		LastSeen:  now,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}

	// Add metadata if provided
	if len(metadata) > 0 && metadata[0] != nil {
		for k, v := range metadata[0] {
			agent.Metadata[k] = v
		}
	}

	r.agents[agentID] = agent
}

// UnregisterAgent removes an agent from the repository
func (r *Repository) UnregisterAgent(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.agents, agentID)
}

// UpdateStatus updates the connection status of an agent
func (r *Repository) UpdateStatus(agentID string, status ConnectionStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		// Auto-register if agent doesn't exist
		now := time.Now()
		agent = &AgentInfo{
			AgentID:   agentID,
			Status:    StatusUnknown,
			LastSeen:  now,
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]string),
		}
		r.agents[agentID] = agent
	}

	oldStatus := agent.Status
	now := time.Now()

	// Update status and timestamps
	agent.Status = status
	agent.LastSeen = now
	agent.UpdatedAt = now

	// Track connection/disconnection times
	switch status {
	case StatusConnected:
		if oldStatus != StatusConnected {
			agent.ConnectedAt = &now
		}
	case StatusDisconnected:
		if oldStatus == StatusConnected {
			agent.DisconnectedAt = &now
		}
	}

	// Create a copy for the notification to avoid holding the lock
	agentCopy := r.copyAgentInfo(agent)

	// Release the lock before notifying to avoid deadlocks
	r.mu.Unlock()

	// Notify listeners about status change
	r.notifyStatusChange(agentID, oldStatus, status, agentCopy)

	// Re-acquire lock for defer unlock
	r.mu.Lock()
}

// UpdateLastSeen updates the last seen time for an agent
func (r *Repository) UpdateLastSeen(agentID string, lastSeen time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		// Auto-register if agent doesn't exist
		now := time.Now()
		agent = &AgentInfo{
			AgentID:   agentID,
			Status:    StatusUnknown,
			LastSeen:  lastSeen,
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]string),
		}
		r.agents[agentID] = agent
		return
	}

	agent.UpdatedAt = time.Now()
	agent.LastSeen = lastSeen
}

// UpdateMetadata updates the metadata for an agent
func (r *Repository) UpdateMetadata(agentID string, metadata map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return // Don't auto-register for metadata updates
	}

	if agent.Metadata == nil {
		agent.Metadata = make(map[string]string)
	}

	for k, v := range metadata {
		agent.Metadata[k] = v
	}
	agent.UpdatedAt = time.Now()
}

// copyAgentInfo creates a deep copy of agent info to avoid race conditions
func (r *Repository) copyAgentInfo(agent *AgentInfo) *AgentInfo {
	result := &AgentInfo{
		AgentID:        agent.AgentID,
		Status:         agent.Status,
		LastSeen:       agent.LastSeen,
		ConnectionAddr: agent.ConnectionAddr,
		Version:        agent.Version,
		CreatedAt:      agent.CreatedAt,
		UpdatedAt:      agent.UpdatedAt,
		Metadata:       make(map[string]string),
	}

	if agent.ConnectedAt != nil {
		connectedAt := *agent.ConnectedAt
		result.ConnectedAt = &connectedAt
	}

	if agent.DisconnectedAt != nil {
		disconnectedAt := *agent.DisconnectedAt
		result.DisconnectedAt = &disconnectedAt
	}

	if agent.Metadata != nil {
		for k, v := range agent.Metadata {
			result.Metadata[k] = v
		}
	}

	return result
}

// MarkConnected marks an agent as connected
func (r *Repository) MarkConnected(agentID string) {
	r.UpdateStatus(agentID, StatusConnected)
}

// MarkDisconnected marks an agent as disconnected
func (r *Repository) MarkDisconnected(agentID string) {
	r.UpdateStatus(agentID, StatusDisconnected)
}

// GetStatus returns the connection status for a specific agent
func (r *Repository) GetStatus(agentID string) (ConnectionStatus, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return StatusUnknown, false
	}

	return agent.Status, true
}

// GetAgent returns the full agent information for a specific agent
func (r *Repository) GetAgent(agentID string) (*AgentInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return r.copyAgentInfo(agent), true
}

// GetConnection returns the full connection information for a specific agent (backward compatibility)
func (r *Repository) GetConnection(agentID string) (*AgentInfo, bool) {
	return r.GetAgent(agentID)
}

// GetAllAgents returns agent information for all agents
func (r *Repository) GetAllAgents() map[string]*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*AgentInfo, len(r.agents))
	for agentID, agent := range r.agents {
		result[agentID] = r.copyAgentInfo(agent)
	}

	return result
}

// GetAllConnections returns connection information for all agents (backward compatibility)
func (r *Repository) GetAllConnections() map[string]*AgentInfo {
	agents := r.GetAllAgents()
	result := make(map[string]*AgentInfo, len(agents))
	for agentID, agent := range agents {
		result[agentID] = agent // AgentConnection is an alias for AgentInfo
	}
	return result
}

// GetConnectedAgents returns a list of all connected agent IDs
func (r *Repository) GetConnectedAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var connected []string
	for agentID, agent := range r.agents {
		if agent.Status == StatusConnected {
			connected = append(connected, agentID)
		}
	}

	return connected
}

// GetDisconnectedAgents returns a list of all disconnected agent IDs
func (r *Repository) GetDisconnectedAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var disconnected []string
	for agentID, agent := range r.agents {
		if agent.Status == StatusDisconnected {
			disconnected = append(disconnected, agentID)
		}
	}

	return disconnected
}

// IsConnected checks if an agent is currently connected
func (r *Repository) IsConnected(agentID string) bool {
	status, exists := r.GetStatus(agentID)
	return exists && status == StatusConnected
}

// GetAgentCount returns the total number of registered agents
func (r *Repository) GetAgentCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.agents)
}

// GetConnectionStats returns statistics about agent connections
func (r *Repository) GetConnectionStats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]int{
		"total":        0,
		"connected":    0,
		"disconnected": 0,
		"unknown":      0,
	}

	for _, agent := range r.agents {
		stats["total"]++
		switch agent.Status {
		case StatusConnected:
			stats["connected"]++
		case StatusDisconnected:
			stats["disconnected"]++
		case StatusUnknown:
			stats["unknown"]++
		}
	}

	return stats
}

// GetAgentsByMetadata returns agents that match the given metadata criteria
func (r *Repository) GetAgentsByMetadata(criteria map[string]string) []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*AgentInfo
	for _, agent := range r.agents {
		if r.matchesMetadata(agent, criteria) {
			result = append(result, r.copyAgentInfo(agent))
		}
	}

	return result
}

// GetAgentsByStatus returns agents with the specified status
func (r *Repository) GetAgentsByStatus(status ConnectionStatus) []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*AgentInfo
	for _, agent := range r.agents {
		if agent.Status == status {
			result = append(result, r.copyAgentInfo(agent))
		}
	}

	return result
}

// AgentExists checks if an agent with the given ID exists in the repository
func (r *Repository) AgentExists(agentID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.agents[agentID]
	return exists
}

// matchesMetadata checks if an agent's metadata matches the given criteria
func (r *Repository) matchesMetadata(agent *AgentInfo, criteria map[string]string) bool {
	if agent.Metadata == nil {
		return len(criteria) == 0
	}

	for key, expectedValue := range criteria {
		if actualValue, exists := agent.Metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}
