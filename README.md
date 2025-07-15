# NodeLink - Remote Server Management Platform

A modern, type-safe web application for managing remote servers (nodes) with real-time communication and Docker integration.

## Features

- **Type-Safe Architecture**: Full TypeScript with shared types and runtime validation
- **REST API + WebSocket**: HTTP API for operations, Socket.IO for real-time updates
- **Docker Integration**: Native Docker API using dockerode for container management
- **Modern Web Interface**: Tabbed interface for nodes, tasks, shell, Docker, and system operations
- **Task-Based Execution**: Scalable task management with retry mechanisms and progress tracking
- **Multi-Node Support**: Connect and manage multiple nodes simultaneously

## Architecture

```
┌─────────────────┐  HTTP/WSS  ┌─────────────────┐  Socket.IO  ┌─────────────────┐
│   Web Frontend  │ ◄────────► │   Server        │ ◄─────────► │   Node Agent    │
│   (Browser)     │            │   (Express +    │             │   (TypeScript)  │
│                 │            │   Socket.IO)    │             │                 │
└─────────────────┘            └─────────────────┘             └─────────────────┘
```

## Prerequisites

- Node.js (v16 or higher)
- npm or yarn
- Docker (optional, for Docker actions)
- `mkcert` (for SSL certificates)

## Quick Start

### 1. Setup SSL Certificates

```bash
# Install mkcert and generate certificates
cd server/certs
mkcert localhost
mkcert -install
```

### 2. Install Dependencies

```bash
# Install shared dependencies
cd shared && npm install

# Install server dependencies
cd ../server && npm install

# Install node agent dependencies
cd ../node-agent && npm install
```

### 3. Build Shared Package

```bash
cd shared
npm run build
```

### 4. Start the Services

```bash
# Terminal 1: Start server
./start-server.sh

# Terminal 2: Start node agent
./start-node.sh node1

# Terminal 3: Start additional nodes (optional)
./start-node.sh node2
```

### 5. Access the Application

Open `https://localhost:8443` in your browser and accept the SSL certificate.

## API Endpoints

### REST API

```
GET  /health                    # Server health check
GET  /api/nodes                 # List all nodes
GET  /api/nodes/:nodeId         # Get specific node
GET  /api/tasks                 # List tasks (with filtering)
GET  /api/tasks/:taskId         # Get specific task
POST /api/tasks                 # Create new task
DELETE /api/tasks/:taskId       # Cancel task
GET  /api/stats                 # Server statistics
```

### WebSocket Events

```javascript
// Real-time events
socket.on('task.created', (data) => { ... });
socket.on('task.output', (data) => { ... });
socket.on('task.completed', (data) => { ... });
socket.on('node.connected', (data) => { ... });
```

## Supported Actions

### Shell Commands

```javascript
{
  "type": "shell.execute",
  "payload": {
    "command": "ls -la",
    "timeout": 30000,
    "cwd": "/home/user"
  }
}
```

### Docker Operations

```javascript
{
  "type": "docker.start",
  "payload": {
    "image": "nginx:latest",
    "containerName": "web-server",
    "ports": [{ "host": 8080, "container": 80 }]
  }
}
```

### System Information

```javascript
{
  "type": "system.info",
  "payload": {
    "includeMetrics": true,
    "includeProcesses": true
  }
}
```

## Project Structure

```
nodelink/
├── shared/                    # Shared TypeScript types and utilities
│   ├── src/types/            # Type definitions
│   ├── src/validation/       # Zod schemas
│   └── package.json
├── server/                   # Main server application
│   ├── src/
│   │   ├── server.ts        # Express server with REST API
│   │   ├── task-manager.ts  # Task lifecycle management
│   │   └── node-manager.ts  # Node registration and monitoring
│   ├── certs/               # SSL certificates
│   └── index.ts
├── node-agent/               # Node agent application
│   ├── src/
│   │   ├── node-agent.ts    # Main agent with Socket.IO
│   │   ├── action-executor.ts # Action execution coordinator
│   │   └── docker-actions.ts # Docker operations with dockerode
│   └── index.ts
├── frontend/                 # Web interface
│   ├── index.html           # Modern tabbed interface
│   └── app.js               # HTTP API + Socket.IO client
├── start-server.sh          # Server startup script
├── start-node.sh            # Node agent startup script
└── test-http-api.js         # API testing script
```

## Testing

### HTTP API Testing

```bash
node test-http-api.js
```

### Web Interface

- **Nodes Tab**: View connected nodes and capabilities
- **Tasks Tab**: Manage and filter tasks
- **Shell Tab**: Execute shell commands
- **Docker Tab**: Container management
- **System Tab**: System information and health checks

## Troubleshooting

### SSL Certificate Issues

```bash
cd server/certs
rm localhost.pem localhost-key.pem
mkcert localhost
```

### Node Connection Issues

- Verify server is running on port 8443
- Check SSL certificates are valid
- Ensure no firewall blocking connections

### Docker Issues

- Verify Docker is installed and running
- Check Docker daemon is accessible
- Ensure user has Docker permissions

## Development

### Building Shared Package

```bash
cd shared
npm run build
```

### Running in Development Mode

```bash
# Server with auto-reload
cd server
npx ts-node index.ts

# Node agent with auto-reload
cd node-agent
npx ts-node index.ts node1
```

## License

This project is open source and available under the [MIT License](LICENSE).
