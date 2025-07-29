# Agent Metrics Monitoring Specification

This document specifies the agent metrics monitoring system for the nodelink project, providing real-time performance metrics and static system information.

## Overview

The agent metrics system provides:
- **Real-time Metrics**: CPU usage, memory usage, disk I/O, network stats
- **System Information**: Static hardware and OS details
- **Task Metrics**: Per-task resource consumption
- **Historical Data**: Time-series metrics with configurable retention
- **Multiple Access Methods**: gRPC streaming, HTTP REST API, SSE
- **Configurable Collection**: Adjustable collection intervals and metrics

## Architecture

```
┌─────────────────┐    gRPC/HTTP    ┌──────────────────┐
│     Client      │ ←──────────────→ │     Server       │
│  Metrics UI     │                 │  Metrics Router  │
└─────────────────┘                 └──────────────────┘
                                             │
                                             ▼
                                    ┌──────────────────┐
                                    │   Task Manager   │
                                    │  Metrics Store   │
                                    └──────────────────┘
                                             │
                                             ▼ gRPC Stream
                                    ┌──────────────────┐
                                    │      Agent       │
                                    │  Metrics         │
                                    │  Collector       │
                                    └──────────────────┘
                                             │
                                             ▼
                                    ┌──────────────────┐
                                    │  System Monitor  │
                                    │  - CPU/Memory    │
                                    │  - Disk/Network  │
                                    │  - Process Stats │
                                    └──────────────────┘
```

## Protocol Buffer Extensions

### 1. Metrics Request Types

```protobuf
// Add to TaskRequest
message TaskRequest {
  string agent_id = 1;
  string task_id = 2;
  oneof task {
    ShellExecute shell_execute = 3;
    DockerOperation docker_operation = 4;
    TaskCancel task_cancel = 5;
    MetricsRequest metrics_request = 6;    // New metrics request
  }
}

// Metrics request variants
message MetricsRequest {
  oneof request_type {
    SystemInfoRequest system_info = 1;        // One-time system info
    MetricsStreamRequest stream_request = 2;  // Start/stop metrics streaming
    MetricsQueryRequest query_request = 3;    // Historical metrics query
  }
}

message SystemInfoRequest {
  bool include_hardware = 1;    // Include CPU, memory, disk specs
  bool include_software = 2;    // Include OS, kernel, installed packages
  bool include_network = 3;     // Include network interfaces
}

message MetricsStreamRequest {
  enum Action {
    START = 0;
    STOP = 1;
    UPDATE_INTERVAL = 2;
  }
  Action action = 1;
  uint32 interval_seconds = 2;  // Collection interval (default: 5s)
  repeated string metrics = 3;  // Specific metrics to collect (empty = all)
}

message MetricsQueryRequest {
  repeated string metrics = 1;        // Metrics to query
  uint64 start_timestamp = 2;         // Query start time (Unix timestamp)
  uint64 end_timestamp = 3;           // Query end time (Unix timestamp)
  uint32 max_points = 4;              // Maximum data points to return
}
```

### 2. Metrics Response Types

```protobuf
// Add to TaskResponse
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
    MetricsResponse metrics_response = 9;     // New metrics response
  }
}

message MetricsResponse {
  oneof response_type {
    SystemInfoResponse system_info = 1;
    MetricsDataResponse metrics_data = 2;
    MetricsQueryResponse query_response = 3;
  }
}
```

### 3. System Information Response

