# Agent Implementation

This document describes the Agent implementation in the nodelink project, which executes tasks on behalf of the server and provides real-time feedback.

## Overview

The Agent is a client application that:
- Connects to the server via gRPC streaming
- Receives task requests from the server
- Executes shell commands and Docker operations
- Streams real-time output back to the server
- Supports task cancellation and cleanup
- Maintains connection health and auto-reconnection

## Architecture

```
┌─────────────────┐    gRPC Stream    ┌──────────────────┐
│     Server      │ ←──────────────→  │      Agent       │
│  Task Manager   │                   │  Task Executor   │
└─────────────────┘                   └──────────────────┘
                                               │
                                               ▼
                                      ┌──────────────────┐
                                      │   Local System   │
                                      │ Shell/Docker/etc │
                                      └──────────────────┘
```

## Core Components

### 1. Main Agent Process

```go
func main() {
    agentID := flag.String("agent_id", "", "Agent ID")
    agentToken := flag.String("agent_token", "", "Agent Auth Token")
    
    // Create and configure gRPC client
    client, err := grpc.NewTaskClient("localhost:9090")
    
    // Register task handlers
    client.AddListener(func(taskRequest *pb.TaskRequest) {
        switch task := taskRequest.Task.(type) {
        case *pb.TaskRequest_TaskCancel:
            handleTaskCancel(taskRequest, client)
        case *pb.TaskRequest_ShellExecute:
            handleShellExecute(taskRequest, task.ShellExecute, client)
        case *pb.TaskRequest_DockerOperation:
            handleDockerOperation(taskRequest, task.DockerOperation, client)
        }
    })
    
    // Connect and maintain connection
    client.Connect(*agentID, *agentToken)
    
    // Wait for shutdown signal
    waitForShutdown()
}
```

### 2. Task Tracking System

#### Running Tasks Registry
```go
var (
    runningTasks = make(map[string]*os.Process)  // Task ID → Process
    cancelledTasks = make(map[string]bool)       // Task ID → Cancelled flag
    tasksMutex sync.RWMutex                      // Protects both maps
)
```

**Purpose:**
- Track active processes for cancellation support
- Coordinate between execution and cancellation handlers
- Clean up resources when tasks complete

**Thread Safety:**
- RWMutex allows concurrent reads (status checks)
- Exclusive writes for process registration/cleanup
- Atomic operations for cancellation flags

### 3. Shell Command Execution

#### Command Lifecycle
```go
func handleShellExecute(taskRequest *pb.TaskRequest, shellExecute *pb.ShellExecute, client *grpc.TaskClient) {
    // 1. Create command with bash -c
    cmd := exec.Command("bash", "-c", shellExecute.Cmd)
    
    // 2. Set up pipes for output streaming
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    
    // 3. Start the process
    cmd.Start()
    
    // 4. Register for cancellation tracking
    tasksMutex.Lock()
    runningTasks[taskRequest.TaskId] = cmd.Process
    tasksMutex.Unlock()
    
    // 5. Stream output concurrently
    var wg sync.WaitGroup
    wg.Add(2)
    go streamOutput(taskRequest, client, stdout, true)   // stdout
    go streamOutput(taskRequest, client, stderr, false)  // stderr
    wg.Wait()
    
    // 6. Wait for completion and send final response
    err := cmd.Wait()
    sendFinalResponse(taskRequest, client, err)
    
    // 7. Clean up tracking
    cleanupTask(taskRequest.TaskId)
}
```

#### Output Streaming
```go
func streamOutput(taskRequest *pb.TaskRequest, client *grpc.TaskClient, reader io.Reader, isStdout bool) {
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // Efficient buffering
    
    for scanner.Scan() {
        text := scanner.Text() + "\n"
        
        // Send incremental response
        client.Send(&pb.TaskResponse{
            AgentId:   taskRequest.AgentId,
            TaskId:    taskRequest.TaskId,
            Status:    pb.TaskResponse_IN_PROGRESS,
            IsFinal:   false,
            Cancelled: false,
            Response: &pb.TaskResponse_ShellExecute{
                ShellExecute: &pb.ShellExecuteResponse{
                    Stdout:   isStdout ? text : "",
                    Stderr:   isStdout ? "" : text,
                    ExitCode: 0,
                },
            },
        })
    }
}
```

**Features:**
- **Line-by-line streaming**: Real-time output delivery
- **Separate stdout/stderr**: Preserved output separation
- **Efficient buffering**: Large buffer sizes for performance
- **Non-blocking**: Concurrent streaming of both streams

