package status

import (
	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles SSE streaming for agent status updates using the new simplified architecture
type SSEHandler struct {
	manager       *Manager
	streamBuilder *sse.StreamBuilder
	broadcaster   *sse.Broadcaster
}

// NewSSEHandler creates a new simplified SSE handler for agent status updates
func NewSSEHandler(manager *Manager, sseManager common.SSEManager) *SSEHandler {
	handler := &SSEHandler{
		manager:       manager,
		streamBuilder: sse.NewStreamBuilder(sseManager),
		broadcaster:   sse.NewBroadcaster(sseManager),
	}

	// Register as a status change listener with the manager
	manager.AddListener(handler)

	return handler
}

// RegisterRoutes registers SSE routes for agent status updates
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/agents/events", h.handleAllAgentStatusEvents)
	router.GET("/agents/:agentId/events", h.handleSpecificAgentStatusEvents)
}

// handleAllAgentStatusEvents handles SSE connections for all agent status events
func (h *SSEHandler) handleAllAgentStatusEvents(c *gin.Context) {
	h.streamBuilder.ForAgent("").AllAgents().Handle(c)
}

// handleSpecificAgentStatusEvents handles SSE connections for specific agent status events
func (h *SSEHandler) handleSpecificAgentStatusEvents(c *gin.Context) {
	h.streamBuilder.ForAgent("").WithCurrentStatus(h.manager).Handle(c)
}

// OnStatusChange implements StatusChangeListener interface
func (h *SSEHandler) OnStatusChange(event common.StatusChangeEvent) {
	h.broadcaster.AgentStatus(event)
}

// Stop gracefully stops the SSE handler
func (h *SSEHandler) Stop() {
	// The broadcaster and stream builder handle cleanup internally
}