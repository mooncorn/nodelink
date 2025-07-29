package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/tasks"
)

// HTTPHandler handles HTTP requests for metrics
type HTTPHandler struct {
	store       *MetricsStore
	taskManager *tasks.TaskManager
}

// NewHTTPHandler creates a new HTTP handler for metrics
func NewHTTPHandler(store *MetricsStore, taskManager *tasks.TaskManager) *HTTPHandler {
	return &HTTPHandler{
		store:       store,
		taskManager: taskManager,
	}
}

// RegisterRoutes registers metrics routes with the given router
func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	// Agent system information
	router.GET("/agents/:agentId/system", h.GetAgentSystem)

	// Current metrics
	router.GET("/agents/:agentId/metrics", h.GetAgentMetrics)

	// Historical metrics
	router.GET("/agents/:agentId/metrics/history", h.GetAgentMetricsHistory)

	// All agents summary
	router.GET("/metrics/agents", h.GetAllAgents)

	// Control metrics streaming
	router.POST("/agents/:agentId/metrics/start", h.StartMetricsStreaming)
	router.POST("/agents/:agentId/metrics/stop", h.StopMetricsStreaming)

	// Request system info
	router.POST("/agents/:agentId/system/refresh", h.RefreshSystemInfo)
}

// GetAgentSystem returns system information for an agent
func (h *HTTPHandler) GetAgentSystem(c *gin.Context) {
	agentID := c.Param("agentId")

	systemInfo, err := h.store.GetSystemInfo(agentID)
	if err != nil {
		if err == ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "no system information available"})
		}
		return
	}

	c.JSON(http.StatusOK, systemInfo)
}

// GetAgentMetrics returns current metrics for an agent
func (h *HTTPHandler) GetAgentMetrics(c *gin.Context) {
	agentID := c.Param("agentId")

	metrics, err := h.store.GetCurrentMetrics(agentID)
	if err != nil {
		if err == ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "no metrics available"})
		}
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetAgentMetricsHistory returns historical metrics for an agent
func (h *HTTPHandler) GetAgentMetricsHistory(c *gin.Context) {
	agentID := c.Param("agentId")

	// Parse query parameters
	startStr := c.DefaultQuery("start", "0")
	endStr := c.DefaultQuery("end", strconv.FormatInt(time.Now().Unix(), 10))
	maxPointsStr := c.DefaultQuery("max_points", "1000")
	metricsStr := c.DefaultQuery("metrics", "")

	start, err := strconv.ParseUint(startStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start timestamp"})
		return
	}

	end, err := strconv.ParseUint(endStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end timestamp"})
		return
	}

	maxPoints, err := strconv.ParseUint(maxPointsStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid max_points"})
		return
	}

	var metrics []string
	if metricsStr != "" {
		metrics = strings.Split(metricsStr, ",")
	}

	history, err := h.store.GetHistoricalMetrics(agentID, start, end, uint32(maxPoints), metrics)
	if err != nil {
		if err == ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get historical metrics"})
		}
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetAllAgents returns a summary of all agents
func (h *HTTPHandler) GetAllAgents(c *gin.Context) {
	agents := h.store.GetAllAgents()
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// StartMetricsStreaming starts metrics streaming for an agent
func (h *HTTPHandler) StartMetricsStreaming(c *gin.Context) {
	agentID := c.Param("agentId")

	var req struct {
		IntervalSeconds int      `json:"interval_seconds"`
		Metrics         []string `json:"metrics"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 5 // default 5 seconds
	}

	// Create metrics request task
	task, err := h.taskManager.SendTask(&pb.TaskRequest{
		AgentId: agentID,
		Task: &pb.TaskRequest_MetricsRequest{
			MetricsRequest: &pb.MetricsRequest{
				RequestType: &pb.MetricsRequest_StreamRequest{
					StreamRequest: &pb.MetricsStreamRequest{
						Action:          pb.MetricsStreamRequest_START,
						IntervalSeconds: uint32(req.IntervalSeconds),
						Metrics:         req.Metrics,
					},
				},
			},
		},
	}, 5*time.Minute)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start metrics streaming", "details": err.Error()})
		return
	}

	// Update store status
	h.store.SetStreamingStatus(agentID, true)

	c.JSON(http.StatusOK, gin.H{
		"message":          "metrics streaming started",
		"task_id":          task.ID,
		"interval_seconds": req.IntervalSeconds,
	})
}

// StopMetricsStreaming stops metrics streaming for an agent
func (h *HTTPHandler) StopMetricsStreaming(c *gin.Context) {
	agentID := c.Param("agentId")

	// Create metrics request task
	task, err := h.taskManager.SendTask(&pb.TaskRequest{
		AgentId: agentID,
		Task: &pb.TaskRequest_MetricsRequest{
			MetricsRequest: &pb.MetricsRequest{
				RequestType: &pb.MetricsRequest_StreamRequest{
					StreamRequest: &pb.MetricsStreamRequest{
						Action: pb.MetricsStreamRequest_STOP,
					},
				},
			},
		},
	}, 5*time.Minute)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop metrics streaming", "details": err.Error()})
		return
	}

	// Update store status
	h.store.SetStreamingStatus(agentID, false)

	c.JSON(http.StatusOK, gin.H{
		"message": "metrics streaming stopped",
		"task_id": task.ID,
	})
}

// RefreshSystemInfo requests fresh system information from an agent
func (h *HTTPHandler) RefreshSystemInfo(c *gin.Context) {
	agentID := c.Param("agentId")

	var req struct {
		IncludeHardware bool `json:"include_hardware"`
		IncludeSoftware bool `json:"include_software"`
		IncludeNetwork  bool `json:"include_network"`
	}

	// Default to include all if not specified
	if err := c.ShouldBindJSON(&req); err != nil {
		req.IncludeHardware = true
		req.IncludeSoftware = true
		req.IncludeNetwork = true
	}

	// Create system info request task
	task, err := h.taskManager.SendTask(&pb.TaskRequest{
		AgentId: agentID,
		Task: &pb.TaskRequest_MetricsRequest{
			MetricsRequest: &pb.MetricsRequest{
				RequestType: &pb.MetricsRequest_SystemInfo{
					SystemInfo: &pb.SystemInfoRequest{
						IncludeHardware: req.IncludeHardware,
						IncludeSoftware: req.IncludeSoftware,
						IncludeNetwork:  req.IncludeNetwork,
					},
				},
			},
		},
	}, 5*time.Minute)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh system info", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "system info refresh requested",
		"task_id": task.ID,
	})
}
