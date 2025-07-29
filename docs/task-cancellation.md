# Task Cancellation Implementation

This document describes the task cancellation feature implementation in the nodelink project.

## Overview

The task cancellation system allows clients to cancel running tasks, which will:
1. Stop the task execution on the agent side
2. Update the task status to "cancelled" 
3. Clean up resources
4. Notify clients via SSE

## Architecture

```
Client Request → Server → Task Manager → gRPC TaskRequest → Agent
     ↓              ↓           ↓              ↓              ↓
Cancel API → Update Status → Send Cancel → Kill Process → Send Response
     ↓              ↓           ↓              ↓              ↓  
SSE Update ← Event Handler ← Process Response ← Cleanup ← ACK Cancel
```

## Implementation Details

### 1. Protocol Buffer Structure

The system uses `TaskRequest` with `TaskCancel` variant for cancellation:

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

message TaskCancel {
  string reason = 1;
}

message TaskCancelResponse {
  string message = 1;
}
```

### 2. Task Manager Implementation

#### Task Registry and Response Processing
```go
type TaskManager struct {
    mu         sync.RWMutex
    tasks      map[string]*Task
    listeners  []TaskEventListener
    taskServer *grpc.TaskServer
    respCh     chan *pb.TaskResponse
    stopCh     chan struct{}
}
```

Key features:
- **Response Channel**: Bounded channel (100 entries) for processing task responses
- **Task Lifecycle**: Automatic cleanup with 5-second delay after completion
- **Status Handling**: Proper differentiation between cancellation and completion

#### Enhanced CancelTask Method
```go
func (tm *TaskManager) CancelTask(taskID string) error {
    // Validate task exists and is cancellable
    // Update task status to cancelled
    // Send TaskCancel request to agent via gRPC
    // Handle agent disconnection gracefully
}
```

#### Response Processing Logic
The task manager processes responses with proper cancellation detection:
- Checks `taskResponse.Cancelled` flag
- Distinguishes between completion and cancellation
- Updates task status appropriately
- Triggers cleanup after final response

### 3. Agent Task Tracking

#### Running Tasks Registry
```go
var runningTasks = make(map[string]*os.Process)
var cancelledTasks = make(map[string]bool)
var tasksMutex sync.RWMutex
```

The agent maintains two maps:
- `runningTasks`: Maps task IDs to running processes
- `cancelledTasks`: Tracks which tasks were cancelled for final response

#### Task Cancellation Handler
```go
func handleTaskCancel(taskRequest *pb.TaskRequest, client *grpc.TaskClient) {
    // Find and kill the running process
    // Mark task as cancelled
    // Send acknowledgment response
    // Clean up tracking
}
```

Process termination:
- Uses `process.Kill()` for immediate termination (SIGKILL)
- Handles missing processes gracefully
- Updates tracking maps atomically

#### Shell Execution with Cancellation Support
```go
func handleShellExecute(taskRequest *pb.TaskRequest, shellExecute *pb.ShellExecute, client *grpc.TaskClient) {
    // Create and start command
    // Register process in runningTasks
    // Stream output with goroutines
    // Check cancellation status in final response
    // Clean up tracking
}
```

Features:
- Process registration after successful start
- Concurrent output streaming
- Cancellation state checking
- Proper cleanup on completion

### 4. API Endpoints

#### Cancel Task
```
DELETE /tasks/{taskId}

Response:
{
  "message": "task cancelled"
}

Error Responses:
- 404: Task not found
- 400: Task already in final state (completed/failed/cancelled)
```

## Usage Examples

### 1. Basic Cancellation

```typescript
const client = new TaskClient();

// Create a long-running task
const task = await client.createShellTask("agent1", "sleep 60", 120);

// Cancel it after 5 seconds
setTimeout(async () => {
    await client.cancelTask(task.task_id);
}, 5000);
```

### 2. Cancellation with SSE Monitoring

```typescript
// Create task and start monitoring
const { taskId, stream } = await client.executeShell("agent1", "for i in {1..20}; do echo \"Line $i\"; sleep 2; done");

// Monitor for cancellation
stream.addEventListener("response", (event) => {
    const data = JSON.parse(event.data);
    if (data.cancelled && data.isFinal) {
        console.log("Task was successfully cancelled");
        stream.close();
    }
});

