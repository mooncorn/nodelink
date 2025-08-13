package agent

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HTTPHandler handles HTTP requests for agent management
type HTTPHandler struct {
	repo *Repository
}

// NewHTTPHandler creates a new HTTP handler for agents
func NewHTTPHandler(repo *Repository) *HTTPHandler {
	return &HTTPHandler{
		repo: repo,
	}
}

// RegisterRoutes registers agent routes with the given router
func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	// Get all agents with optional status filter
	router.GET("/agents", h.GetAgents)

	// Get specific agent by ID
	router.GET("/agents/:agentId", h.GetAgent)
}

// GetAgentsResponse represents the response for getting all agents
type GetAgentsResponse struct {
	Agents []AgentInfo    `json:"agents"`
	Stats  map[string]int `json:"stats"`
	Total  int            `json:"total"`
}

// GetAgents handles GET /agents with optional status filter
func (h *HTTPHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("agentId")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "agent_id is required",
		})
		return
	}

	agent, exists := h.repo.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "agent not found",
		})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// GetAgents handles GET /agents with optional status filter
func (h *HTTPHandler) GetAgents(c *gin.Context) {
	statusFilter := c.Query("status")

	var agents []AgentInfo

	if statusFilter != "" {
		// Validate status filter
		status := ConnectionStatus(strings.ToLower(statusFilter))
		switch status {
		case StatusConnected, StatusDisconnected, StatusUnknown:
			agentPtrs := h.repo.GetAgentsByStatus(status)
			agents = make([]AgentInfo, 0, len(agentPtrs))
			for _, agent := range agentPtrs {
				agents = append(agents, *agent)
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid status filter. Must be one of: connected, disconnected, unknown",
			})
			return
		}
	} else {
		// Get all agents
		allAgents := h.repo.GetAllAgents()
		agents = make([]AgentInfo, 0, len(allAgents))
		for _, agent := range allAgents {
			agents = append(agents, *agent)
		}
	}

	// Get stats
	stats := h.repo.GetConnectionStats()

	response := GetAgentsResponse{
		Agents: agents,
		Stats:  stats,
		Total:  len(agents),
	}

	c.JSON(http.StatusOK, response)
}
