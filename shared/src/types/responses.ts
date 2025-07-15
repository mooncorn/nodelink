// Response types for different actions
export interface DockerStartResponse {
  containerId: string;
  containerName: string;
  status: string;
  ports: Array<{
    host: number;
    container: number;
    protocol: "tcp" | "udp";
  }>;
}

export interface DockerStopResponse {
  containerId: string;
  stopped: boolean;
  exitCode?: number;
}

export interface DockerListResponse {
  containers: Array<{
    id: string;
    name: string;
    image: string;
    status: string;
    ports: Array<{
      host: number;
      container: number;
      protocol: "tcp" | "udp";
    }>;
    created: Date;
  }>;
}

export interface ShellExecuteResponse {
  exitCode: number;
  output: string;
  error?: string;
  duration: number;
}

export interface SystemInfoResponse {
  platform: string;
  arch: string;
  version: string;
  hostname: string;
  uptime: number;
  memory: {
    total: number;
    free: number;
    used: number;
    usage: number;
  };
  cpu: {
    model: string;
    cores: number;
    usage: number;
  };
  disk: {
    total: number;
    free: number;
    used: number;
    usage: number;
  };
  network?: {
    interfaces: Array<{
      name: string;
      address: string;
      mac: string;
      internal: boolean;
    }>;
  };
  processes?: Array<{
    pid: number;
    name: string;
    cpu: number;
    memory: number;
  }>;
}

export interface SystemHealthResponse {
  healthy: boolean;
  checks: {
    disk?: {
      healthy: boolean;
      usage: number;
      threshold: number;
    };
    memory?: {
      healthy: boolean;
      usage: number;
      threshold: number;
    };
    cpu?: {
      healthy: boolean;
      usage: number;
      threshold: number;
    };
  };
  timestamp: Date;
}

export interface ErrorResponse {
  error: string;
  code: string;
  details?: any;
}

// Union type for all response types
export type ActionResponse =
  | DockerStartResponse
  | DockerStopResponse
  | DockerListResponse
  | ShellExecuteResponse
  | SystemInfoResponse
  | SystemHealthResponse
  | ErrorResponse;
