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

## Development Guidelines

### 4. SSE Streaming
- Use [`SSEManager`](server/internal/sse/manager.go) for real-time client updates
- Send to specific clients via `SendToClient()` or broadcast with `BroadcastToRoom()`
- Handle connection lifecycle with proper cleanup

## File Organization

### Server Structure
- [`cmd/server/main.go`](server/cmd/server/main.go): Main server entry point
- [`internal/sse/`](server/internal/sse/): Real-time streaming infrastructure
- [`internal/metrics/`](server/internal/metrics/): Metrics collection and storage
- [`internal/command/`](server/internal/command/): Command execution feature
- [`internal/docker/`](server/internal/command/): Docker control feature

### Agent Structure  
- [`cmd/agent/main.go`](agent/cmd/agent/main.go): Main agent entry point
- [`pkg/grpc/client.go`](agent/pkg/grpc/client.go): gRPC client implementation
- [`pkg/metrics/`](agent/pkg/metrics/): Metrics collection on agent side

### Protocol Definitions
- [`proto/agent.proto`](proto/agent.proto): Protocol buffer definitions

## Security Considerations
- Agent authentication via tokens
- Resource limits and timeouts for long-running tasks