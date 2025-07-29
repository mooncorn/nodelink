# Agent Metrics System

The Agent Metrics System provides comprehensive monitoring and observability for agents in the NodeLink distributed task execution platform. It enables real-time collection, storage, and streaming of system metrics including CPU, memory, disk, network, and process information.

## Overview

The metrics system consists of three main components:

1. **Agent-Side Metrics Collection** - Collects system metrics using the gopsutil library
2. **Server-Side Metrics Storage** - Stores and manages metrics data with HTTP APIs
3. **Client Libraries** - TypeScript/JavaScript libraries for consuming metrics data

```
┌─────────────────┐    gRPC Stream    ┌──────────────────┐
│     Server      │ ←──────────────→  │      Agent       │
│  Metrics Store  │                   │ Metrics Collector│
│  HTTP API       │                   │ System Monitor   │
└─────────────────┘                   └──────────────────┘
         │                                       │
         ▼                                       ▼
┌─────────────────┐                    ┌──────────────────┐
│   Web Client    │                    │   System APIs    │
│  Dashboard      │                    │  gopsutil/psutil │
│  SSE Stream     │                    │  /proc, /sys     │
└─────────────────┘                    └──────────────────┘
```

## Features

### System Information Collection
- **Hardware Information**: CPU model, cores, memory, disk configuration
- **Software Information**: Operating system, hostname, uptime, installed packages
- **Network Interfaces**: Interface details, IP addresses, MAC addresses

### Real-Time Metrics
- **CPU Metrics**: Usage percentage, per-core utilization, user/system/idle/iowait breakdown
- **Memory Metrics**: Total/used/available memory, swap usage, cache and buffer statistics
- **Disk Metrics**: Usage percentages, I/O rates (read/write bytes and operations per second)
- **Network Metrics**: Interface traffic rates, packet counts, error and drop statistics
- **Process Metrics**: Total process counts by state, task-specific process tracking
- **Load Metrics**: System load averages (1, 5, and 15 minute)

### Streaming and Storage
- **Real-Time Streaming**: Server-Sent Events (SSE) for live metrics updates
- **Configurable Intervals**: Adjustable collection frequency (default: 5 seconds)
- **Historical Storage**: In-memory storage with automatic cleanup
- **Multiple Agents**: Support for monitoring multiple agents simultaneously
- **Immediate Agent Registration**: Agents appear in HTTP API immediately upon gRPC connection
- **Connection Status Tracking**: Real-time tracking of agent online/offline status
- **Event-Based SSE**: Server sends events with type "metrics" requiring `addEventListener('metrics', callback)`
- **Automatic Stream Management**: Streaming starts/stops automatically based on SSE client connections

## Architecture

### Two-Layer Agent Management

The system uses a two-layer approach for agent management:

1. **gRPC Layer**: Handles real-time bidirectional communication
   - Agents connect and maintain persistent streams
   - Task distribution and response collection
   - Connection health monitoring

2. **HTTP/Metrics Layer**: Provides REST API and data storage
   - Agent visibility based on data contribution
   - Metrics storage and historical data
   - Public API for client applications

**Key Insight**: Agents are immediately visible in the HTTP API upon gRPC connection, showing their connection status. System info and metrics data are populated separately as they become available.

### Agent-Side Components

#### MetricsCollector (`agent/pkg/metrics/collector.go`)
The core metrics collection engine that:
- Uses goroutines for non-blocking metrics collection
- Leverages the `gopsutil` library for cross-platform system metrics
- Calculates rates and deltas for network and disk I/O
- Tracks task-specific process metrics
- Sends metrics via gRPC streaming

#### MetricsHandler (`agent/pkg/metrics/handler.go`)
Handles incoming metrics requests from the server:
- Processes system info requests
- Manages streaming start/stop/update commands
- Handles historical metrics queries (future enhancement)

### Server-Side Components

#### MetricsStore (`server/pkg/metrics/store.go`)
In-memory storage for metrics data:
- Maintains agent status and metadata
- Stores current and historical metrics
- Provides automatic cleanup of old data
- Thread-safe operations with read-write mutexes

#### HTTP Handlers (`server/pkg/metrics/http_handler.go`)
RESTful API endpoints for metrics access:
- Agent discovery and status
- System information retrieval
- Metrics streaming control
- Current metrics snapshots
- Historical metrics queries

#### SSE Handler (`server/pkg/metrics/sse_handler.go`)
Real-time metrics streaming via Server-Sent Events:
- Live metrics broadcasting
- Per-agent streaming channels
- Automatic client connection management
- Smart streaming lifecycle: starts streaming when first SSE client connects, stops when last client disconnects
- Client reference counting to prevent resource waste

