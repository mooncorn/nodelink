package status

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// HTTPHandler handles HTTP requests for agent status management
type HTTPHandler struct {
	manager *Manager
}

// NewHTTPHandler creates a new HTTP handler for agent status
func NewHTTPHandler(manager *Manager) *HTTPHandler {
	return &HTTPHandler{
		manager: manager,
	}
}

// RegisterRoutes registers status routes with the given router
func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	// Get all agents with optional status filter
	router.GET("/agents", h.GetAgents)

	// Get specific agent by ID
	router.GET("/agents/:agentId", h.GetAgent)
}

// GetAgentsResponse represents the response for getting all agents
type GetAgentsResponse struct {
	Agents []common.AgentInfo `json:"agents"`
	Stats  map[string]int     `json:"stats"`
	Total  int                `json:"total"`
}

// GetAgent handles GET /agents/:agentId
func (h *HTTPHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("agentId")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "agent_id is required",
		})
		return
	}

	agent, exists := h.manager.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "agent not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent": agent,
	})
}

// GetAgents handles GET /agents with optional status filter
func (h *HTTPHandler) GetAgents(c *gin.Context) {
	// Get status filter from query parameter
	statusFilter := c.Query("status")

	var agents []*common.AgentInfo
	if statusFilter != "" {
		// Filter by status
		allAgents := h.manager.GetAllAgents()
		for _, agent := range allAgents {
			if strings.EqualFold(string(agent.Status), statusFilter) {
				agents = append(agents, agent)
			}
		}
	} else {
		// Get all agents
		agents = h.manager.GetAllAgents()
	}

	// Calculate statistics
	stats := make(map[string]int)
	stats["total"] = len(agents)
	stats["online"] = 0
	stats["offline"] = 0

	for _, agent := range agents {
		switch agent.Status {
		case common.AgentStatusOnline:
			stats["online"]++
		case common.AgentStatusOffline:
			stats["offline"]++
		}
	}

	response := GetAgentsResponse{
		Agents: make([]common.AgentInfo, len(agents)),
		Stats:  stats,
		Total:  len(agents),
	}

	// Convert pointers to values for response
	for i, agent := range agents {
		response.Agents[i] = *agent
	}

	c.JSON(http.StatusOK, response)
}