```protobuf
message SystemInfoResponse {
  SystemHardware hardware = 1;
  SystemSoftware software = 2;
  repeated NetworkInterface network_interfaces = 3;
  uint64 timestamp = 4;
}

message SystemHardware {
  CpuInfo cpu = 1;
  MemoryInfo memory = 2;
  repeated DiskInfo disks = 3;
  string architecture = 4;        // x86_64, arm64, etc.
  uint32 core_count = 5;
  uint32 thread_count = 6;
}

message CpuInfo {
  string model = 1;               // CPU model name
  uint32 cores = 2;               // Physical cores
  uint32 threads = 3;             // Logical cores
  double base_frequency_ghz = 4;  // Base frequency
  double max_frequency_ghz = 5;   // Max boost frequency
  repeated string features = 6;   // CPU features/flags
}

message MemoryInfo {
  uint64 total_bytes = 1;         // Total physical memory
  uint64 available_bytes = 2;     // Available memory
  string memory_type = 3;         // DDR4, DDR5, etc.
  uint32 speed_mhz = 4;          // Memory speed
}

message DiskInfo {
  string device = 1;              // /dev/sda1, C:, etc.
  string mount_point = 2;         // /, /home, C:\, etc.
  string filesystem = 3;          // ext4, ntfs, xfs, etc.
  uint64 total_bytes = 4;         // Total disk space
  uint64 available_bytes = 5;     // Available disk space
  string disk_type = 6;          // SSD, HDD, NVMe
}

message NetworkInterface {
  string name = 1;                // eth0, wlan0, etc.
  string mac_address = 2;         // MAC address
  repeated string ip_addresses = 3; // IPv4/IPv6 addresses
  uint64 speed_mbps = 4;         // Interface speed
  bool is_up = 5;                // Interface status
}

message SystemSoftware {
  OperatingSystem os = 1;
  string hostname = 2;
  uint64 uptime_seconds = 3;
  repeated InstalledPackage packages = 4;
}

message OperatingSystem {
  string name = 1;                // Linux, Windows, macOS
  string version = 2;             // Ubuntu 22.04, Windows 11, etc.
  string kernel_version = 3;      // Kernel version
  string distribution = 4;        // ubuntu, centos, debian (Linux only)
}

message InstalledPackage {
  string name = 1;
  string version = 2;
  string package_manager = 3;     // apt, yum, brew, etc.
}
```

### 4. Real-time Metrics Data

```protobuf
message MetricsDataResponse {
  uint64 timestamp = 1;           // Unix timestamp
  CpuMetrics cpu = 2;
  MemoryMetrics memory = 3;
  repeated DiskMetrics disks = 4;
  repeated NetworkMetrics network_interfaces = 5;
  ProcessMetrics processes = 6;
  LoadMetrics load = 7;
}

message CpuMetrics {
  double usage_percent = 1;       // Overall CPU usage (0-100)
  repeated double core_usage = 2; // Per-core usage
  double user_percent = 3;        // User space CPU time
  double system_percent = 4;      // Kernel space CPU time
  double idle_percent = 5;        // Idle time
  double iowait_percent = 6;      // I/O wait time
  double temperature_celsius = 7; // CPU temperature (if available)
}

message MemoryMetrics {
  uint64 total_bytes = 1;
  uint64 used_bytes = 2;
  uint64 available_bytes = 3;
  uint64 free_bytes = 4;
  uint64 cached_bytes = 5;
  uint64 buffers_bytes = 6;
  double usage_percent = 7;       // used/total * 100
  
  // Swap metrics
  uint64 swap_total_bytes = 8;
  uint64 swap_used_bytes = 9;
  uint64 swap_free_bytes = 10;
  double swap_usage_percent = 11;
}

message DiskMetrics {
  string device = 1;
  string mount_point = 2;
  uint64 total_bytes = 3;
  uint64 used_bytes = 4;
  uint64 available_bytes = 5;
  double usage_percent = 6;
  
  // I/O metrics
  uint64 read_bytes_per_sec = 7;
  uint64 write_bytes_per_sec = 8;
  uint64 read_ops_per_sec = 9;
  uint64 write_ops_per_sec = 10;
  double io_util_percent = 11;    // I/O utilization
}

message NetworkMetrics {
  string interface = 1;
  uint64 bytes_sent_per_sec = 2;
  uint64 bytes_recv_per_sec = 3;
  uint64 packets_sent_per_sec = 4;
  uint64 packets_recv_per_sec = 5;
  uint64 errors_in = 6;
  uint64 errors_out = 7;
  uint64 drops_in = 8;
  uint64 drops_out = 9;
}

message ProcessMetrics {
  uint32 total_processes = 1;
  uint32 running_processes = 2;
  uint32 sleeping_processes = 3;
  uint32 zombie_processes = 4;
  repeated TaskProcessMetrics task_processes = 5; // Per-task metrics
}

message TaskProcessMetrics {
  string task_id = 1;
  uint32 pid = 2;
  double cpu_percent = 3;
  uint64 memory_bytes = 4;
  uint64 virtual_memory_bytes = 5;
  uint32 threads = 6;
  uint64 read_bytes = 7;
  uint64 write_bytes = 8;
}

message LoadMetrics {
  double load1 = 1;               // 1-minute load average
  double load5 = 2;               // 5-minute load average
  double load15 = 3;              // 15-minute load average
}
```