## API Reference

### Agent Discovery

#### Get All Agents
```http
GET /metrics/agents
```
Returns overview of all connected agents with their metrics status.

**Response:**
```json
{
  "agents": {
    "agent1": {
      "agent_id": "agent1",
      "connected": true,
      "last_seen": "2025-07-29T13:32:57Z",
      "last_update": "2025-07-29T13:32:57Z",
      "streaming_active": true,
      "has_system_info": true,
      "has_metrics": true
    }
  }
}
```

**Response Fields:**
- `connected`: Whether the agent is currently connected via gRPC
- `last_seen`: Timestamp of last gRPC communication
- `last_update`: Timestamp of last metrics/system info update
- `streaming_active`: Whether metrics streaming is currently active
- `has_system_info`: Whether system information is available
- `has_metrics`: Whether current metrics data is available

### System Information

#### Refresh System Info
```http
POST /agents/{agentId}/system/refresh
```
Triggers collection of fresh system information from the agent.

#### Get System Info
```http
GET /agents/{agentId}/system
```
Returns detailed system information including hardware, software, and network configuration.

**Response includes:**
- CPU details (model, cores, frequency, features)
- Memory configuration (total, type, speed)
- Disk information (devices, filesystems, capacity)
- Network interfaces (names, MAC addresses, IP addresses)
- Operating system details (name, version, kernel)
- Installed packages (future enhancement)

### Metrics Collection

#### Start Metrics Streaming
```http
POST /agents/{agentId}/metrics/start
Content-Type: application/json

{
  "interval_seconds": 5,
  "metrics": ["cpu", "memory", "disk", "network"]
}
```

**Note**: This manual endpoint is available but typically not needed. The system automatically starts streaming when the first SSE client connects via `/agents/{agentId}/metrics/stream`.

#### Stop Metrics Streaming
```http
POST /agents/{agentId}/metrics/stop
```

**Note**: This manual endpoint is available but typically not needed. The system automatically stops streaming when the last SSE client disconnects. Alternative methods to stop streaming:
- **Automatic**: Last SSE client disconnects (recommended)
- **Task Cancellation**: `DELETE /tasks/{taskId}` (for administrative control)
- **Manual API**: This endpoint (for custom applications)

#### Get Current Metrics
```http
GET /agents/{agentId}/metrics
```
Returns the most recent metrics snapshot.

#### Stream Real-Time Metrics
```http
GET /agents/{agentId}/metrics/stream?interval=5
Accept: text/event-stream
```
Server-Sent Events stream providing real-time metrics updates.

**Automatic Streaming Management**: 
- When the first SSE client connects, the server automatically starts metrics streaming on the agent
- When the last SSE client disconnects, the server automatically stops metrics streaming
- This ensures efficient resource usage and prevents streaming to empty audiences
- Multiple clients can connect simultaneously and share the same metrics stream

**Important**: The server sends events with event type "metrics". Use `eventSource.addEventListener('metrics', callback)` instead of `eventSource.onmessage` for proper event handling.

#### Get Historical Metrics
```http
GET /agents/{agentId}/metrics/history?metrics=cpu.usage_percent,memory.usage_percent&start=1640995200&end=1641000000&interval=60&max_points=100
```

## Client Libraries

### TypeScript/JavaScript Client

```typescript
import { MetricsClient, MetricsFormatter } from './metrics-client';

const client = new MetricsClient('http://localhost:8080');

// Get all agents
const agents = await client.getAllAgents();

// Get system information
const systemInfo = await client.getSystemInfo('agent1');

// Stream real-time metrics (automatically starts streaming on agent)
const eventSource = client.streamMetrics('agent1', 5);
// Note: Server sends events with type "metrics", so use addEventListener
eventSource.addEventListener('metrics', (event) => {
  const metrics = JSON.parse(event.data);
  console.log('CPU:', MetricsFormatter.formatPercentage(metrics.cpu.usage_percent));
});

// Manual streaming control (optional - typically not needed)
// await client.startMetricsStreaming('agent1', { interval_seconds: 3 });

// Get current metrics
const metrics = await client.getCurrentMetrics('agent1');

// When done, close the event source (automatically stops streaming if last client)
// eventSource.close();

// Set up EventSource polyfill for Node.js environments
import EventSource from 'eventsource';
if (typeof window === 'undefined') {
  (global as any).EventSource = EventSource;
}
```

