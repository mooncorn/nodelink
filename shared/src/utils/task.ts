import { v4 as uuidv4 } from "uuid";
import { Task, TaskStatus, TaskUpdate, TaskExecution } from "../types/tasks";
import { ActionType, ActionPayload } from "../types/actions";

export function createTask<T extends ActionType>(
  type: T,
  nodeId: string,
  payload: ActionPayload<T>,
  options: {
    timeout?: number;
    maxRetries?: number;
  } = {}
): Task<T> {
  return {
    id: uuidv4(),
    type,
    nodeId,
    payload,
    status: "pending",
    createdAt: new Date(),
    timeout: options.timeout || 30000, // 30 seconds default
    retryCount: 0,
    maxRetries: options.maxRetries || 3,
  };
}

export function createTaskExecution(task: Task): TaskExecution {
  return {
    taskId: task.id,
    nodeId: task.nodeId,
    action: task.type,
    payload: task.payload,
    timeout: task.timeout,
  };
}

export function createTaskUpdate(
  taskId: string,
  status: TaskStatus,
  options: {
    progress?: number;
    output?: string;
    error?: string;
    result?: any;
  } = {}
): TaskUpdate {
  return {
    taskId,
    status,
    timestamp: new Date(),
    ...options,
  };
}

export function updateTaskStatus(task: Task, status: TaskStatus): Task {
  const now = new Date();
  const updatedTask = { ...task, status };

  if (status === "running" && !task.startedAt) {
    updatedTask.startedAt = now;
  }

  if (
    ["completed", "failed", "cancelled"].includes(status) &&
    !task.completedAt
  ) {
    updatedTask.completedAt = now;
  }

  return updatedTask;
}

export function isTaskComplete(task: Task): boolean {
  return ["completed", "failed", "cancelled"].includes(task.status);
}

export function isTaskRunning(task: Task): boolean {
  return task.status === "running";
}

export function canRetryTask(task: Task): boolean {
  return (
    task.status === "failed" &&
    task.retryCount !== undefined &&
    task.maxRetries !== undefined &&
    task.retryCount < task.maxRetries
  );
}

export function getTaskDuration(task: Task): number | null {
  if (!task.startedAt) return null;

  const endTime = task.completedAt || new Date();
  return endTime.getTime() - task.startedAt.getTime();
}

export function isTaskExpired(task: Task): boolean {
  if (!task.timeout || !task.startedAt) return false;

  const now = new Date();
  const elapsed = now.getTime() - task.startedAt.getTime();
  return elapsed > task.timeout;
}

export function getTaskAge(task: Task): number {
  const now = new Date();
  return now.getTime() - task.createdAt.getTime();
}