// Cancel after 10 seconds
setTimeout(() => client.cancelTask(taskId), 10000);
```

### 3. Race Condition Handling

```typescript
// Test cancelling a quick task (might complete before cancellation)
const task = await client.createShellTask("agent1", "echo 'Quick task' && sleep 1", 30);

setTimeout(async () => {
    try {
        await client.cancelTask(task.task_id);
        console.log("Task cancelled");
    } catch (error) {
        console.log("Task already completed:", error.message);
    }
}, 2000);
```

## Task Status Flow

```
pending → sent → in_progress → completed/failed
    ↓       ↓          ↓
cancelled ← cancelled ← cancelled
```

### Status Transitions with Cancellation

1. **pending** → **cancelled**: Task cancelled before being sent to agent
2. **sent** → **cancelled**: Task cancelled after being sent but before agent starts
3. **in_progress** → **cancelled**: Task cancelled while running on agent
4. **completed/failed**: Final states, cannot be cancelled

### Response Processing

The task manager handles cancellation responses by:
1. Checking `taskResponse.Cancelled` flag
2. Setting appropriate status based on `isFinal` and `Cancelled` flags
3. Triggering cleanup after 5-second delay for final responses

## Error Handling

### Client-Side Errors
- **Task Not Found (404)**: Task ID doesn't exist in registry
- **Already Final State (400)**: Task is completed/failed/cancelled
- **Network Error**: Server unreachable or timeout

### Agent-Side Handling
- **Process Kill Failure**: Logs error but still sends cancellation acknowledgment
- **Task Not Found**: Sends acknowledgment (task may have completed naturally)
- **Connection Loss**: Server detects via gRPC stream closure

### Server-Side Resilience
- **Agent Disconnected**: Task marked as failed
- **Cancel Event Send Failure**: Logs error but updates task status
- **Concurrent Access**: Mutex protection for task registry
- **Response Channel Full**: Drops responses with logging

## Performance Considerations

### Memory Usage
- **Bounded Channels**: Response channel limited to 100 entries
- **Automatic Cleanup**: Tasks removed 5 seconds after completion
- **Process Tracking**: Agent cleans up tracking maps immediately

### Latency
- **Immediate Processing**: Cancel requests processed without delay
- **gRPC Streaming**: Low-latency bidirectional communication
- **Process Termination**: SIGKILL provides immediate termination

### Scalability
- **Per-Agent Isolation**: Tasks isolated by agent ID
- **Concurrent Operations**: Multiple cancellations supported
- **Lock-Free Reads**: Read operations use RWMutex for better concurrency

## Testing

### Comprehensive Test Suite

The `test-cancellation.ts` includes three test scenarios:

1. **Quick Cancellation**: Cancel long-running task after 3 seconds
2. **Immediate Cancellation**: Cancel task after 500ms
3. **Race Condition**: Try cancelling quick task that might complete first

### Test Output Analysis
- Monitors task status through SSE
- Checks `cancelled` flag in responses
- Verifies proper cleanup and final status
- Handles both successful cancellation and race conditions

## Security Considerations

### Process Management
- **Clean Termination**: Uses `process.Kill()` for reliable termination
- **Resource Cleanup**: Proper cleanup of pipes and goroutines
- **Privilege Isolation**: No elevated privileges required

### Authentication
- **Same as Task Creation**: Uses existing agent token system
- **Task Ownership**: Only the creating client can cancel tasks
- **Agent Validation**: Agents validate task IDs before processing

## Known Limitations

1. **Shell Tasks Only**: Currently only shell tasks support cancellation
2. **SIGKILL Only**: Uses immediate kill instead of graceful termination
3. **Process Trees**: Child processes may not be killed automatically
4. **File Cleanup**: Temporary files created by cancelled tasks may remain

## Future Enhancements

1. **Graceful Cancellation**: Implement SIGTERM → SIGKILL sequence
2. **Docker Support**: Add cancellation for Docker operations
3. **Process Groups**: Kill entire process trees using process groups
4. **Cleanup Hooks**: Allow tasks to register cleanup callbacks
5. **Cancellation Reasons**: More detailed cancellation reason codes
6. **Timeout Cancellation**: Automatic cancellation on timeout
6. **Webhook Notifications**: HTTP callbacks on cancellation