### 5. Historical Metrics Query

```protobuf
message MetricsQueryResponse {
  repeated MetricsTimePoint data_points = 1;
  uint64 query_start_timestamp = 2;
  uint64 query_end_timestamp = 3;
  uint32 total_points = 4;
  bool truncated = 5;             // True if results were limited by max_points
}

message MetricsTimePoint {
  uint64 timestamp = 1;
  map<string, double> values = 2; // metric_name -> value
}
```

## HTTP API Endpoints

### 1. Agent System Information

```http
GET /agents/{agentId}/system

Response:
{
  "agent_id": "agent1",
  "timestamp": 1690538400000,
  "hardware": {
    "cpu": {
      "model": "Intel Core i7-12700K",
      "cores": 12,
      "threads": 20,
      "base_frequency_ghz": 3.6,
      "max_frequency_ghz": 5.0,
      "features": ["sse4", "avx2", "avx512"]
    },
    "memory": {
      "total_bytes": 34359738368,
      "memory_type": "DDR5",
      "speed_mhz": 5600
    },
    "disks": [
      {
        "device": "/dev/nvme0n1p1",
        "mount_point": "/",
        "filesystem": "ext4",
        "total_bytes": 1000204886016,
        "available_bytes": 750153664512,
        "disk_type": "NVMe"
      }
    ],
    "architecture": "x86_64",
    "core_count": 12,
    "thread_count": 20
  },
  "software": {
    "os": {
      "name": "Linux",
      "version": "Ubuntu 22.04.3 LTS",
      "kernel_version": "6.2.0-39-generic",
      "distribution": "ubuntu"
    },
    "hostname": "worker-node-01",
    "uptime_seconds": 1234567
  },
  "network_interfaces": [
    {
      "name": "eth0",
      "mac_address": "aa:bb:cc:dd:ee:ff",
      "ip_addresses": ["192.168.1.100", "fe80::1234:5678:90ab:cdef"],
      "speed_mbps": 1000,
      "is_up": true
    }
  ]
}
```

### 2. Current Metrics Snapshot

```http
GET /agents/{agentId}/metrics

Response:
{
  "agent_id": "agent1",
  "timestamp": 1690538400000,
  "cpu": {
    "usage_percent": 25.4,
    "core_usage": [23.1, 28.7, 24.2, 26.8],
    "user_percent": 18.3,
    "system_percent": 7.1,
    "idle_percent": 74.6,
    "iowait_percent": 0.0,
    "temperature_celsius": 45.2
  },
  "memory": {
    "total_bytes": 34359738368,
    "used_bytes": 12884901888,
    "available_bytes": 21474836480,
    "usage_percent": 37.5,
    "swap_total_bytes": 2147483648,
    "swap_used_bytes": 0,
    "swap_usage_percent": 0.0
  },
  "disks": [
    {
      "device": "/dev/nvme0n1p1",
      "mount_point": "/",
      "usage_percent": 25.0,
      "read_bytes_per_sec": 1048576,
      "write_bytes_per_sec": 524288,
      "io_util_percent": 5.2
    }
  ],
  "load": {
    "load1": 1.25,
    "load5": 1.15,
    "load15": 1.05
  }
}
```

### 3. Historical Metrics Query

```http
GET /agents/{agentId}/metrics/history?metrics=cpu.usage_percent,memory.usage_percent&start=1690537800&end=1690538400&interval=60

Response:
{
  "agent_id": "agent1",
  "query": {
    "metrics": ["cpu.usage_percent", "memory.usage_percent"],
    "start_timestamp": 1690537800,
    "end_timestamp": 1690538400,
    "interval_seconds": 60
  },
  "data": [
    {
      "timestamp": 1690537800,
      "cpu.usage_percent": 23.4,
      "memory.usage_percent": 35.2
    },
    {
      "timestamp": 1690537860,
      "cpu.usage_percent": 25.1,
      "memory.usage_percent": 36.8
    }
  ]
}
```

### 4. Real-time Metrics Streaming