### Formatting Utilities

The `MetricsFormatter` class provides utilities for displaying metrics:

- `formatBytes(bytes)` - Human-readable byte sizes (1.5 GB)
- `formatPercentage(value)` - Percentage formatting (45.2%)
- `formatUptime(seconds)` - Uptime formatting (2h 15m)
- `formatLoadAverage(load)` - Load average formatting (1.23)
- `formatNetworkSpeed(bytesPerSec)` - Network speed (150 Mbps)
- `formatDiskIO(bytesPerSec)` - Disk I/O rates (45 MB/s)

## Protocol Buffer Definitions

The metrics system uses Protocol Buffers for efficient data serialization between agents and server:

### Key Message Types

- `MetricsRequest` - Commands sent to agents (system info, streaming control)
- `MetricsResponse` - Data returned from agents (system info, metrics data)
- `SystemInfoResponse` - Complete system information
- `MetricsDataResponse` - Real-time metrics snapshot

### Streaming Protocol

Metrics are streamed using the existing gRPC bidirectional streaming infrastructure:

1. Server sends `MetricsRequest` with streaming parameters
2. Agent starts metrics collection loop
3. Agent sends periodic `MetricsDataResponse` messages
4. Server processes and stores metrics data
5. Server broadcasts to SSE clients

### Automatic Streaming Management

The system includes intelligent streaming lifecycle management to optimize resource usage:

#### How It Works
1. **First Client Connection**: When the first SSE client connects to `/agents/{agentId}/metrics/stream`, the server automatically starts metrics streaming on the agent
2. **Client Tracking**: The server maintains a reference count of connected SSE clients per agent
3. **Automatic Cleanup**: When the last SSE client disconnects, the server automatically stops metrics streaming on the agent
4. **Resource Efficiency**: This prevents agents from collecting and streaming metrics when no one is listening

#### Benefits
- **No Resource Waste**: Agents only collect metrics when needed
- **Automatic Management**: No manual start/stop required for typical use cases
- **Multiple Client Support**: Multiple SSE clients can share the same metrics stream
- **Clean Disconnection Handling**: Proper cleanup when browsers close or clients disconnect

#### Manual Override
The manual `/agents/{agentId}/metrics/start` and `/agents/{agentId}/metrics/stop` endpoints are still available for:
- Debugging and testing scenarios
- Custom applications that need streaming without SSE
- Administrative control over metrics collection

**Example Workflow:**
```javascript
// First client connects - streaming starts automatically
const eventSource1 = client.streamMetrics('agent1', 5);

// Second client connects - shares existing stream
const eventSource2 = client.streamMetrics('agent1', 5);

// First client disconnects - streaming continues
eventSource1.close();

// Second client disconnects - streaming stops automatically
eventSource2.close();
```

## Configuration

### Agent Configuration
- Collection intervals: 1-300 seconds (default: 5 seconds)
- Metric selection: Choose specific metric categories
- Process tracking: Enable task-specific process monitoring

### Server Configuration
- Storage retention: Automatic cleanup after configurable time (default: 30 minutes)
- Buffer sizes: Configurable memory limits for metrics storage
- Cleanup intervals: Periodic cleanup frequency (default: 5 minutes)

## Performance Considerations

### Memory Usage
- **Agent**: ~100KB per metrics collection cycle
- **Server**: ~1MB per agent with 1 hour of 5-second interval data
- **Automatic Cleanup**: Prevents unbounded memory growth

### CPU Impact
- **Agent**: <1% CPU overhead for 5-second collection intervals
- **Server**: Minimal impact, scales linearly with number of agents

### Network Usage
- **Metrics Data**: ~2KB per metrics snapshot
- **SSE Streaming**: Proportional to number of connected clients

## Monitoring and Alerting

### Health Indicators
- Agent connection status
- Metrics collection success rate
- Streaming lag and buffer sizes
- Memory usage trends

### Built-in Metrics
The system provides self-monitoring:
- Collection timing and success rates
- gRPC connection health
- Memory usage by component
- Task execution correlation

## Security Considerations

### Authentication
- Uses existing agent authentication tokens
- HTTP endpoints inherit server security model
- No additional authentication required for metrics

### Data Privacy
- System metrics only (no application data)
- No persistent storage (in-memory only)
- Automatic data expiration

### Network Security
- Encrypted gRPC communication (when TLS enabled)
- CORS-enabled SSE endpoints for browser clients

## Troubleshooting

### Agent Registration and Visibility

