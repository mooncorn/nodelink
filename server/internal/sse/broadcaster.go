package sse

import (
	"log"
	"time"

	"github.com/mooncorn/nodelink/server/internal/common"
	pb "github.com/mooncorn/nodelink/server/internal/proto"
)

// Broadcaster provides a centralized way to broadcast domain-specific events
type Broadcaster struct {
	sseManager common.SSEManager
}

// NewBroadcaster creates a new broadcaster instance
func NewBroadcaster(sseManager common.SSEManager) *Broadcaster {
	return &Broadcaster{
		sseManager: sseManager,
	}
}

// AgentStatus broadcasts agent status change events to standard rooms
func (b *Broadcaster) AgentStatus(event common.StatusChangeEvent) {
	// Broadcast to general agents room (for clients listening to all agents)
	if err := b.sseManager.SendToRoom("agents", event, "agent_status_change"); err != nil {
		log.Printf("Failed to broadcast agent status change to agents room: %v", err)
	}

	// Broadcast to agent-specific room (for clients listening to specific agent)
	agentRoom := "agent_" + event.AgentID
	if err := b.sseManager.SendToRoom(agentRoom, event, "status_change"); err != nil {
		log.Printf("Failed to broadcast to agent room %s: %v", agentRoom, err)
	}
}

// TerminalOutput broadcasts terminal output to the session-specific room
func (b *Broadcaster) TerminalOutput(sessionID string, output []byte) {
	room := "terminal_" + sessionID
	
	outputData := map[string]interface{}{
		"session_id": sessionID,
		"output":     string(output),
		"timestamp":  time.Now().Unix(),
	}

	if err := b.sseManager.SendToRoom(room, outputData, "terminal_output"); err != nil {
		log.Printf("Failed to broadcast terminal output to room %s: %v", room, err)
	}
}

// TerminalStatus broadcasts terminal session status changes
func (b *Broadcaster) TerminalStatus(sessionID, status, message string) {
	room := "terminal_" + sessionID
	
	statusData := map[string]interface{}{
		"session_id": sessionID,
		"status":     status,
		"message":    message,
		"timestamp":  time.Now().Unix(),
	}

	if err := b.sseManager.SendToRoom(room, statusData, "terminal_status"); err != nil {
		log.Printf("Failed to broadcast terminal status to room %s: %v", room, err)
	}
}

// Metrics broadcasts system metrics to the agent-specific metrics room
func (b *Broadcaster) Metrics(agentID string, metrics *pb.SystemMetrics) {
	room := "metrics_" + agentID
	
	metricsData := map[string]interface{}{
		"agent_id":  agentID,
		"metrics":   metrics,
		"timestamp": time.Now().Unix(),
	}

	if err := b.sseManager.SendToRoom(room, metricsData, "metrics"); err != nil {
		log.Printf("Failed to broadcast metrics to room %s: %v", room, err)
	}
}

// SystemInfo broadcasts system information to the agent-specific metrics room
func (b *Broadcaster) SystemInfo(agentID string, systemInfo *pb.SystemInfo) {
	room := "metrics_" + agentID
	
	systemInfoData := map[string]interface{}{
		"agent_id":    agentID,
		"system_info": systemInfo,
		"timestamp":   time.Now().Unix(),
	}

	if err := b.sseManager.SendToRoom(room, systemInfoData, "system_info"); err != nil {
		log.Printf("Failed to broadcast system info to room %s: %v", room, err)
	}
}

// MetricsError broadcasts metrics collection errors
func (b *Broadcaster) MetricsError(agentID, errorMsg string) {
	room := "metrics_" + agentID
	
	errorData := map[string]interface{}{
		"agent_id":  agentID,
		"error":     errorMsg,
		"timestamp": time.Now().Unix(),
	}

	if err := b.sseManager.SendToRoom(room, errorData, "metrics_error"); err != nil {
		log.Printf("Failed to broadcast metrics error to room %s: %v", room, err)
	}
}

// AgentOffline broadcasts agent offline status to metrics room
func (b *Broadcaster) AgentOffline(agentID string) {
	room := "metrics_" + agentID
	
	offlineData := map[string]interface{}{
		"agent_id":  agentID,
		"status":    "offline",
		"timestamp": time.Now().Unix(),
	}

	if err := b.sseManager.SendToRoom(room, offlineData, "agent_offline"); err != nil {
		log.Printf("Failed to broadcast agent offline to room %s: %v", room, err)
	}
}

// Custom broadcasts a custom event to a specific room
func (b *Broadcaster) Custom(room, eventType string, data interface{}) {
	if err := b.sseManager.SendToRoom(room, data, eventType); err != nil {
		log.Printf("Failed to broadcast custom event %s to room %s: %v", eventType, room, err)
	}
}

// Global broadcasts an event to all connected clients
func (b *Broadcaster) Global(eventType string, data interface{}) {
	if err := b.sseManager.Broadcast(data, eventType); err != nil {
		log.Printf("Failed to broadcast global event %s: %v", eventType, err)
	}
}