```http
GET /agents/{agentId}/metrics/stream?interval=5
Accept: text/event-stream

Events:
event: metrics
data: {
  "timestamp": 1690538400000,
  "cpu": {
    "usage_percent": 25.4,
    "temperature_celsius": 45.2
  },
  "memory": {
    "usage_percent": 37.5
  }
}

event: metrics
data: {
  "timestamp": 1690538405000,
  "cpu": {
    "usage_percent": 28.1,
    "temperature_celsius": 46.1
  },
  "memory": {
    "usage_percent": 38.2
  }
}
```

### 5. All Agents Metrics

```http
GET /metrics/agents

Response:
{
  "agents": [
    {
      "agent_id": "agent1",
      "status": "connected",
      "last_metrics_update": 1690538400000,
      "metrics": {
        "cpu": {"usage_percent": 25.4},
        "memory": {"usage_percent": 37.5}
      }
    },
    {
      "agent_id": "agent2",
      "status": "disconnected",
      "last_metrics_update": 1690538200000,
      "metrics": null
    }
  ]
}
```

## Agent Implementation

### 1. Metrics Collector

```go
type MetricsCollector struct {
    interval    time.Duration
    stopCh      chan struct{}
    client      *grpc.TaskClient
    collecting  bool
    mu          sync.RWMutex
}

func (mc *MetricsCollector) Start(interval time.Duration) {
    mc.mu.Lock()
    defer mc.mu.Unlock()
    
    if mc.collecting {
        return
    }
    
    mc.interval = interval
    mc.stopCh = make(chan struct{})
    mc.collecting = true
    
    go mc.collectLoop()
}

func (mc *MetricsCollector) collectLoop() {
    ticker := time.NewTicker(mc.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            metrics := mc.collectMetrics()
            mc.sendMetrics(metrics)
        case <-mc.stopCh:
            return
        }
    }
}
```

### 2. System Information Collection

```go
func (mc *MetricsCollector) collectSystemInfo() *pb.SystemInfoResponse {
    return &pb.SystemInfoResponse{
        Hardware:          mc.collectHardwareInfo(),
        Software:          mc.collectSoftwareInfo(),
        NetworkInterfaces: mc.collectNetworkInfo(),
        Timestamp:         uint64(time.Now().Unix()),
    }
}

func (mc *MetricsCollector) collectHardwareInfo() *pb.SystemHardware {
    // Implementation using system calls and /proc filesystem
}
```

### 3. Real-time Metrics Collection

```go
func (mc *MetricsCollector) collectMetrics() *pb.MetricsDataResponse {
    return &pb.MetricsDataResponse{
        Timestamp: uint64(time.Now().Unix()),
        Cpu:       mc.collectCpuMetrics(),
        Memory:    mc.collectMemoryMetrics(),
        Disks:     mc.collectDiskMetrics(),
        Network:   mc.collectNetworkMetrics(),
        Processes: mc.collectProcessMetrics(),
        Load:      mc.collectLoadMetrics(),
    }
}
```

## Server Implementation

### 1. Metrics Storage

```go
type MetricsStore struct {
    mu          sync.RWMutex
    agentData   map[string]*AgentMetrics
    retention   time.Duration
}

type AgentMetrics struct {
    SystemInfo      *pb.SystemInfoResponse
    CurrentMetrics  *pb.MetricsDataResponse
    HistoricalData  *TimeSeriesStore
    LastUpdate      time.Time
    StreamingActive bool
}
```

### 2. HTTP Handlers

```go
func (h *MetricsHandler) GetAgentSystem(w http.ResponseWriter, r *http.Request) {
    agentID := mux.Vars(r)["agentId"]
    
    systemInfo, err := h.metricsStore.GetSystemInfo(agentID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    
    json.NewEncoder(w).Encode(systemInfo)
}
```

### 3. SSE Streaming

```go
func (h *MetricsHandler) StreamAgentMetrics(w http.ResponseWriter, r *http.Request) {
    agentID := mux.Vars(r)["agentId"]
    interval := getQueryParam(r, "interval", "5")
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Stream metrics
    h.streamMetricsToClient(w, agentID, interval)
}
```

## Client Libraries

### 1. TypeScript Client

