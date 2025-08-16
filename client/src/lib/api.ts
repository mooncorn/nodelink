const API_BASE_URL = 'http://localhost:8080'

export interface Node {
  id: string
  name: string
  host: string
  status: 'online' | 'offline'
  lastSeen: string
  os?: string
  arch?: string
  uptime?: string
}

export interface SystemInfo {
  os: string
  kernel: string
  arch: string
  hostname: string
  uptime: string
}

export interface SystemMetrics {
  cpu: {
    usage: number
    cores: number
    model: string
    frequency: string
  }
  memory: {
    total: number
    used: number
    free: number
    usage: number
  }
  disk: {
    total: number
    used: number
    free: number
    usage: number
  }
  network: {
    interfaces: Array<{
      name: string
      ip: string
      rx: string
      tx: string
    }>
  }
}

export interface CommandExecution {
  id: string
  command: string
  output: string
  exitCode: number
  timestamp: Date
  duration: number
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
    try {
      const response = await this.fetchWithTimeout(`${API_BASE_URL}/api/nodes`)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      return await response.json()
    } catch (error) {
      console.error('Failed to fetch nodes:', error)
      // Return mock data as fallback
      return [
        {
          id: "node-1",
          name: "Production Server",
          host: "192.168.1.100",
          status: "online",
          lastSeen: "2 minutes ago",
          os: "Ubuntu 22.04"
        },
        {
          id: "node-2", 
          name: "Development Server",
          host: "192.168.1.101",
          status: "online",
          lastSeen: "5 minutes ago",
          os: "CentOS 8"
        },
        {
          id: "node-3",
          name: "Backup Server",
          host: "192.168.1.102",
          status: "offline",
          lastSeen: "2 hours ago",
          os: "Debian 11"
        }
      ]
    }
  }

  async getNodeStatus(nodeId: string): Promise<SystemInfo & SystemMetrics> {
    try {
      const response = await this.fetchWithTimeout(`${API_BASE_URL}/api/nodes/${nodeId}/status`)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      return await response.json()
    } catch (error) {
      console.error(`Failed to fetch status for node ${nodeId}:`, error)
      // Return mock data as fallback
      return {
        os: "Ubuntu 22.04.3 LTS",
        kernel: "5.15.0-78-generic",
        arch: "x86_64",
        hostname: "prod-server-01",
        uptime: "15 days, 4 hours, 32 minutes",
        cpu: {
          usage: 45,
          cores: 8,
          model: "Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz",
          frequency: "2.30 GHz"
        },
        memory: {
          total: 16384,
          used: 10975,
          free: 5409,
          usage: 67
        },
        disk: {
          total: 1000,
          used: 230,
          free: 770,
          usage: 23
        },
        network: {
          interfaces: [
            {
              name: "eth0",
              ip: "192.168.1.100",
              rx: "1.2 GB",
              tx: "850 MB"
            }
          ]
        }
      }
    }
  }

  async executeCommand(nodeId: string, command: string): Promise<CommandExecution> {
    try {
      const response = await this.fetchWithTimeout(`${API_BASE_URL}/api/nodes/${nodeId}/command`, {
        method: 'POST',
        body: JSON.stringify({ command }),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const result = await response.json()
      return {
        id: Date.now().toString(),
        command,
        output: result.output || '',
        exitCode: result.exit_code || 0,
        timestamp: new Date(),
        duration: result.duration || 0
      }
    } catch (error) {
      console.error(`Failed to execute command on node ${nodeId}:`, error)
      // Return mock data as fallback
      return {
        id: Date.now().toString(),
        command,
        output: `Simulated output for: ${command}\n\nError: Could not connect to node ${nodeId}. Using mock data.`,
        exitCode: 1,
        timestamp: new Date(),
        duration: 100
      }
    }
  }

  // WebSocket connection for terminal
  connectTerminal(nodeId: string): WebSocket | null {
    try {
      const ws = new WebSocket(`ws://localhost:8080/api/nodes/${nodeId}/terminal`)
      return ws
    } catch (error) {
      console.error(`Failed to connect to terminal for node ${nodeId}:`, error)
      return null
    }
  }

  async getSystemInfo(nodeId: string): Promise<SystemInfo> {
    try {
      const response = await this.fetchWithTimeout(`${API_BASE_URL}/api/nodes/${nodeId}/system/info`)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      return await response.json()
    } catch (error) {
      console.error(`Failed to fetch system info for node ${nodeId}:`, error)
      // Return mock data as fallback
      return {
        hostname: nodeId,
        os: "Ubuntu 22.04 LTS",
        kernel: "5.15.0-78-generic",
        arch: "x86_64",
        uptime: "15 days, 3 hours, 42 minutes"
      }
    }
  }

  async getSystemMetrics(nodeId: string): Promise<SystemMetrics> {
    try {
      const response = await this.fetchWithTimeout(`${API_BASE_URL}/api/nodes/${nodeId}/system/metrics`)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      return await response.json()
    } catch (error) {
      console.error(`Failed to fetch system metrics for node ${nodeId}:`, error)
      // Return mock data as fallback
      return {
        cpu: {
          model: "Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz",
          cores: 8,
          usage: 45,
          frequency: "3.70GHz"
        },
        memory: {
          total: 32768,
          used: 18432,
          free: 14336,
          usage: 56
        },
        disk: {
          total: 500,
          used: 320,
          free: 180,
          usage: 64
        },
        network: {
          interfaces: [
            {
              name: "eth0",
              ip: "192.168.1.100",
              rx: "15.4 MB/s",
              tx: "8.2 MB/s"
            }
          ]
        }
      }
    }
  }
}

export const apiService = new ApiService()
