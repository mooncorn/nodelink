# Task Management System

This document describes the task management system implemented in the nodelink project. The system provides a centralized orchestrator for managing tasks between clients and agents, with real-time notifications via Server-Sent Events (SSE).

## Overview

The task management system consists of:

1. **Task Manager** - Central orchestrator that manages task lifecycle
2. **Task Registry** - In-memory storage for active tasks
3. **Event Correlation** - Links agent responses to client requests
4. **Automatic Cleanup** - Removes completed/expired tasks
5. **Real-time Notifications** - SSE streaming for task updates

## Architecture

```
Client → HTTP API → Task Manager → gRPC → Agent
   ↑                      ↓
   └─── SSE Stream ←── Event Handler
```

### Components

#### Task Manager (`server/pkg/tasks/taskmgr.go`)
- **Task Registry**: In-memory map storing active tasks by ID
- **Task Lifecycle**: Create → Execute → Track → Complete/Timeout
- **Event Correlation**: Links incoming agent responses to original client requests
- **Cleanup**: Automatic task removal after completion or timeout

#### Task Statuses
- `pending` - Task created but not sent to agent
- `sent` - Task sent to agent
- `in_progress` - Agent acknowledged and started processing
- `completed` - Task finished successfully
- `failed` - Task failed with error
- `timeout` - Task exceeded timeout duration
- `cancelled` - Task was cancelled by client

## API Endpoints

### Task Creation

#### Create Shell Task
```
POST /tasks/shell
Content-Type: application/json

{
  "agent_id": "agent1",
  "cmd": "echo 'Hello World'",
  "timeout": 300
}

Response:
{
  "task_id": "uuid-string",
  "agent_id": "agent1",
  "status": "pending",
  "created_at": "2025-07-28T..."
}
```

#### Create Docker Task
```
POST /tasks/docker
Content-Type: application/json

{
  "agent_id": "agent1",
  "operation": "run",
  "image": "hello-world",
  "timeout": 300
}

Response:
{
  "task_id": "uuid-string",
  "agent_id": "agent1", 
  "status": "pending",
  "created_at": "2025-07-28T..."
}
```

**Docker Operations:**
- `run` - Run a new container (requires `image`)
- `start` - Start existing container (requires `id`)
- `stop` - Stop running container (requires `id`)
- `logs` - Get container logs (requires `id`)

### Task Management

#### Get Task Status
```
GET /tasks/{taskId}

Response:
{
  "task_id": "uuid-string",
  "agent_id": "agent1",
  "status": "completed",
  "created_at": "2025-07-28T...",
  "updated_at": "2025-07-28T...",
  "timeout": "5m0s"
}
```

#### Cancel Task
```
DELETE /tasks/{taskId}

Response:
{
  "message": "task cancelled"
}

Error Responses:
- 404: Task not found  
- 400: Task already in final state (completed/failed)
```

**Note**: Cancelling a task will:
- Send a cancel event to the agent via gRPC
- Kill the running process on the agent
- Update task status to "cancelled"
- Send cancellation acknowledgment via SSE

#### List Agent Tasks
```
GET /agents/{agentId}/tasks

Response:
{
  "agent_id": "agent1",
  "tasks": [
    {
      "task_id": "uuid-string",
      "agent_id": "agent1",
      "status": "completed",
      "created_at": "2025-07-28T...",
      "updated_at": "2025-07-28T..."
    }
  ]
}
```

#### List All Tasks
```
GET /tasks

Response:
{
  "tasks": [
    {
      "task_id": "uuid-string",
      "agent_id": "agent1",
      "status": "completed",
      "created_at": "2025-07-28T...",
      "updated_at": "2025-07-28T..."
    }
  ]
}
```

### Real-time Streaming

#### Stream Task Results
```
GET /stream?ref={taskId}
Accept: text/event-stream

Events:
- event: response
  data: {
    "event_response": {
      "event_ref": "task-id",
      "status": "SUCCESS",
      "result": {
        "data": {
          "output": "Hello World\n",
          "type": "stdout",
          "timestamp": 1690538400000,
          "sequence": 1,
          "is_final": true,
          "exit_code": 0
        }
      }
    }
  }
```