```typescript
export class MetricsClient {
    constructor(private baseURL: string) {}
    
    // Get system information
    async getSystemInfo(agentId: string): Promise<SystemInfo> {
        const response = await fetch(`${this.baseURL}/agents/${agentId}/system`);
        return response.json();
    }
    
    // Get current metrics snapshot
    async getCurrentMetrics(agentId: string): Promise<MetricsSnapshot> {
        const response = await fetch(`${this.baseURL}/agents/${agentId}/metrics`);
        return response.json();
    }
    
    // Stream real-time metrics
    streamMetrics(agentId: string, interval: number = 5): EventSource {
        return new EventSource(`${this.baseURL}/agents/${agentId}/metrics/stream?interval=${interval}`);
    }
    
    // Query historical metrics
    async getHistoricalMetrics(agentId: string, query: MetricsQuery): Promise<MetricsHistory> {
        const params = new URLSearchParams({
            metrics: query.metrics.join(','),
            start: query.startTime.toString(),
            end: query.endTime.toString(),
            interval: query.interval.toString()
        });
        
        const response = await fetch(`${this.baseURL}/agents/${agentId}/metrics/history?${params}`);
        return response.json();
    }
}
```

### 2. React Components

```typescript
export function AgentMetricsDashboard({ agentId }: { agentId: string }) {
    const [metrics, setMetrics] = useState<MetricsSnapshot | null>(null);
    const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
    
    useEffect(() => {
        const client = new MetricsClient('/api');
        
        // Load system info once
        client.getSystemInfo(agentId).then(setSystemInfo);
        
        // Stream real-time metrics
        const eventSource = client.streamMetrics(agentId, 5);
        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            setMetrics(data);
        };
        
        return () => eventSource.close();
    }, [agentId]);
    
    return (
        <div className="metrics-dashboard">
            <SystemInfoCard info={systemInfo} />
            <CpuUsageChart metrics={metrics?.cpu} />
            <MemoryUsageChart metrics={metrics?.memory} />
            <DiskUsageChart metrics={metrics?.disks} />
        </div>
    );
}
```

## Configuration

### 1. Agent Configuration

```yaml
metrics:
  enabled: true
  collection_interval: 5s
  retention_period: 24h
  collected_metrics:
    - cpu
    - memory
    - disk
    - network
    - load
  exclude_metrics:
    - cpu.temperature  # May not be available on all systems
  process_tracking:
    enabled: true
    track_task_processes: true
```

### 2. Server Configuration

```yaml
metrics:
  storage:
    retention_period: 7d
    max_points_per_query: 10000
    compression_enabled: true
  api:
    enable_sse_streaming: true
    max_concurrent_streams: 100
    rate_limiting:
      requests_per_minute: 60
```

## Security Considerations

### 1. Authentication
- Same token-based authentication as existing gRPC communication
- HTTP endpoints require same authentication headers
- SSE streams authenticated via query parameters or headers

### 2. Authorization
- Agents can only report their own metrics
- Clients can only access metrics for agents they have permission to manage
- Admin users can access all agent metrics

### 3. Privacy
- System information may contain sensitive details (IP addresses, hostnames)
- Option to anonymize or exclude sensitive fields
- Configurable data retention policies

## Performance Impact

### 1. Agent Overhead
- **CPU**: ~1-2% additional CPU usage for metrics collection
- **Memory**: ~10-20MB for metrics buffers and historical data
- **Network**: ~1-5KB/second additional gRPC traffic

### 2. Server Resources
- **Storage**: ~1MB per agent per day for compressed metrics
- **Memory**: ~1-5MB per connected agent for in-memory metrics
- **CPU**: Minimal overhead for metrics aggregation and serving

### 3. Optimization Strategies
- Configurable collection intervals (trade-off between granularity and overhead)
- Selective metric collection (disable unused metrics)
- Data compression for historical storage
- Rate limiting for HTTP API access

## Future Enhancements

### 1. Advanced Features
- **Alerting**: Threshold-based alerts for metric values
- **Dashboards**: Pre-built visualization dashboards
- **Correlations**: Automatic correlation between metrics and task performance
- **Predictions**: ML-based capacity planning and anomaly detection

### 2. Integration
- **Prometheus**: Export metrics in Prometheus format
- **Grafana**: Native Grafana dashboard templates
- **InfluxDB**: Optional time-series database backend
- **Webhooks**: HTTP callbacks for metric threshold violations

### 3. Extended Metrics
- **Container Metrics**: Docker container resource usage
- **GPU Metrics**: NVIDIA/AMD GPU utilization
- **Custom Metrics**: User-defined application metrics
- **Environmental**: Temperature sensors, power consumption
