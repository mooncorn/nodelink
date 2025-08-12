package agent

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// // HTTPHandler handles HTTP requests for connection operations
// type HTTPHandler struct {
// 	service *Service
// }

// // NewHTTPHandler creates a new HTTP handler for connection operations
// func NewHTTPHandler(service *Service) *HTTPHandler {
// 	return &HTTPHandler{
// 		service: service,
// 	}
// }

// // ConnectionResponse represents the response format for agent connection data
// type ConnectionResponse struct {
// 	AgentID        string `json:"agent_id"`
// 	Status         string `json:"status"`
// 	LastSeen       string `json:"last_seen"`
// 	ConnectedAt    string `json:"connected_at,omitempty"`
// 	DisconnectedAt string `json:"disconnected_at,omitempty"`
// }

// // ConnectionStatsResponse represents the response format for connection statistics
// type ConnectionStatsResponse struct {
// 	Total        int `json:"total"`
// 	Connected    int `json:"connected"`
// 	Disconnected int `json:"disconnected"`
// 	Unknown      int `json:"unknown"`
// }

// // GetAgentConnection handles GET /connections/agents/:agent_id
// func (h *HTTPHandler) GetAgentConnection(c *gin.Context) {
// 	agentID := c.Param("agent_id")

// 	connection, exists := h.service.GetConnection(agentID)
// 	if !exists {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
// 		return
// 	}

// 	response := ConnectionResponse{
// 		AgentID:  connection.AgentID,
// 		Status:   string(connection.Status),
// 		LastSeen: connection.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
// 	}

// 	if connection.ConnectedAt != nil {
// 		response.ConnectedAt = connection.ConnectedAt.Format("2006-01-02T15:04:05Z07:00")
// 	}

// 	if connection.DisconnectedAt != nil {
// 		response.DisconnectedAt = connection.DisconnectedAt.Format("2006-01-02T15:04:05Z07:00")
// 	}

// 	c.JSON(http.StatusOK, response)
// }

// // GetAllConnections handles GET /connections/agents
// func (h *HTTPHandler) GetAllConnections(c *gin.Context) {
// 	connections := h.service.GetAllConnections()

// 	responses := make([]ConnectionResponse, 0, len(connections))
// 	for _, connection := range connections {
// 		response := ConnectionResponse{
// 			AgentID:  connection.AgentID,
// 			Status:   string(connection.Status),
// 			LastSeen: connection.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
// 		}

// 		if connection.ConnectedAt != nil {
// 			response.ConnectedAt = connection.ConnectedAt.Format("2006-01-02T15:04:05Z07:00")
// 		}

// 		if connection.DisconnectedAt != nil {
// 			response.DisconnectedAt = connection.DisconnectedAt.Format("2006-01-02T15:04:05Z07:00")
// 		}

// 		responses = append(responses, response)
// 	}

// 	c.JSON(http.StatusOK, gin.H{"connections": responses})
// }

// // GetConnectedAgents handles GET /connections/agents/connected
// func (h *HTTPHandler) GetConnectedAgents(c *gin.Context) {
// 	agents := h.service.GetConnectedAgents()
// 	c.JSON(http.StatusOK, gin.H{"connected_agents": agents})
// }

// // GetDisconnectedAgents handles GET /connections/agents/disconnected
// func (h *HTTPHandler) GetDisconnectedAgents(c *gin.Context) {
// 	agents := h.service.GetDisconnectedAgents()
// 	c.JSON(http.StatusOK, gin.H{"disconnected_agents": agents})
// }

// // GetConnectionStats handles GET /connections/stats
// func (h *HTTPHandler) GetConnectionStats(c *gin.Context) {
// 	stats := h.service.GetConnectionStats()

// 	response := ConnectionStatsResponse{
// 		Total:        stats["total"],
// 		Connected:    stats["connected"],
// 		Disconnected: stats["disconnected"],
// 		Unknown:      stats["unknown"],
// 	}

// 	c.JSON(http.StatusOK, response)
// }

// // UpdateAgentStatus handles PUT /connections/agents/:agent_id/status
// func (h *HTTPHandler) UpdateAgentStatus(c *gin.Context) {
// 	agentID := c.Param("agent_id")

// 	var req struct {
// 		Status string `json:"status" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
// 		return
// 	}

// 	// Validate status
// 	var status ConnectionStatus
// 	switch req.Status {
// 	case "connected":
// 		status = StatusConnected
// 	case "disconnected":
// 		status = StatusDisconnected
// 	case "unknown":
// 		status = StatusUnknown
// 	default:
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status. Must be one of: connected, disconnected, unknown"})
// 		return
// 	}

// 	h.service.UpdateStatus(agentID, status)

// 	c.JSON(http.StatusOK, gin.H{
// 		"agent_id": agentID,
// 		"status":   req.Status,
// 		"message":  "status updated successfully",
// 	})
// }

// // RegisterAgent handles POST /connections/agents
// func (h *HTTPHandler) RegisterAgent(c *gin.Context) {
// 	var req struct {
// 		AgentID string `json:"agent_id" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
// 		return
// 	}

// 	h.service.RegisterAgent(req.AgentID)

// 	c.JSON(http.StatusCreated, gin.H{
// 		"agent_id": req.AgentID,
// 		"message":  "agent registered successfully",
// 	})
// }

// // UnregisterAgent handles DELETE /connections/agents/:agent_id
// func (h *HTTPHandler) UnregisterAgent(c *gin.Context) {
// 	agentID := c.Param("agent_id")

// 	h.service.UnregisterAgent(agentID)

// 	c.JSON(http.StatusOK, gin.H{
// 		"agent_id": agentID,
// 		"message":  "agent unregistered successfully",
// 	})
// }

// // RegisterRoutes registers HTTP routes for connection operations
// func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
// 	connections := router.Group("/connections")
// 	{
// 		agents := connections.Group("/agents")
// 		{
// 			agents.GET("", h.GetAllConnections)
// 			agents.POST("", h.RegisterAgent)
// 			agents.GET("/connected", h.GetConnectedAgents)
// 			agents.GET("/disconnected", h.GetDisconnectedAgents)
// 			agents.GET("/:agent_id", h.GetAgentConnection)
// 			agents.PUT("/:agent_id/status", h.UpdateAgentStatus)
// 			agents.DELETE("/:agent_id", h.UnregisterAgent)
// 		}
// 		connections.GET("/stats", h.GetConnectionStats)
// 	}
// }
