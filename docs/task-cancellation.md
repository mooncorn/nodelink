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
Client Request → Server → Task Manager → gRPC Cancel Event → Agent
     ↓              ↓           ↓              ↓              ↓
Cancel API → Update Status → Send Cancel → Kill Process → Send Response
     ↓              ↓           ↓              ↓              ↓  
SSE Update ← Event Handler ← Process Response ← Cleanup ← ACK Cancel
```

## Implementation Details

### 1. Protocol Buffer Changes

Added `TaskCancel` message and updated `ServerToNodeEvent`:

```protobuf
message ServerToNodeEvent {
  string agent_id = 1;
  string task_id = 2;
  oneof task {
    ShellExecute shell_execute = 3;
    DockerOperation docker_operation = 4;
    LogMessage log_message = 5;
    TaskCancel task_cancel = 6;  // NEW
  }
}

message TaskCancel {
  string reason = 1;
}
```

### 2. Task Manager Updates

#### EventSender Interface
Added interface to avoid circular dependencies:
```go
type EventSender interface {
    Send(event *eventstream.ServerToNodeEvent) (string, error)
}
```

#### Enhanced CancelTask Method
```go
func (tm *TaskManager) CancelTask(taskID string) error {
    // Check if task exists and can be cancelled
    // Update task status to cancelled
    // Send cancel event to agent via gRPC
    // Clean up resources
}
```

Key features:
- Prevents cancellation of already completed tasks
- Sends cancel notification to agent
- Updates task status atomically
- Handles agent disconnection gracefully

### 3. Agent Task Tracking

#### Running Tasks Registry
```go
var (
    runningTasks = make(map[string]*exec.Cmd)
    tasksMutex   sync.RWMutex
)
```

#### Cancel Event Handler
```go
func handleTaskCancel(taskID string, client *grpc.EventClient) {
    // Find running process
    // Kill the process
    // Send acknowledgment
    // Clean up tracking
}
```

#### Shell Execution Updates
- Store running commands in registry
- Enable process termination via `Process.Kill()`
- Clean up on completion or cancellation

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
- 400: Task already in final state
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

### 2. Cancellation with Status Monitoring

```typescript
// Create task and start monitoring
const { taskId, stream } = await client.executeShell("agent1", "long-running-command");

// Monitor status
const checkStatus = setInterval(async () => {
    const status = await client.getTaskStatus(taskId);
    console.log(`Task status: ${status.status}`);
    
    if (status.status === 'cancelled') {
        clearInterval(checkStatus);
        stream.close();
    }
}, 1000);

// Cancel after 10 seconds
setTimeout(() => client.cancelTask(taskId), 10000);
```

### 3. Bulk Cancellation

```typescript
// Cancel all running tasks for an agent
const tasks = await client.listAgentTasks("agent1");
const runningTasks = tasks.filter(t => 
    t.status === 'sent' || t.status === 'in_progress'
);

for (const task of runningTasks) {
    try {
        await client.cancelTask(task.task_id);
        console.log(`Cancelled task ${task.task_id}`);
    } catch (error) {
        console.error(`Failed to cancel ${task.task_id}:`, error);
    }
}
```

## Task Status Flow

```
pending → sent → in_progress → completed/failed
    ↓       ↓          ↓
cancelled ← cancelled ← cancelled
```

### Status Transitions

1. **pending** → **cancelled**: Task cancelled before being sent to agent
2. **sent** → **cancelled**: Task cancelled after being sent but before agent response
3. **in_progress** → **cancelled**: Task cancelled while running on agent
4. **completed/failed**: Final states, cannot be cancelled

## Error Handling

### Client-Side Errors
- **Task Not Found**: Task ID doesn't exist
- **Already Completed**: Task is in final state
- **Network Error**: Server unreachable

### Agent-Side Handling
- **Process Kill Failure**: Log error but still mark as cancelled
- **Task Not Found**: Send acknowledgment (task may have completed)
- **Connection Loss**: Server detects via gRPC stream closure

### Server-Side Resilience
- **Agent Disconnected**: Mark task as failed
- **Cancel Event Send Failure**: Log error but update task status
- **Concurrent Access**: Mutex protection for task registry

## Testing

### Automated Tests
```bash
# Run cancellation test suite
./test_cancellation.sh
```

### Manual Testing
```bash
# Terminal 1: Start server
cd server && go run cmd/server/main.go

# Terminal 2: Start agent  
cd agent && go run cmd/agent/main.go -agent_id=agent1 -agent_token=secret_token1

# Terminal 3: Test cancellation
cd client && npx ts-node test-cancellation.ts
```

### Test Scenarios

1. **Basic Cancellation**: Cancel a simple shell command
2. **Long-Running Task**: Cancel a task that runs for minutes
3. **Multiple Tasks**: Cancel multiple concurrent tasks
4. **Edge Cases**: Cancel already completed/failed tasks
5. **Agent Disconnect**: Cancel when agent is offline
6. **Rapid Cancel**: Cancel immediately after creation

## Performance Considerations

### Memory Usage
- Task registry cleaned up automatically
- Response channels are bounded (100 entries)
- Completed tasks removed after 30 minutes

### Latency
- Cancel requests processed immediately
- gRPC streaming provides low-latency communication
- Process termination is immediate (`SIGKILL`)

### Scalability
- Concurrent task cancellation supported
- Per-agent task isolation
- Lock-free read operations where possible

## Security Considerations

### Authentication
- Same authentication as task creation
- Agent tokens required for all operations

### Authorization
- Clients can only cancel their own tasks
- Agents validate task ownership

### Process Isolation
- Processes killed cleanly with proper cleanup
- No privileged operations required

## Monitoring and Logging

### Server Logs
```
Task abc123 cancelled
Sent cancel event to agent agent1 for task abc123
Failed to send cancel event to agent agent1 for task abc123: connection lost
```

### Agent Logs
```
Agent received task cancel for task abc123: cancelled by user
Cancelling task abc123
Successfully killed process for task abc123
Task abc123 not found in running tasks
```

### Metrics (Future Enhancement)
- Cancellation rate per agent
- Average time to cancel
- Failed cancellation attempts

## Known Limitations

1. **Docker Tasks**: Currently only shell tasks support cancellation
2. **Process Trees**: Child processes may not be killed
3. **File Cleanup**: Temporary files may remain after cancellation
4. **Graceful Shutdown**: Uses `SIGKILL` instead of `SIGTERM`

## Future Enhancements

1. **Graceful Cancellation**: Send `SIGTERM` first, then `SIGKILL`
2. **Docker Support**: Cancel running containers
3. **Process Groups**: Kill entire process trees
4. **Cancellation Reason**: More detailed cancellation reasons
5. **Timeout Cancellation**: Configurable timeouts
6. **Webhook Notifications**: HTTP callbacks on cancellation
