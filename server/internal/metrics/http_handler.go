package metrics

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// HTTPHandler handles HTTP requests for metrics
type HTTPHandler struct {
	handler          *Handler
	sseManager       common.SSEManager
	streamingManager *StreamingManager
}

// NewHTTPHandler creates a new HTTP handler for metrics
func NewHTTPHandler(handler *Handler, sseManager common.SSEManager, streamingManager *StreamingManager) *HTTPHandler {
	return &HTTPHandler{
		handler:          handler,
		sseManager:       sseManager,
		streamingManager: streamingManager,
	}
}

// RegisterRoutes registers the metrics routes
func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/metrics/:agentID", h.getSystemInfo)
}

// getSystemInfo handles GET /metrics/:agentID
func (h *HTTPHandler) getSystemInfo(c *gin.Context) {
	agentID := c.Param("agentID")

	// Check if agent exists and is online
	if !h.handler.statusManager.IsAgentOnline(agentID) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Agent not found or offline",
		})
		return
	}

	// Try to get cached system info first
	systemInfo, exists := h.streamingManager.GetCachedSystemInfo(agentID)
	if !exists {
		// If not cached, request it directly
		var err error
		systemInfo, err = h.handler.RequestSystemInfo(c.Request.Context(), agentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":    agentID,
		"system_info": systemInfo,
		"timestamp":   time.Now().Unix(),
	})
}
