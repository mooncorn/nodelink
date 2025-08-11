package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/events"
	"github.com/mooncorn/nodelink/server/internal/events/processors"
	"github.com/mooncorn/nodelink/server/internal/metrics"
	"github.com/mooncorn/nodelink/server/internal/sse"
	"github.com/mooncorn/nodelink/server/internal/tasks"
	"github.com/mooncorn/nodelink/server/internal/types"
	"google.golang.org/grpc"
)

func main() {
	// Create metrics store and handlers
	metricsStore := metrics.NewMetricsStore(7 * 24 * time.Hour) // 7 days retention

	taskServer := tasks.NewTaskServer(metricsStore)
	metricsHTTPHandler := metrics.NewHTTPHandler(metricsStore, taskServer)
	metricsSSEHandler := metrics.NewSSEHandler(metricsStore, taskServer)

	// Start metrics SSE handler
	metricsSSEHandler.Start()
	defer metricsSSEHandler.Stop()

	// Create SSE manager for tasks
	config := sse.ManagerConfig{
		BufferSize:     100,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	eventHandler := sse.NewDefaultEventHandler[*pb.TaskResponse](true)
	sseManager := sse.NewManager(config, eventHandler)
	sseManager.Start()
	defer sseManager.Stop()

	// Create event router with processors
	eventRouter := events.NewEventRouter(sseManager, metricsStore)

	// Register event processors
	eventRouter.RegisterProcessor(processors.NewShellProcessor())
	eventRouter.RegisterProcessor(processors.NewMetricsProcessor(metricsStore))
	eventRouter.RegisterProcessor(processors.NewDockerProcessor())

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcServer, taskServer)

	// Add task listener to forward task events to event router
	taskServer.AddListener(func(task *types.Task) {
		sseManager.EnableRoomBuffering(task.ID, 10)

		// Process the event through the event router
		if task.Response != nil {
			eventRouter.ProcessAndRelay(task.Response)
		}

		// Legacy compatibility: also send to room directly
		sseManager.SendToRoom(task.ID, task.Response, "response")

		// Process metrics responses for legacy metrics handler
		if task.Response != nil {
			if metricsResp := task.Response.GetMetricsResponse(); metricsResp != nil {
				metricsSSEHandler.ProcessMetricsResponse(task.Request.AgentId, metricsResp)
			}
		}
	})

	// Start cleanup routine for completed tasks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			taskServer.CleanupCompletedTasks(30 * time.Minute)
			metricsStore.CleanupOldData()
		}
	}()

	// HTTP/SSE server setup
	router := gin.Default()
	registerRESTRoutes(router, taskServer)
	registerSSERoutes(router, sseManager)

	// Register metrics routes
	metricsHTTPHandler.RegisterRoutes(router)
	metricsSSEHandler.RegisterRoutes(router)

	// Start gRPC server in background
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		log.Println("gRPC Event Server starting on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	log.Println("HTTP Server starting on :8080")
	router.Run()
}

func registerSSERoutes(router gin.IRouter, sseManager *sse.Manager[*pb.TaskResponse]) {
	router.GET("/stream", sse.SSEHeaders(), sse.SSEConnection(sseManager), func(c *gin.Context) {
		client, ok := sse.GetClientFromContext[*pb.TaskResponse](c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}

		taskId := c.Query("ref")
		if taskId != "" {
			sseManager.JoinRoom(client.ID, taskId)
		}

		// Handle the stream
		sse.HandleSSEStream[*pb.TaskResponse](c)
	})
}

func registerRESTRoutes(router gin.IRouter, taskSender interface {
	SendTask(request *pb.TaskRequest, timeout time.Duration) (*types.Task, error)
	GetTask(taskID string) (*types.Task, bool)
	CancelTask(taskID string) error
	ListTasks(agentID string) []*types.Task
}) {

	// Create a shell execution task
	router.POST("/tasks/shell", func(ctx *gin.Context) {
		var req struct {
			AgentId string `json:"agent_id" binding:"required"`
			Cmd     string `json:"cmd" binding:"required"`
			Timeout int    `json:"timeout"` // timeout in seconds, default 300 (5 minutes)
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}

		timeout := time.Duration(req.Timeout) * time.Second
		if timeout == 0 {
			timeout = 5 * time.Minute // default timeout
		}

		// Create task
		task, err := taskSender.SendTask(&pb.TaskRequest{
			AgentId: req.AgentId,
			Task: &pb.TaskRequest_ShellExecute{
				ShellExecute: &pb.ShellExecute{
					Cmd: req.Cmd,
				},
			},
		}, timeout)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task", "details": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"task_id":    task.ID,
			"agent_id":   task.Request.AgentId,
			"status":     task.Status.String(),
			"created_at": task.CreatedAt,
		})
	})

	// Get task status and details
	router.GET("/tasks/:taskId", func(ctx *gin.Context) {
		taskId := ctx.Param("taskId")
		task, exists := taskSender.GetTask(taskId)
		if !exists {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"task_id":    task.ID,
			"agent_id":   task.Request.AgentId,
			"status":     task.Status.String(),
			"created_at": task.CreatedAt,
			"updated_at": task.UpdatedAt,
			"timeout":    task.Timeout.String(),
		})
	})

	// Cancel a task
	router.DELETE("/tasks/:taskId", func(ctx *gin.Context) {
		taskId := ctx.Param("taskId")
		err := taskSender.CancelTask(taskId)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
	})

	// List tasks for an agent
	router.GET("/agents/:agentId/tasks", func(ctx *gin.Context) {
		agentId := ctx.Param("agentId")
		tasks := taskSender.ListTasks(agentId)

		var taskList []gin.H
		for _, task := range tasks {
			taskList = append(taskList, gin.H{
				"task_id":    task.ID,
				"agent_id":   task.Request.AgentId,
				"status":     task.Status.String(),
				"created_at": task.CreatedAt,
				"updated_at": task.UpdatedAt,
			})
		}

		ctx.JSON(http.StatusOK, gin.H{
			"agent_id": agentId,
			"tasks":    taskList,
		})
	})

	// List all tasks
	router.GET("/tasks", func(ctx *gin.Context) {
		allTasks := taskSender.ListTasks("")

		var taskList []gin.H
		for _, task := range allTasks {
			taskList = append(taskList, gin.H{
				"task_id":    task.ID,
				"agent_id":   task.Request.AgentId,
				"status":     task.Status.String(),
				"created_at": task.CreatedAt,
				"updated_at": task.UpdatedAt,
			})
		}

		ctx.JSON(http.StatusOK, gin.H{"tasks": taskList})
	})
}
