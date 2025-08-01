package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	servergrpc "github.com/mooncorn/nodelink/server/pkg/grpc"
	"github.com/mooncorn/nodelink/server/pkg/metrics"
	"github.com/mooncorn/nodelink/server/pkg/sse"
	"github.com/mooncorn/nodelink/server/pkg/tasks"
	"google.golang.org/grpc"
)

func main() {
	// Create task manager
	taskManager := tasks.NewTaskManager()

	// Create metrics store and handlers
	metricsStore := metrics.NewMetricsStore(7 * 24 * time.Hour) // 7 days retention
	metricsHTTPHandler := metrics.NewHTTPHandler(metricsStore, taskManager)
	metricsSSEHandler := metrics.NewSSEHandler(metricsStore, taskManager)

	// Start metrics SSE handler
	metricsSSEHandler.Start()
	defer metricsSSEHandler.Stop()

	// Create gRPC server and register it
	grpcServer := grpc.NewServer()
	agentServer := servergrpc.NewTaskServer(taskManager.GetResponseChannel(), metricsStore)
	pb.RegisterAgentServiceServer(grpcServer, agentServer)

	// Set the task server in task manager for sending tasks
	taskManager.SetTaskServer(agentServer)

	// Start cleanup routine for completed tasks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			taskManager.CleanupCompletedTasks(30 * time.Minute)
			metricsStore.CleanupOldData()
		}
	}()

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

	// Add task manager listener to forward task events to SSE
	taskManager.AddListener(func(task *tasks.Task) {
		sseManager.EnableRoomBuffering(task.ID, 10)
		sseManager.SendToRoom(task.ID, task.Response, "response")

		// Process metrics responses
		if task.Response != nil {
			if metricsResp := task.Response.GetMetricsResponse(); metricsResp != nil {
				metricsSSEHandler.ProcessMetricsResponse(task.Request.AgentId, metricsResp)
			}
		}
	})

	// HTTP/SSE server setup
	router := gin.Default()
	registerRESTRoutes(router, taskManager)
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

func registerRESTRoutes(router gin.IRouter, taskManager *tasks.TaskManager) {
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
		task, err := taskManager.SendTask(&pb.TaskRequest{
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
		task, exists := taskManager.GetTask(taskId)
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
		err := taskManager.CancelTask(taskId)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
	})

	// List tasks for an agent
	router.GET("/agents/:agentId/tasks", func(ctx *gin.Context) {
		agentId := ctx.Param("agentId")
		tasks := taskManager.ListTasks(agentId)

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
		allTasks := taskManager.ListTasks("")

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
