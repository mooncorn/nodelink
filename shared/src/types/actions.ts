// Action definitions for commands that can be sent to nodes
export interface ActionRegistry {
  "docker.run": DockerRunAction;
  "docker.delete": DockerDeleteAction;
  "docker.start": DockerStartAction;
  "docker.stop": DockerStopAction;
  "docker.list": DockerListAction;
  "shell.execute": ShellExecuteAction;
  "system.info": SystemInfoAction;
  "system.health": SystemHealthAction;
}

export interface DockerRunAction {
  image: string;
  containerName?: string;
  ports?: Array<{
    host: number;
    container: number;
    protocol?: "tcp" | "udp";
  }>;
  environment?: Record<string, string>;
  volumes?: Array<{
    host: string;
    container: string;
    mode?: "ro" | "rw";
  }>;
}

export interface DockerDeleteAction {
  containerId: string;
  force?: boolean; // default: false
  removeVolumes?: boolean; // default: false
}

export interface DockerStartAction {
  containerId: string;
}

export interface DockerStopAction {
  containerId: string;
  force?: boolean; // default: false
}

export interface DockerListAction {
  all?: boolean;
  filters?: Record<string, string>;
}

export interface ShellExecuteAction {
  command: string;
  cwd?: string;
  timeout?: number;
  env?: Record<string, string>;
}

export interface SystemInfoAction {
  includeMetrics?: boolean;
  includeProcesses?: boolean;
  includeNetwork?: boolean;
}

export interface SystemHealthAction {
  checkDisk?: boolean;
  checkMemory?: boolean;
  checkCpu?: boolean;
}

// Union type for all possible action types
export type ActionType = keyof ActionRegistry;
export type ActionPayload<T extends ActionType> = ActionRegistry[T];
