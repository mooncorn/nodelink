package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
	pb "github.com/mooncorn/nodelink/server/internal/proto"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles metrics SSE endpoints using the new simplified architecture
type SSEHandler struct {
	handler          *Handler
	streamingManager *StreamingManager
	statusManager    common.StatusManager
	streamBuilder    *sse.StreamBuilder
	broadcaster      *sse.Broadcaster
}

// NewSSEHandler creates a new simplified SSE handler for metrics streaming
func NewSSEHandler(handler *Handler, streamingManager *StreamingManager, sseManager common.SSEManager) *SSEHandler {
	sseHandler := &SSEHandler{
		handler:          handler,
		streamingManager: streamingManager,
		statusManager:    handler.statusManager,
		streamBuilder:    sse.NewStreamBuilder(sseManager),
		broadcaster:      sse.NewBroadcaster(sseManager),
	}

	// Connect the streaming manager to this SSE handler for broadcasting
	streamingManager.SetSSEHandler(sseHandler)

	return sseHandler
}

// RegisterRoutes registers SSE routes for metrics streaming and HTTP endpoints
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/metrics/:agentID/stream", h.handleMetricsStream)
}

// handleMetricsStream handles SSE connections for metrics streaming (metrics only, no system info)
func (h *SSEHandler) handleMetricsStream(c *gin.Context) {
	agentID := c.Param("agentID")

	// Get cached metrics only (no system info in stream)
	cachedMetrics, hasMetrics := h.streamingManager.GetCachedMetrics(agentID)

	var metrics any
	if hasMetrics {
		// Format cached metrics for initial message
		metrics = map[string]any{
			"agent_id":  agentID,
			"metrics":   cachedMetrics,
			"timestamp": h.getCurrentTimestamp(),
		}
	}

	h.streamBuilder.ForMetrics(agentID).
		WithStatusManager(h.statusManager).
		WithCachedData(metrics).
		Handle(c)
}

// BroadcastMetrics broadcasts system metrics to the agent's metrics room
func (h *SSEHandler) BroadcastMetrics(agentID string, metrics *pb.SystemMetrics) {
	h.broadcaster.Metrics(agentID, metrics)
}

// BroadcastMetricsError broadcasts metrics collection errors
func (h *SSEHandler) BroadcastMetricsError(agentID, errorMsg string) {
	h.broadcaster.MetricsError(agentID, errorMsg)
}

// BroadcastAgentOffline broadcasts agent offline status
func (h *SSEHandler) BroadcastAgentOffline(agentID string) {
	h.broadcaster.AgentOffline(agentID)
}

// getCurrentTimestamp is a helper to get current timestamp
func (h *SSEHandler) getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