### 4. Task Cancellation Handling

#### Cancellation Process
```go
func handleTaskCancel(taskRequest *pb.TaskRequest, client *grpc.TaskClient) {
    taskID := taskRequest.TaskId
    
    tasksMutex.Lock()
    process, exists := runningTasks[taskID]
    if exists {
        // Kill the process immediately
        if process != nil {
            err := process.Kill() // SIGKILL for immediate termination
            if err != nil {
                log.Printf("Error killing process for task %s: %v", taskID, err)
            }
        }
        
        // Mark as cancelled and remove from running tasks
        cancelledTasks[taskID] = true
        delete(runningTasks, taskID)
    }
    tasksMutex.Unlock()
    
    // Send cancellation acknowledgment
    client.Send(&pb.TaskResponse{
        AgentId:   taskRequest.AgentId,
        TaskId:    taskID,
        IsFinal:   true,
        Status:    pb.TaskResponse_COMPLETED,
        Cancelled: true,
        Response: &pb.TaskResponse_TaskCancel{
            TaskCancel: &pb.TaskCancelResponse{
                Message: "Task cancelled successfully",
            },
        },
    })
}
```

**Cancellation Features:**
- **Immediate Termination**: Uses `process.Kill()` (SIGKILL)
- **Race Condition Safe**: Handles missing processes gracefully
- **Acknowledgment**: Always sends response even if process not found
- **State Tracking**: Updates cancellation flags for final response coordination

### 5. Error Handling

#### Graceful Error Response
```go
func sendErrorResponse(taskRequest *pb.TaskRequest, client *grpc.TaskClient, err error) {
    client.Send(&pb.TaskResponse{
        AgentId:   taskRequest.AgentId,
        TaskId:    taskRequest.TaskId,
        Status:    pb.TaskResponse_FAILURE,
        IsFinal:   true,
        Cancelled: false,
        Response: &pb.TaskResponse_ShellExecute{
            ShellExecute: &pb.ShellExecuteResponse{
                Stdout:   "",
                Stderr:   err.Error(),
                ExitCode: 1,
            },
        },
    })
}
```

**Error Scenarios:**
- **Command Creation Failure**: Invalid shell syntax
- **Pipe Creation Failure**: System resource exhaustion
- **Process Start Failure**: Permission issues, missing executables
- **Connection Loss**: Network interruption during execution

## Process Management

### 1. Process Lifecycle

```
Create Command → Set up Pipes → Start Process → Register for Cancellation
      ↓                ↓              ↓                    ↓
Stream Output ← Read from Pipes ← Process Running ← Track in Registry
      ↓                ↓              ↓                    ↓
Wait Complete ← Close Pipes ← Process Exits ← Remove from Registry
```

### 2. Resource Management

#### Process Cleanup
```go
defer func() {
    tasksMutex.Lock()
    delete(runningTasks, taskRequest.TaskId)
    delete(cancelledTasks, taskRequest.TaskId)
    tasksMutex.Unlock()
}()
```

#### Pipe Management
- **Automatic Closure**: Deferred cleanup of stdin/stdout/stderr pipes
- **Buffer Management**: Efficient scanner buffers for large outputs
- **Goroutine Coordination**: WaitGroup ensures all streaming completes

### 3. Concurrency Model

#### Goroutine Usage
1. **Main Thread**: gRPC message handling and task dispatch
2. **Task Execution**: One goroutine per task for command execution
3. **Output Streaming**: Two goroutines per task (stdout + stderr)
4. **Background Cleanup**: Periodic cleanup of expired tracking data

#### Synchronization
- **RWMutex**: Protects task tracking maps
- **WaitGroup**: Coordinates output streaming completion
- **Context**: Handles cancellation and timeouts
- **Channels**: gRPC client manages connection state

## Communication Protocol

### 1. Task Request Processing

#### Message Types
```protobuf
message TaskRequest {
  string agent_id = 1;
  string task_id = 2;
  oneof task {
    ShellExecute shell_execute = 3;
    DockerOperation docker_operation = 4;
    TaskCancel task_cancel = 5;
  }
}
```

#### Response Types
```protobuf
message TaskResponse {
  string agent_id = 1;
  string task_id = 2;
  TaskResponse.Status status = 3;
  bool is_final = 4;
  bool cancelled = 5;
  oneof response {
    ShellExecuteResponse shell_execute = 6;
    DockerOperationResponse docker_operation = 7;
    TaskCancelResponse task_cancel = 8;
  }
}
```

