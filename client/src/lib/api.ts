const API_BASE_URL = import.meta.env.VITE_API_URL

// Agent/Node interfaces based on server types
export interface Node {
  agent_id: string
  status: 'online' | 'offline'
  last_seen: string
  connected_at?: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface AgentsResponse {
  agents: Node[]
  stats: {
    total: number
    online: number
    offline: number
  }
  total: number
}

// System Info interface based on protobuf SystemInfo
export interface SystemInfo {
  hostname: string
  platform: string
  arch: string
  os_version: string
  cpu_count: number
  total_memory: number
  network_interfaces: string[]
  kernel_version: string
  uptime_seconds: number
}

// System Metrics interfaces based on protobuf SystemMetrics
export interface MemoryMetrics {
  total: number
  available: number
  used: number
  used_percent: number
  free: number
  cached: number
  buffers: number
}

export interface DiskMetrics {
  device: string
  mountpoint: string
  filesystem: string
  total: number
  used: number
  free: number
  used_percent: number
}

export interface NetworkMetrics {
  interface: string
  bytes_sent: number
  bytes_recv: number
  packets_sent: number
  packets_recv: number
  errors_in: number
  errors_out: number
  drops_in: number
  drops_out: number
}

export interface ProcessMetrics {
  pid: number
  name: string
  cpu_percent: number
  memory_rss: number
  memory_vms: number
  status: string
  create_time: number
  num_threads: number
}

export interface SystemMetrics {
  cpu_usage_percent: number
  memory: MemoryMetrics
  disks: DiskMetrics[]
  network_interfaces: NetworkMetrics[]
  processes: ProcessMetrics[]
  timestamp: number
  load_average_1m: number
  load_average_5m: number
  load_average_15m: number
}

// Terminal interfaces
export interface TerminalSession {
  session_id: string
  agent_id: string
  shell: string
  working_dir: string
  status: 'active' | 'inactive' | 'closed'
  created_at: string
}

export interface CreateTerminalSessionRequest {
  agent_id: string
  shell?: string
  working_dir?: string
  env?: Record<string, string>
}

export interface ExecuteTerminalCommandRequest {
  command: string
}

// Command execution interfaces
export interface CommandExecution {
  id: string
  command: string
  output: string
  exitCode: number
  timestamp: Date
  duration: number
}

export interface ExecuteCommandRequest {
  agent_id: string
  command: string
  args?: string[]
  env?: Record<string, string>
  working_dir?: string
  timeout_seconds?: number
}

export interface ExecuteCommandResponse {
  request_id: string
  exit_code: number
  stdout: string
  stderr: string
  error?: string
  timeout: boolean
}

// SSE Event interfaces
export interface StatusChangeEvent {
  agent_id: string
  old_status: 'online' | 'offline'
  new_status: 'online' | 'offline'
  timestamp: string
  agent: Node
}

export interface TerminalOutputEvent {
  session_id: string
  command_id: string
  output: string
  error?: string
  is_final: boolean
  exit_code?: number
}

export interface MetricsStreamEvent {
  agent_id: string
  system_info?: SystemInfo
  metrics?: SystemMetrics
  timestamp: number
}

class ApiService {
  private async fetchWithTimeout(url: string, options: RequestInit = {}, timeout = 5000) {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeout)
    
    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      })
      
      clearTimeout(timeoutId)
      return response
    } catch (error) {
      clearTimeout(timeoutId)
      throw error
    }
  }

  async getNodes(): Promise<Node[]> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/agents`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data: AgentsResponse = await response.json()
    return data.agents
  }

  async getNode(agentId: string): Promise<Node> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/agents/${agentId}`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data = await response.json()
    return data.agent
  }

  async getSystemInfo(agentId: string): Promise<SystemInfo> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/metrics/${agentId}`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data = await response.json()
    return data.system_info
  }

  async executeCommand(agentId: string, command: string, args?: string[]): Promise<ExecuteCommandResponse> {
    const requestBody: ExecuteCommandRequest = {
      agent_id: agentId,
      command,
      args: args || [],
      timeout_seconds: 30
    }

    const response = await this.fetchWithTimeout(`${API_BASE_URL}/commands`, {
      method: 'POST',
      body: JSON.stringify(requestBody),
    })
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    return await response.json()
  }

  // Terminal session management
  async createTerminalSession(request: CreateTerminalSessionRequest): Promise<TerminalSession> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/terminals`, {
      method: 'POST',
      body: JSON.stringify(request),
    })
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    return await response.json()
  }

  async getTerminalSessions(): Promise<TerminalSession[]> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/terminals`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data = await response.json()
    return data.sessions
  }

  async executeTerminalCommand(sessionId: string, command: string): Promise<{ command_id: string; status: string }> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/terminals/${sessionId}/command`, {
      method: 'POST',
      body: JSON.stringify({ command }),
    })
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    return await response.json()
  }

  async closeTerminalSession(sessionId: string): Promise<void> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}/terminals/${sessionId}`, {
      method: 'DELETE',
    })
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
  }

  // SSE Connections
  connectToAgentStatusEvents(): EventSource | null {
    try {
      const eventSource = new EventSource(`${API_BASE_URL}/agents/events`)
      return eventSource
    } catch (error) {
      console.error('Failed to connect to agent status events:', error)
      return null
    }
  }

  connectToSpecificAgentEvents(agentId: string): EventSource | null {
    try {
      const eventSource = new EventSource(`${API_BASE_URL}/agents/${agentId}/events`)
      return eventSource
    } catch (error) {
      console.error(`Failed to connect to agent ${agentId} events:`, error)
      return null
    }
  }

  connectToTerminalStream(sessionId: string): EventSource | null {
    try {
      const eventSource = new EventSource(`${API_BASE_URL}/terminals/${sessionId}/stream?user_id=default-user`)
      return eventSource
    } catch (error) {
      console.error(`Failed to connect to terminal stream for session ${sessionId}:`, error)
      return null
    }
  }

  connectToMetricsStream(agentId: string): EventSource | null {
    try {
      const eventSource = new EventSource(`${API_BASE_URL}/metrics/${agentId}/stream`)
      return eventSource
    } catch (error) {
      console.error(`Failed to connect to metrics stream for agent ${agentId}:`, error)
      return null
    }
  }

  // Legacy methods for backward compatibility (updated to use new API)
  async getNodeStatus(nodeId: string): Promise<SystemInfo & { metrics?: SystemMetrics }> {
    const systemInfo = await this.getSystemInfo(nodeId)
    return systemInfo
  }

  async getSystemMetrics(_nodeId: string): Promise<SystemMetrics | null> {
    // Note: HTTP endpoint only provides system_info, metrics are only available via SSE
    // This method returns null to indicate metrics should be fetched via SSE
    return null
  }
}

export const apiService = new ApiService()
