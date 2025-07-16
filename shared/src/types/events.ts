import { Task, TaskUpdate, TaskExecution, TaskResult } from "./tasks";

// Events sent from server to nodes
export interface ServerToNodeEvents {
  "task.execute": TaskExecution;
  "task.cancel": { taskId: string };
  "node.ping": { timestamp: Date };
  "node.config": { config: NodeConfig };
  "node.register.failed": { error: string };
}

// Events sent from nodes to server
export interface NodeToServerEvents {
  "node.register": NodeRegistration;
  "node.heartbeat": NodeHeartbeat;
  "task.update": TaskUpdate;
  "task.result": TaskResult;
  "task.output": TaskOutput;
  "node.pong": { timestamp: Date };
}

// Events sent from server to frontend
export interface ServerToFrontendEvents {
  "node.list": { nodes: NodeInfo[] };
  "task.created": { task: Task };
  "task.updated": { task: Task };
  "task.output": TaskOutput;
  "task.completed": { task: Task };
}

// TODO: replace with REST pattern
// Events sent from frontend to server
export interface FrontendToServerEvents {
  "task.create": {
    nodeId: string;
    type: string;
    payload: any;
  };
  "task.cancel": { taskId: string };
  "node.list": {};
}

export interface NodeRegistration {
  id: string;
  token: string;
  capabilities: string[];
  systemInfo: {
    platform: string;
    arch: string;
    version: string;
    hostname: string;
  };
}

export interface NodeHeartbeat {
  nodeId: string;
  timestamp: Date;
  status: "online" | "busy" | "offline";
  runningTasks: string[];
  systemMetrics?: {
    cpuUsage: number;
    memoryUsage: number;
    diskUsage: number;
  };
}

export interface NodeInfo {
  id: string;
  status: "online" | "busy" | "offline";
  capabilities: string[];
  systemInfo: {
    platform: string;
    arch: string;
    version: string;
    hostname: string;
  };
  systemMetrics?: { cpuUsage: number; memoryUsage: number; diskUsage: number };
  lastSeen: Date;
  runningTasks: number;
}

export interface NodeConfig {
  heartbeatInterval: number;
  maxConcurrentTasks: number;
  taskTimeout: number;
  enableMetrics: boolean;
}

export interface TaskOutput {
  taskId: string;
  output: string;
  stream: "stdout" | "stderr";
  timestamp: Date;
}
