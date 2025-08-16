package metrics

import (
	"time"

	pb "github.com/mooncorn/nodelink/proto"
)

// MetricsMessageFormatter formats metrics messages for SSE transmission
type MetricsMessageFormatter struct{}

// NewMetricsMessageFormatter creates a new metrics message formatter
func NewMetricsMessageFormatter() *MetricsMessageFormatter {
	return &MetricsMessageFormatter{}
}

// FormatMetricsMessage formats metrics data for SSE
func (f *MetricsMessageFormatter) FormatMetricsMessage(agentID string, metrics *pb.SystemMetrics) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":  agentID,
		"metrics":   metrics,
		"timestamp": time.Now().Unix(),
	}
}

// FormatSystemInfoMessage formats system info data for SSE
func (f *MetricsMessageFormatter) FormatSystemInfoMessage(agentID string, systemInfo *pb.SystemInfo) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":    agentID,
		"system_info": systemInfo,
		"timestamp":   time.Now().Unix(),
	}
}

// FormatErrorMessage formats error messages for SSE
func (f *MetricsMessageFormatter) FormatErrorMessage(agentID, errorMsg string) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":  agentID,
		"error":     errorMsg,
		"timestamp": time.Now().Unix(),
	}
}

// FormatAgentOfflineMessage formats agent offline messages for SSE
func (f *MetricsMessageFormatter) FormatAgentOfflineMessage(agentID string) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":  agentID,
		"status":    "offline",
		"timestamp": time.Now().Unix(),
	}
}

// MetricsRooms manages room names for metrics SSE
type MetricsRooms struct{}

// NewMetricsRooms creates a new metrics rooms manager
func NewMetricsRooms() *MetricsRooms {
	return &MetricsRooms{}
}

// GetMetricsRoomName returns the room name for an agent's metrics
func (r *MetricsRooms) GetMetricsRoomName(agentID string) string {
	return "metrics-" + agentID
}
