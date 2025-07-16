# Types Directory

This directory contains the core TypeScript type definitions for a distributed task execution system.

## Core Concepts

### Actions (`actions.ts`)

Actions are commands that can be executed on remote nodes. The system supports:

- **Docker operations**: Run, start, stop, delete, and list containers
- **Shell commands**: Execute arbitrary shell commands with environment and timeout control
- **System monitoring**: Get system information and health checks

Each action type has a specific payload structure that defines the required parameters.

### Tasks (`tasks.ts`)

Tasks represent the execution lifecycle of actions. A task:

- Has a unique ID and tracks which node will execute it
- Goes through status phases: pending → running → completed/failed/cancelled
- Includes timing information (created, started, completed)
- Supports retry logic and timeout handling
- Stores execution results and error information

### Events (`events.ts`)

Events enable real-time communication between system components:

- **Server ↔ Node**: Task execution, cancellation, heartbeats, and configuration
- **Server ↔ Frontend**: Task updates, node status, and real-time progress
- **Node → Server**: Registration, heartbeats, task results, and output streaming

The event system supports bidirectional WebSocket communication for low-latency updates.

### Responses (`responses.ts`)

Responses define the structured data returned after action execution:

- Each action type has a corresponding response format
- Includes success/failure information, results, and metadata
- Provides standardized error handling across all operations
- Contains detailed information like exit codes, performance metrics, and output

## System Architecture

This type system supports a distributed architecture where frontend clients can submit tasks that get executed on remote nodes, with real-time progress updates flowing back through the server.
