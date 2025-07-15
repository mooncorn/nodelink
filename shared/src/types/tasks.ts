import { ActionType, ActionPayload } from "./actions";

export type TaskStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface Task<T extends ActionType = ActionType> {
  id: string;
  type: T;
  nodeId: string;
  payload: ActionPayload<T>;
  status: TaskStatus;
  createdAt: Date;
  startedAt?: Date;
  completedAt?: Date;
  timeout?: number;
  retryCount?: number;
  maxRetries?: number;
  error?: string;
  result?: any;
}

export interface TaskUpdate {
  taskId: string;
  status: TaskStatus;
  progress?: number;
  output?: string;
  error?: string;
  result?: any;
  timestamp: Date;
}

export interface TaskExecution {
  taskId: string;
  nodeId: string;
  action: ActionType;
  payload: any;
  timeout?: number;
}

export interface TaskResult {
  taskId: string;
  success: boolean;
  result?: any;
  error?: string;
  output?: string;
  exitCode?: number;
  duration?: number;
}
