# Copilot Instructions for Nodelink Project

## Project Overview
Nodelink is a distributed task execution system with gRPC-based agent communication and real-time streaming capabilities. The system consists of:
- **Server**: Central orchestrator with HTTP APIs and SSE streaming
- **Agent**: Task executor that connects to server via gRPC
- **Proto**: Shared protocol buffer definitions for communication

## Architecture Components

### Communication Protocol
- **gRPC**: Bidirectional streaming between server and agents
- **Protocol Buffers**: [`agent.proto`](proto/agent.proto) defines all message types
- **Message Types**: Uses `oneof` for type-safe message routing (ping/pong, commands, responses)

### Internal Package Structure
Each internal package is self-contained and manages its own domain:

#### Status Management (`internal/status/`)
- **Purpose**: Centralized agent status tracking and lifecycle management
- **Components**:
  - `manager.go`: Core status tracking with event notifications
  - `http_handler.go`: HTTP API for querying agent status
  - `sse_handler.go`: Real-time status change streaming
- **Dependencies**: None (foundation layer)

#### Ping/Pong Monitoring (`internal/ping/`)
- **Purpose**: Connection health monitoring through periodic ping/pong exchanges
- **Components**:
  - `handler.go`: Ping/pong message processing and timeout management
- **Dependencies**: `status` (updates agent status based on ping responses)

#### Command Execution (`internal/command/`)
- **Purpose**: Remote command execution on agents
- **Components**:
  - `handler.go`: Command request/response management
  - `http_handler.go`: HTTP API for executing commands
- **Dependencies**: `status` (validates agent availability)

#### Terminal Management (`internal/terminal/`)
- **Purpose**: Interactive terminal session management and real-time terminal streaming
- **Components**:
  - `handler.go`: Terminal session lifecycle and command processing
  - `http_handler.go`: HTTP API for terminal operations
  - `sse_handler.go`: Real-time terminal output streaming
  - `session.go`: Terminal session state management
- **Dependencies**: `status`, `common`, `sse` (validates agent availability, uses shared interfaces, leverages SSE utilities)

#### Metrics Collection (`internal/metrics/`)
- **Purpose**: System metrics collection and real-time metrics streaming
- **Components**:
  - `handler.go`: Metrics request/response management
  - `http_handler.go`: HTTP API for metrics endpoints
  - `sse_handler.go`: Real-time metrics streaming
  - `manager.go`: Centralized metrics polling and distribution
- **Dependencies**: `status`, `common`, `sse` (validates agent availability, uses shared interfaces, leverages SSE utilities)

#### Common Types and Interfaces (`internal/common/`)
- **Purpose**: Shared types, interfaces, and constants used across packages
- **Components**:
  - `types.go`: Common data structures (AgentInfo, AgentStatus, StatusChangeEvent, etc.)
  - `interfaces.go`: Shared interfaces (StreamSender, StatusManager, Authenticator, etc.)
  - `constants.go`: Common constants and error definitions
- **Dependencies**: None (foundation layer)

#### Server-Sent Events Infrastructure (`internal/sse/`)
- **Purpose**: Real-time streaming infrastructure and SSE utilities
- **Components**:
  - `manager.go`: SSE client management and message distribution
  - `utils.go`: Reusable SSE connection handling and utilities
  - `formatters.go`: Message formatting utilities for SSE
- **Dependencies**: `common` (implements common interfaces)

#### Authentication (`internal/auth/`)
- **Purpose**: Agent authentication and security
- **Components**:
  - `authenticator.go`: Agent authentication logic
- **Dependencies**: `common` (uses shared error definitions)

#### Communication (`internal/comm/`)
- **Purpose**: gRPC stream management and message routing
- **Components**:
  - `communication.go`: Bidirectional stream handling and message dispatch
- **Dependencies**: `status`, `ping`, `command`, `terminal`, `metrics`, `auth` (coordinates all communication)

## Development Guidelines

### Code Quality Guidelines

#### Interface Usage
- **Always use interfaces when injecting dependencies**: Prefer `common.StatusManager` over `*status.Manager`
- **Use interfaces, types, and constants from common package when possible**: Leverage shared definitions to maintain consistency
- **Move interfaces, types, and constants to common unless domain-specific**: Only keep in local packages when they are truly package-specific

#### SSE Development
- **SSE handlers should use reusable SSE logic**: Leverage `sse.ConnectionHandler` and utilities from the SSE package
- **Use common SSE patterns**: Follow the established connection lifecycle patterns for consistency

### Package Design Principles
1. **Single Responsibility**: Each package manages one core concern
2. **Self-Contained**: Packages include their own HTTP/SSE handlers
3. **Dependency Direction**: Clear one-way dependency flow (no circular dependencies)
4. **Event-Driven**: Status changes propagate through listener pattern

### Dependency Flow
```
comm → status, ping, command, terminal, metrics, auth, common, sse (orchestrates all)
terminal → status, common, sse (manages sessions, uses shared interfaces, leverages SSE utilities)
metrics → status, common, sse (collects metrics, uses shared interfaces, leverages SSE utilities)
ping → status, common (updates agent health, uses shared constants)
command → status, common (checks agent availability, uses shared interfaces)
auth → common (uses shared error definitions)
status → common (uses shared types and interfaces)
sse → common (implements common interfaces)
common → (no dependencies - foundation layer)
```

### Adding New Features
1. Update protocol buffers if new message types are needed
2. Create new package in `internal/` for the feature domain 
3. Implement core logic, HTTP handlers, and SSE handlers within the package
4. Add dependencies only to lower-level packages
5. Register routes in main.go
6. Implement core logic in the agent

## File Organization

### Server Structure
- `cmd/server/main.go`: Main server entry point
- `internal/sse/`: Real-time streaming infrastructure
- `internal/status/`: Centralized agent status tracking
- `internal/ping/`: Heartbeat and connection monitoring
- `internal/command/`: Command execution feature
- `internal/terminal/`: Interactive terminal session management
- `internal/metrics/`: System metrics collection and streaming
- `internal/comm/`: gRPC communication and message routing
- `internal/auth/`: Agent authentication and security
- `internal/common/`: Shared types, interfaces, and constants

### Agent Structure  
- `cmd/agent/main.go`: Main agent entry point
- `pkg/grpc/client.go`: gRPC client implementation
- `pkg/command/`: Command execution handling on agent side
- `pkg/terminal/`: Terminal session management on agent side
- `pkg/metrics/`: Metrics collection on agent side

### Protocol Definitions
- Use generate.sh script to generate protobuf files
- `proto/agent.proto`: Protocol buffer definitions

## Security Considerations
- Agent authentication via tokens
- Resource limits and timeouts for long-running tasks