## Client Usage

### TypeScript Client

```typescript
import { TaskClient } from './task-client';

const client = new TaskClient();

// Execute shell command with streaming
const { taskId, stream } = await client.executeShell("agent1", "ls -la");

// Listen for results
stream.addEventListener("response", (event) => {
  const data = JSON.parse(event.data);
  console.log("Task output:", data);
});

// Check task status
const status = await client.getTaskStatus(taskId);
console.log("Task status:", status);

// List all tasks
const tasks = await client.listAllTasks();
console.log("All tasks:", tasks);
```

### Legacy Compatibility

The system maintains backward compatibility with the existing `/agents/{agentId}/shell` endpoint:

```javascript
const response = await fetch(`${serverURL}/agents/agent1/shell`, {
  method: "POST",
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({cmd: "echo hello"})
});

const data = await response.json();
const taskId = data.ref; // Now returns task ID

// Stream results using task ID
const es = new EventSource(`${serverURL}/stream?ref=${taskId}`);
```

## Features

### Task Lifecycle Management
- **Creation**: Tasks are created with unique IDs and stored in memory
- **Execution**: Tasks are sent to appropriate agents via gRPC
- **Tracking**: Task status is updated based on agent responses
- **Completion**: Tasks are marked complete when agents finish
- **Timeout**: Tasks automatically timeout if no response within deadline
- **Cleanup**: Completed tasks are automatically removed after 30 minutes

### Event Correlation
- **Request/Response Linking**: Each task gets a unique ID that agents use to reference responses
- **Status Updates**: Task status is updated in real-time based on agent responses
- **Progress Tracking**: Support for incremental updates during long-running tasks

### Real-time Notifications
- **SSE Streaming**: Clients can stream task results in real-time
- **Event Types**: Support for different event types (response, progress, etc.)
- **Room-based Broadcasting**: Multiple clients can subscribe to the same task

### Error Handling
- **Agent Disconnection**: Tasks fail gracefully if agent disconnects
- **Timeout Management**: Tasks timeout if agents don't respond within deadline
- **Cancellation**: Clients can cancel running tasks
- **Error Propagation**: Agent errors are properly propagated to clients

## Configuration

### Default Settings
- **Task Timeout**: 5 minutes (configurable per task)
- **Cleanup Interval**: 5 minutes
- **Cleanup Age**: 30 minutes for completed tasks
- **Buffer Size**: 100 responses per task

### Memory Management
- **Automatic Cleanup**: Completed tasks are automatically removed
- **Bounded Queues**: Task response channels are bounded to prevent memory leaks
- **Graceful Shutdown**: Tasks are properly cleaned up on server shutdown

## Development

### Running the System

1. **Start the server:**
```bash
cd server
go run cmd/server/main.go
```

2. **Start an agent:**
```bash
cd agent  
go run cmd/agent/main.go -agent_id=agent1 -agent_token=secret_token1
```

3. **Test with client:**
```bash
cd client
npm install
npx ts-node task-client.ts
```

### Testing

The system includes comprehensive testing capabilities:

- **Unit Tests**: Test individual components
- **Integration Tests**: Test full request/response cycle
- **Load Tests**: Test with multiple concurrent tasks
- **Stress Tests**: Test timeout and error conditions

### Monitoring

The system provides extensive logging for monitoring:

- Task creation and completion
- Agent connections and disconnections  
- Error conditions and timeouts
- Performance metrics

## Future Enhancements

- **Persistence**: Optional database storage for task history
- **Authentication**: Secure task management with user auth
- **Rate Limiting**: Prevent task spam from clients
- **Metrics**: Prometheus metrics for monitoring
- **Webhooks**: HTTP callbacks for task completion
- **Scheduling**: Delayed and recurring task execution