### 2. Response Flow

#### Incremental Responses
- **Purpose**: Real-time output streaming
- **Frequency**: One response per line of output
- **Status**: `IN_PROGRESS`
- **IsFinal**: `false`

#### Final Response
- **Purpose**: Task completion notification
- **Content**: Exit code, final status, cancellation flag
- **Status**: `COMPLETED` or `FAILURE`
- **IsFinal**: `true`

## Security Considerations

### 1. Command Execution
- **Shell Wrapper**: All commands executed via `bash -c`
- **No Privilege Escalation**: Runs with agent user privileges
- **Process Isolation**: Each task runs in separate process

### 2. Authentication
- **Agent Token**: Required for server connection
- **Agent ID**: Identifies agent in multi-agent deployments
- **gRPC Security**: Can be configured with TLS

### 3. Resource Limits
- **Process Limits**: Inherits system ulimits
- **Memory Usage**: Bounded by scanner buffer sizes
- **File Descriptors**: Properly cleaned up via defer statements

## Performance Characteristics

### 1. Execution Overhead
- **Process Creation**: ~1-5ms per task
- **Output Buffering**: 64KB read buffer, 1MB max line
- **gRPC Streaming**: Low-latency bidirectional communication

### 2. Memory Usage
- **Per Task**: ~100KB for buffers and tracking
- **Concurrent Tasks**: Linear scaling with task count
- **Cleanup**: Immediate cleanup prevents accumulation

### 3. Scalability
- **Task Concurrency**: Limited by system process limits
- **Output Throughput**: Handled by goroutine per stream
- **Network Efficiency**: Streaming reduces latency vs batch

## Configuration

### 1. Command Line Options
```bash
./agent -agent_id=agent1 -agent_token=secret_token1
```

### 2. Environment Variables (Future)
- `NODELINK_SERVER_ADDRESS`: gRPC server address
- `NODELINK_AGENT_ID`: Agent identifier
- `NODELINK_AGENT_TOKEN`: Authentication token
- `NODELINK_MAX_CONCURRENT_TASKS`: Task limit

### 3. Configuration File (Future)
```yaml
server:
  address: "localhost:9090"
  tls_enabled: false

agent:
  id: "agent1"
  token: "secret_token1"
  max_concurrent_tasks: 100
  
logging:
  level: "info"
  format: "json"
```

## Monitoring and Observability

### 1. Logging
```go
log.Printf("Agent received shell execute for task %s: %s", taskRequest.TaskId, cmd)
log.Printf("Cancelling task %s", taskID)
log.Printf("Successfully killed process for task %s", taskID)
log.Printf("Task %s not found in running tasks", taskID)
```

### 2. Metrics (Future Enhancement)
- Task execution count
- Task success/failure rates
- Average task duration
- Concurrent task count
- Output streaming throughput

### 3. Health Checks
- gRPC connection status
- System resource utilization
- Running task count
- Last heartbeat timestamp

## Error Recovery

### 1. Connection Recovery
- **Auto-reconnect**: gRPC client handles connection failures
- **Exponential Backoff**: Prevents connection spam
- **State Preservation**: Running tasks continue during brief disconnections

### 2. Task Recovery
- **Orphaned Tasks**: Server timeout handles lost tasks
- **Process Cleanup**: OS handles process cleanup on agent crash
- **State Synchronization**: Task IDs prevent duplicate execution

### 3. Resource Recovery
- **Memory Leaks**: Proper cleanup prevents accumulation
- **File Descriptors**: Deferred cleanup ensures release
- **Goroutine Leaks**: WaitGroups prevent abandoned goroutines

## Future Enhancements

### 1. Advanced Process Management
- **Process Groups**: Kill entire process trees
- **Resource Limits**: Per-task CPU/memory limits  
- **Graceful Termination**: SIGTERM before SIGKILL

### 2. Enhanced Security
- **Sandboxing**: Containerized task execution
- **Command Filtering**: Whitelist/blacklist support
- **User Switching**: Run tasks as different users

### 3. Extended Functionality
- **File Transfer**: Support for file upload/download
- **Environment Management**: Per-task environment variables
- **Working Directory**: Configurable execution directory
- **Interactive Shell**: Support for interactive commands

### 4. Monitoring Integration
- **Prometheus Metrics**: Detailed performance metrics
- **Health Endpoints**: HTTP health check endpoints
- **Distributed Tracing**: OpenTelemetry integration
- **Log Aggregation**: Structured logging for centralized collection
