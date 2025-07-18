// Import shared types
export type {
  ActionType,
  Task,
  TaskStatus,
  NodeRegistration,
  NodeInfo,
  ServerToFrontendEvents,
  NodeToServerEvents,
  FrontendToServerEvents,
  ServerToNodeEvents,
} from "nodelink-shared";

// Import action types for use in type aliases
import type {
  NodeInfo,
  ShellExecuteAction,
  DockerRunAction,
  DockerDeleteAction,
  DockerStartAction,
  DockerStopAction,
  DockerListAction,
  SystemInfoAction,
  SystemHealthAction,
} from "nodelink-shared";

// Frontend-specific Node interface that extends NodeInfo
export interface Node extends Omit<NodeInfo, "status" | "lastSeen"> {
  status: "online" | "offline"; // Frontend only uses online/offline
  lastSeen: string | Date; // Support both string and Date for API flexibility
  token?: string; // Frontend-specific field
  socketId?: string; // Frontend-specific field
}

// Re-export action types
export type {
  ShellExecuteAction,
  DockerRunAction,
  DockerDeleteAction,
  DockerStartAction,
  DockerStopAction,
  DockerListAction,
  SystemInfoAction,
  SystemHealthAction,
};

// Frontend-specific types that don't exist in shared
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Stats {
  nodes: number;
  onlineNodes: number;
  totalTasks: number;
  runningTasks: number;
  frontendConnections: number;
}

// Type aliases for action payloads (for backward compatibility)
export type ShellPayload = ShellExecuteAction;
export type DockerRunPayload = DockerRunAction;
export type DockerControlPayload =
  | DockerDeleteAction
  | DockerStartAction
  | DockerStopAction;

// Task options interface (extending shared Task type concept)
export interface TaskOptions {
  timeout?: number;
  priority?: number;
}
