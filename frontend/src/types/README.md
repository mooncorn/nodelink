# Frontend Types

This file contains TypeScript type definitions for the React frontend. Most types are now imported from the shared `nodelink-shared` package to avoid duplication and ensure consistency across the codebase.

## Imported from Shared Package

The following types are imported from `nodelink-shared`:

### Core Types

- `ActionType` - Union type of all available action types
- `Task` - Task interface with execution details
- `TaskStatus` - Task status union type
- `NodeRegistration` - Node registration payload
- `NodeInfo` - Node information from server

### Event Types

- `ServerToFrontendEvents` - WebSocket events from server to frontend
- `NodeToServerEvents` - WebSocket events from nodes to server
- `FrontendToServerEvents` - WebSocket events from frontend to server
- `ServerToNodeEvents` - WebSocket events from server to nodes

### Action Payload Types

- `ShellExecuteAction` - Shell command execution payload
- `DockerRunAction` - Docker container run payload
- `DockerDeleteAction` - Docker container delete payload
- `DockerStartAction` - Docker container start payload
- `DockerStopAction` - Docker container stop payload
- `DockerListAction` - Docker container list payload
- `SystemInfoAction` - System information request payload
- `SystemHealthAction` - System health check payload

## Frontend-Specific Types

### `Node`

Frontend-specific node interface that extends the shared `NodeInfo` type with additional fields:

- Simplified status: only "online" | "offline" (excludes "busy")
- Flexible `lastSeen`: supports both string and Date types for API compatibility
- Additional frontend fields: `token?` and `socketId?`

### `ApiResponse<T>`

Generic wrapper for REST API responses:

```typescript
interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}
```

### `Stats`

Server statistics for the dashboard:

```typescript
interface Stats {
  nodes: number;
  onlineNodes: number;
  totalTasks: number;
  runningTasks: number;
  frontendConnections: number;
}
```

### `TaskOptions`

Additional task configuration options:

```typescript
interface TaskOptions {
  timeout?: number;
  priority?: number;
}
```

## Type Aliases for Backward Compatibility

- `ShellPayload` → `ShellExecuteAction`
- `DockerRunPayload` → `DockerRunAction`
- `DockerControlPayload` → `DockerDeleteAction | DockerStartAction | DockerStopAction`

## Usage

```typescript
import type {
  Node,
  Task,
  ActionType,
  ApiResponse,
  ShellPayload,
  DockerRunPayload,
} from "../types";
```
