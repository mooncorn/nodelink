package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mooncorn/nodelink/agent/pkg/grpc"
)

// Track running tasks to enable cancellation
var runningTasks = make(map[string]*os.Process)
var cancelledTasks = make(map[string]bool)
var tasksMutex sync.RWMutex

func main() {
	agentID := flag.String("agent_id", "", "Agent ID")
	agentToken := flag.String("agent_token", "", "Agent Auth Token")
	flag.Parse()

	log.Println("Starting Agent...")

	// Create grpc client
	client, err := grpc.NewHeartbeatClient("localhost:9090")
	if err != nil {
		log.Fatalf("Failed to create grpc client: %v", err)
	}
	defer client.Close()

	// Connect to the server
	if err := client.Connect(*agentID, *agentToken); err != nil {
		log.Fatalf("Failed to connect to grpc server: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent is running. Press Ctrl+C to exit.")
	<-c

	log.Println("Agent shutting down...")
}

// // handleTaskCancel handles task cancellation requests
// func handleTaskCancel(taskRequest *pb.TaskRequest, client *grpc.TaskClient, metricsHandler *metrics.Handler) {
// 	taskID := taskRequest.TaskId
// 	tasksMutex.Lock()
// 	process, exists := runningTasks[taskID]
// 	if exists {
// 		log.Printf("Cancelling shell task %s", taskID)

// 		// Kill the process
// 		if process != nil {
// 			err := process.Kill()
// 			if err != nil {
// 				log.Printf("Error killing process for task %s: %v", taskID, err)
// 			} else {
// 				log.Printf("Successfully killed process for task %s", taskID)
// 			}
// 		}

// 		// Mark task as cancelled
// 		cancelledTasks[taskID] = true
// 		delete(runningTasks, taskID)
// 	} else {
// 		// Check if this is a metrics streaming task
// 		if metricsHandler.GetCollector().IsStreamingTask(taskID) {
// 			log.Printf("Cancelling metrics streaming task %s", taskID)
// 			metricsHandler.GetCollector().StopStreaming()
// 		} else {
// 			log.Printf("Task %s not found in running tasks or metrics streaming", taskID)
// 		}
// 	}
// 	tasksMutex.Unlock()

// 	// Send cancellation acknowledgment
// 	client.Send(&pb.TaskResponse{
// 		AgentId:   taskRequest.AgentId,
// 		TaskId:    taskID,
// 		IsFinal:   true,
// 		Status:    pb.TaskResponse_COMPLETED,
// 		Cancelled: true,
// 		EventType: "task_cancel",
// 		Timestamp: time.Now().Unix(),
// 		Response: &pb.TaskResponse_TaskCancel{
// 			TaskCancel: &pb.TaskCancelResponse{
// 				Message: "Task cancelled succesfully",
// 			},
// 		},
// 	})
// }

// func handleShellExecute(taskRequest *pb.TaskRequest, shellExecute *pb.ShellExecute, client *grpc.TaskClient, metricsCollector *metrics.MetricsCollector) {
// 	cmdStr := shellExecute.Cmd
// 	cmd := exec.Command("bash", "-c", cmdStr)

// 	// Get pipes before starting
// 	stdout, err := cmd.StdoutPipe()
// 	if err != nil {
// 		log.Printf("Failed to create stdout pipe: %v", err)
// 		sendErrorResponse(taskRequest, client, err)
// 		return
// 	}
// 	defer stdout.Close()

// 	stderr, err := cmd.StderrPipe()
// 	if err != nil {
// 		log.Printf("Failed to create stderr pipe: %v", err)
// 		sendErrorResponse(taskRequest, client, err)
// 		return
// 	}
// 	defer stderr.Close()

// 	// Start the command
// 	if err := cmd.Start(); err != nil {
// 		log.Printf("Failed to start command: %v", err)
// 		sendErrorResponse(taskRequest, client, err)
// 		return
// 	}

// 	// Store the running command AFTER it starts
// 	tasksMutex.Lock()
// 	runningTasks[taskRequest.TaskId] = cmd.Process
// 	tasksMutex.Unlock()

// 	// Track process for metrics if collector is available
// 	if metricsCollector != nil {
// 		metricsCollector.AddTaskProcess(taskRequest.TaskId, int32(cmd.Process.Pid))
// 	}

// 	// Remove from running tasks when done
// 	defer func() {
// 		tasksMutex.Lock()
// 		delete(runningTasks, taskRequest.TaskId)
// 		tasksMutex.Unlock()

// 		// Remove process tracking
// 		if metricsCollector != nil {
// 			metricsCollector.RemoveTaskProcess(taskRequest.TaskId)
// 		}
// 	}()

// 	// Use channels to coordinate goroutines and prevent race conditions
// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	// Stream stdout
// 	go func() {
// 		defer wg.Done()
// 		streamOutput(taskRequest, client, stdout, true)
// 	}()

// 	// Stream stderr
// 	go func() {
// 		defer wg.Done()
// 		streamOutput(taskRequest, client, stderr, false)
// 	}()

// 	// Wait for streaming to complete
// 	wg.Wait()

// 	// Wait for process to complete
// 	err = cmd.Wait()
// 	exitCode := 0
// 	if err != nil {
// 		if exitErr, ok := err.(*exec.ExitError); ok {
// 			exitCode = exitErr.ExitCode()
// 		}
// 	}

// 	// Check if task was cancelled
// 	tasksMutex.RLock()
// 	wasCancelled := cancelledTasks[taskRequest.TaskId]
// 	tasksMutex.RUnlock()

// 	// Send final response
// 	client.Send(&pb.TaskResponse{
// 		AgentId:   taskRequest.AgentId,
// 		TaskId:    taskRequest.TaskId,
// 		Status:    pb.TaskResponse_COMPLETED,
// 		IsFinal:   true,
// 		Cancelled: wasCancelled,
// 		EventType: "shell_output",
// 		Timestamp: time.Now().Unix(),
// 		Response: &pb.TaskResponse_ShellExecute{
// 			ShellExecute: &pb.ShellExecuteResponse{
// 				Stdout:   "",
// 				Stderr:   "",
// 				ExitCode: int32(exitCode),
// 			},
// 		},
// 	})

// 	// Clean up cancellation tracking
// 	tasksMutex.Lock()
// 	delete(cancelledTasks, taskRequest.TaskId)
// 	tasksMutex.Unlock()
// }

// // Helper function to send error responses
// func sendErrorResponse(taskRequest *pb.TaskRequest, client *grpc.TaskClient, err error) {
// 	client.Send(&pb.TaskResponse{
// 		AgentId:   taskRequest.AgentId,
// 		TaskId:    taskRequest.TaskId,
// 		Status:    pb.TaskResponse_FAILURE,
// 		IsFinal:   true,
// 		Cancelled: false,
// 		EventType: "shell_output",
// 		Timestamp: time.Now().Unix(),
// 		Response: &pb.TaskResponse_ShellExecute{
// 			ShellExecute: &pb.ShellExecuteResponse{
// 				Stdout:   "",
// 				Stderr:   err.Error(),
// 				ExitCode: 1,
// 			},
// 		},
// 	})
// }

// // Helper function to stream output without race conditions
// func streamOutput(taskRequest *pb.TaskRequest, client *grpc.TaskClient, reader io.Reader, isStdout bool) {
// 	scanner := bufio.NewScanner(reader)
// 	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // More efficient buffering

// 	for scanner.Scan() {
// 		text := scanner.Text() + "\n"

// 		var stdout, stderr string
// 		if isStdout {
// 			stdout = text
// 		} else {
// 			stderr = text
// 		}

// 		client.Send(&pb.TaskResponse{
// 			AgentId:   taskRequest.AgentId,
// 			TaskId:    taskRequest.TaskId,
// 			Status:    pb.TaskResponse_IN_PROGRESS,
// 			IsFinal:   false,
// 			Cancelled: false,
// 			EventType: "shell_output",
// 			Timestamp: time.Now().Unix(),
// 			Response: &pb.TaskResponse_ShellExecute{
// 				ShellExecute: &pb.ShellExecuteResponse{
// 					Stdout:   stdout,
// 					Stderr:   stderr,
// 					ExitCode: 0,
// 				},
// 			},
// 		})
// 	}
// }
