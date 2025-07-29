import { EventSource } from "eventsource";

export interface SystemInfo {
  agent_id: string;
  timestamp: number;
  hardware: {
    cpu: {
      model: string;
      cores: number;
      threads: number;
      base_frequency_ghz: number;
      max_frequency_ghz: number;
      features: string[];
    };
    memory: {
      total_bytes: number;
      available_bytes: number;
      memory_type: string;
      speed_mhz: number;
    };
    disks: Array<{
      device: string;
      mount_point: string;
      filesystem: string;
      total_bytes: number;
      available_bytes: number;
      disk_type: string;
    }>;
    architecture: string;
    core_count: number;
    thread_count: number;
  };
  software: {
    os: {
      name: string;
      version: string;
      kernel_version: string;
      distribution: string;
    };
    hostname: string;
    uptime_seconds: number;
    packages: Array<{
      name: string;
      version: string;
      package_manager: string;
    }>;
  };
  network_interfaces: Array<{
    name: string;
    mac_address: string;
    ip_addresses: string[];
    speed_mbps: number;
    is_up: boolean;
  }>;
}

export interface MetricsSnapshot {
  agent_id: string;
  timestamp: number;
  cpu: {
    usage_percent: number;
    core_usage: number[];
    user_percent: number;
    system_percent: number;
    idle_percent: number;
    iowait_percent: number;
    temperature_celsius: number;
  };
  memory: {
    total_bytes: number;
    used_bytes: number;
    available_bytes: number;
    free_bytes: number;
    cached_bytes: number;
    buffers_bytes: number;
    usage_percent: number;
    swap_total_bytes: number;
    swap_used_bytes: number;
    swap_free_bytes: number;
    swap_usage_percent: number;
  };
  disks: Array<{
    device: string;
    mount_point: string;
    total_bytes: number;
    used_bytes: number;
    available_bytes: number;
    usage_percent: number;
    read_bytes_per_sec: number;
    write_bytes_per_sec: number;
    read_ops_per_sec: number;
    write_ops_per_sec: number;
    io_util_percent: number;
  }>;
  network_interfaces: Array<{
    interface: string;
    bytes_sent_per_sec: number;
    bytes_recv_per_sec: number;
    packets_sent_per_sec: number;
    packets_recv_per_sec: number;
    errors_in: number;
    errors_out: number;
    drops_in: number;
    drops_out: number;
  }>;
  processes: {
    total_processes: number;
    running_processes: number;
    sleeping_processes: number;
    zombie_processes: number;
    task_processes: Array<{
      task_id: string;
      pid: number;
      cpu_percent: number;
      memory_bytes: number;
      virtual_memory_bytes: number;
      threads: number;
      read_bytes: number;
      write_bytes: number;
    }>;
  };
  load: {
    load1: number;
    load5: number;
    load15: number;
  };
}

export interface MetricsHistory {
  agent_id?: string;
  query?: {
    metrics: string[];
    start_timestamp: number;
    end_timestamp: number;
    interval_seconds: number;
  };
  data?: Array<{
    timestamp: number;
    [metricName: string]: number;
  }>;
  // Actual server response format
  data_points?: Array<{
    timestamp: number;
    values: {
      [metricName: string]: number;
    };
  }>;
  query_start_timestamp?: number;
  query_end_timestamp?: number;
  total_points?: number;
}

export interface AgentSummary {
  agent_id: string;
  status: string;
  last_update: string;
  streaming_active: boolean;
  has_system_info: boolean;
  has_metrics: boolean;
}

export interface AgentsOverview {
  agents: { [agentId: string]: AgentSummary };
}

export class MetricsClient {
  constructor(private baseURL: string) {}

  // Get all agents with metrics status
  async getAllAgents(): Promise<AgentsOverview> {
    const response = await fetch(`${this.baseURL}/metrics/agents`);
    if (!response.ok) {
      throw new Error(`Failed to get agents: ${response.statusText}`);
    }
    return response.json();
  }

