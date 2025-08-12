package heartbeat

// import (
// 	"net/http"
// 	"time"

// 	"github.com/gin-gonic/gin"
// )

// // HTTPHandler handles HTTP requests for heartbeat operations
// type HTTPHandler struct {
// 	server *HeartbeatServer
// }

// // NewHTTPHandler creates a new HTTP handler for heartbeat operations
// func NewHTTPHandler(server *HeartbeatServer) *HTTPHandler {
// 	return &HTTPHandler{
// 		server: server,
// 	}
// }

// // AgentStatusResponse represents the HTTP response for agent status
// type AgentStatusResponse struct {
// 	AgentID       string    `json:"agent_id"`
// 	LastHeartbeat time.Time `json:"last_heartbeat"`
// 	Connected     bool      `json:"connected"`
// }

// // HealthCheckResponse represents the HTTP response for health check
// type HealthCheckResponse struct {
// 	TotalAgents     int                            `json:"total_agents"`
// 	ConnectedAgents int                            `json:"connected_agents"`
// 	Agents          map[string]AgentStatusResponse `json:"agents"`
// }

// // GetAgentStatus handles GET /heartbeat/agents/:agent_id
// func (h *HTTPHandler) GetAgentStatus(c *gin.Context) {
// 	agentID := c.Param("agent_id")
// 	if agentID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
// 		return
// 	}

// 	status, exists := h.server.GetAgentStatus(agentID)
// 	if !exists {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
// 		return
// 	}

// 	response := AgentStatusResponse{
// 		AgentID:       status.AgentID,
// 		LastHeartbeat: status.LastHeartbeat,
// 		Connected:     h.server.IsAgentConnected(agentID),
// 	}

// 	c.JSON(http.StatusOK, response)
// }

// // GetAllAgents handles GET /heartbeat/agents
// func (h *HTTPHandler) GetAllAgents(c *gin.Context) {
// 	allAgents := h.server.GetAllAgents()

// 	agents := make(map[string]AgentStatusResponse, len(allAgents))
// 	connectedCount := 0

// 	for agentID, status := range allAgents {
// 		connected := h.server.IsAgentConnected(agentID)
// 		if connected {
// 			connectedCount++
// 		}

// 		agents[agentID] = AgentStatusResponse{
// 			AgentID:       status.AgentID,
// 			LastHeartbeat: status.LastHeartbeat,
// 			Connected:     connected,
// 		}
// 	}

// 	response := HealthCheckResponse{
// 		TotalAgents:     len(allAgents),
// 		ConnectedAgents: connectedCount,
// 		Agents:          agents,
// 	}

// 	c.JSON(http.StatusOK, response)
// }

// // GetConnectedAgents handles GET /heartbeat/agents/connected
// func (h *HTTPHandler) GetConnectedAgents(c *gin.Context) {
// 	connectedAgents := h.server.GetConnectedAgents()
// 	c.JSON(http.StatusOK, gin.H{"connected_agents": connectedAgents})
// }

// // GetDisconnectedAgents handles GET /heartbeat/agents/disconnected
// func (h *HTTPHandler) GetDisconnectedAgents(c *gin.Context) {
// 	allAgents := h.server.GetAllAgents()
// 	connectedAgents := h.server.GetConnectedAgents()

// 	// Create a set of connected agents for fast lookup
// 	connectedSet := make(map[string]bool)
// 	for _, agentID := range connectedAgents {
// 		connectedSet[agentID] = true
// 	}

// 	// Find disconnected agents
// 	disconnectedAgents := make([]string, 0)
// 	for agentID := range allAgents {
// 		if !connectedSet[agentID] {
// 			disconnectedAgents = append(disconnectedAgents, agentID)
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{"disconnected_agents": disconnectedAgents})
// }

// // RegisterRoutes registers HTTP routes for heartbeat operations
// func (h *HTTPHandler) RegisterRoutes(router *gin.RouterGroup) {
// 	heartbeat := router.Group("/heartbeat")
// 	{
// 		agents := heartbeat.Group("/agents")
// 		{
// 			agents.GET("", h.GetAllAgents)
// 			agents.GET("/connected", h.GetConnectedAgents)
// 			agents.GET("/disconnected", h.GetDisconnectedAgents)
// 			agents.GET("/:agent_id", h.GetAgentStatus)
// 		}
// 	}
// }