#### Immediate Agent Visibility
Agents now appear in the HTTP API immediately upon gRPC connection with basic status information:

1. **gRPC Connection**: Agents connect via gRPC and are immediately registered
2. **HTTP Visibility**: Agents appear in API immediately with connection status
3. **Data Population**: System info and metrics populate separately as they become available

```typescript
// Agents appear immediately after connection:
const agents = await client.getAllAgents();
// {
//   "agent1": {
//     "connected": true,
//     "last_seen": "2025-01-29T...",
//     "has_system_info": false,  // Initially false
//     "has_metrics": false       // Initially false
//   }
// }

// Optional: populate system info
await client.refreshSystemInfo('agent1');  // Adds system info
```

### Common Issues

#### No Metrics Available
1. Check agent connection status via `/metrics/agents`
2. Verify metrics streaming is active
3. Ensure gopsutil dependencies are available
4. Check agent logs for collection errors

#### SSE Connection Issues
1. **Event Type Mismatch**: Use `addEventListener('metrics', callback)` instead of `onmessage`
2. **Node.js Environment**: Ensure EventSource polyfill is properly configured
3. **CORS Configuration**: Verify CORS settings for browser clients
4. **Network Connectivity**: Check firewall rules and network connectivity
5. **Server Logs**: Monitor server logs for SSE client connections

```typescript
// Correct SSE setup for Node.js
import EventSource from 'eventsource';
if (typeof window === 'undefined') {
  (global as any).EventSource = EventSource;
}

const eventSource = client.streamMetrics('agent1', 3);
eventSource.addEventListener('metrics', (event) => {
  const metrics = JSON.parse(event.data);
  // Handle metrics...
});
```

#### High Memory Usage
1. Reduce metrics collection intervals
2. Decrease historical data retention
3. Monitor cleanup job execution
4. Check for agent disconnection cleanup

### Debugging

Enable debug logging to trace:
- Metrics collection timing
- gRPC message flow
- Storage operations
- SSE client connections

```bash
# Agent debug logging
go run cmd/agent/main.go -agent_id=agent1 -agent_token=token1 -v=2

# Server debug logging  
go run cmd/server/main.go -v=2
```

## Future Enhancements

### Planned Features
- **Persistent Storage**: Optional database backend for historical metrics
- **Advanced Alerting**: Threshold-based notifications and webhooks
- **Metrics Aggregation**: Cross-agent summaries and cluster-wide views
- **Custom Metrics**: Support for application-specific metrics
- **Grafana Integration**: Direct Prometheus metrics export
- **Performance Profiles**: CPU and memory profiling integration

### Extensibility
- Plugin architecture for custom metrics collectors
- Configurable data retention policies
- Advanced querying and filtering capabilities
- Multi-tenant isolation and access controls

## Examples

### Basic Usage
See the following example implementations:
- [`client/test-metrics.ts`](../client/test-metrics.ts) - Comprehensive single-agent testing
- [`client/test-multi-agent.ts`](../client/test-multi-agent.ts) - Multi-agent system testing
- [`client/test-agent-registration.ts`](../client/test-agent-registration.ts) - Agent registration demonstration
- [`client/metrics-client.ts`](../client/metrics-client.ts) - Full client library implementation

### Multi-Agent Testing
```typescript
// Test all connected agents with automatic streaming
const agents = await client.getAllAgents();
for (const agentId of Object.keys(agents.agents)) {
  console.log(`Testing agent: ${agentId}`);
  
  // Stream metrics (automatically starts streaming on agent)
  const eventSource = client.streamMetrics(agentId, 3);
  eventSource.addEventListener('metrics', (event) => {
    const metrics = JSON.parse(event.data);
    console.log(`${agentId} CPU: ${MetricsFormatter.formatPercentage(metrics.cpu.usage_percent)}`);
  });
  
  // Let it stream for a bit, then close (automatically stops streaming)
  setTimeout(() => eventSource.close(), 10000);
}
```

### Agent Registration Process
```typescript
// Agents are automatically registered upon gRPC connection
const agents = await client.getAllAgents(); // Immediately shows connected agents

// Optional: populate additional data
await client.refreshSystemInfo('agent1');  // Adds system info

// Start streaming simply by connecting via SSE (automatic streaming management)
const eventSource = client.streamMetrics('agent1', 5);
eventSource.addEventListener('metrics', handleMetrics);

// Manual control is still available if needed for special cases
// await client.startMetricsStreaming('agent1', { interval_seconds: 5 });
```

For more examples and integration guides, refer to the `/docs` directory.