  // Refresh agent system information
  async refreshSystemInfo(agentId: string): Promise<{ task_id: string; message: string }> {
    const response = await fetch(`${this.baseURL}/agents/${agentId}/system/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    if (!response.ok) {
      throw new Error(`Failed to refresh system info: ${response.statusText}`);
    }
    return response.json();
  }

  // Get agent system information
  async getSystemInfo(agentId: string): Promise<SystemInfo> {
    const response = await fetch(`${this.baseURL}/agents/${agentId}/system`);
    if (!response.ok) {
      throw new Error(`Failed to get system info: ${response.statusText}`);
    }
    return response.json();
  }

  // Get current metrics snapshot
  async getCurrentMetrics(agentId: string): Promise<MetricsSnapshot> {
    const response = await fetch(`${this.baseURL}/agents/${agentId}/metrics`);
    if (!response.ok) {
      throw new Error(`Failed to get current metrics: ${response.statusText}`);
    }
    return response.json();
  }

  // Stream real-time metrics via SSE
  // Note: Server sends events with type "metrics", so use eventSource.addEventListener('metrics', callback)
  streamMetrics(agentId: string, interval: number = 5): EventSource {
    const url = `${this.baseURL}/agents/${agentId}/metrics/stream?interval=${interval}`;
    
    // Use the imported EventSource directly
    return new EventSource(url);
  }

  // Start metrics streaming
  async startMetricsStreaming(agentId: string, options: {
    interval_seconds?: number;
    metrics?: string[];
  }): Promise<{ task_id: string; message: string; interval_seconds: number }> {
    const response = await fetch(`${this.baseURL}/agents/${agentId}/metrics/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(options)
    });
    if (!response.ok) {
      throw new Error(`Failed to start metrics streaming: ${response.statusText}`);
    }
    return response.json();
  }

  // Stop metrics streaming
  async stopMetricsStreaming(agentId: string): Promise<{ task_id: string; message: string }> {
    const response = await fetch(`${this.baseURL}/agents/${agentId}/metrics/stop`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    if (!response.ok) {
      throw new Error(`Failed to stop metrics streaming: ${response.statusText}`);
    }
    return response.json();
  }

  // Get historical metrics
  async getHistoricalMetrics(agentId: string, query: {
    metrics: string[];
    startTime: number;
    endTime: number;
    maxPoints?: number;
  }): Promise<MetricsHistory> {
    const params = new URLSearchParams({
      metrics: query.metrics.join(','),
      start: query.startTime.toString(),
      end: query.endTime.toString(),
      interval: '60'
    });
    
    if (query.maxPoints) {
      params.append('max_points', query.maxPoints.toString());
    }

    const response = await fetch(`${this.baseURL}/agents/${agentId}/metrics/history?${params}`);
    if (!response.ok) {
      throw new Error(`Failed to get historical metrics: ${response.statusText}`);
    }
    return response.json();
  }
}

export class MetricsFormatter {
  static formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  static formatPercentage(value: number): string {
    return `${value.toFixed(1)}%`;
  }

  static formatUptime(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else {
      return `${minutes}m`;
    }
  }

  static formatLoadAverage(load: number): string {
    return load.toFixed(2);
  }

  static formatTimestamp(timestamp: number): string {
    return new Date(timestamp * 1000).toLocaleString();
  }

  static formatNetworkSpeed(bytesPerSec: number): string {
    const bitsPerSec = bytesPerSec * 8;
    if (bitsPerSec < 1000) return `${bitsPerSec.toFixed(0)} bps`;
    if (bitsPerSec < 1000000) return `${(bitsPerSec / 1000).toFixed(1)} Kbps`;
    if (bitsPerSec < 1000000000) return `${(bitsPerSec / 1000000).toFixed(1)} Mbps`;
    return `${(bitsPerSec / 1000000000).toFixed(1)} Gbps`;
  }

  static formatDiskIO(bytesPerSec: number): string {
    if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`;
    if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
    if (bytesPerSec < 1024 * 1024 * 1024) return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`;
    return `${(bytesPerSec / (1024 * 1024 * 1024)).toFixed(1)} GB/s`;
  }

  static formatCpuCores(coreUsage: number[]): string {
    if (!coreUsage || coreUsage.length === 0) return 'N/A';
    const avg = coreUsage.reduce((sum, usage) => sum + usage, 0) / coreUsage.length;
    return `${avg.toFixed(1)}% avg (${coreUsage.length} cores)`;
  }

  static formatMemoryDetails(memory: MetricsSnapshot['memory']): string {
    const used = MetricsFormatter.formatBytes(memory.used_bytes);
    const total = MetricsFormatter.formatBytes(memory.total_bytes);
    const cached = MetricsFormatter.formatBytes(memory.cached_bytes);
    const available = MetricsFormatter.formatBytes(memory.available_bytes);
    
    return `Used: ${used}/${total} (${memory.usage_percent.toFixed(1)}%), Available: ${available}, Cached: ${cached}`;
  }

  static formatDiskDetails(disk: MetricsSnapshot['disks'][0]): string {
    const used = MetricsFormatter.formatBytes(disk.used_bytes);
    const total = MetricsFormatter.formatBytes(disk.total_bytes);
    const readSpeed = MetricsFormatter.formatDiskIO(disk.read_bytes_per_sec);
    const writeSpeed = MetricsFormatter.formatDiskIO(disk.write_bytes_per_sec);
    
    return `${disk.device} (${disk.mount_point}): ${used}/${total} (${disk.usage_percent.toFixed(1)}%), Read: ${readSpeed}, Write: ${writeSpeed}`;
  }
}
