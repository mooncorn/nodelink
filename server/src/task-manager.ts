import { EventEmitter } from "events";
import {
  Task,
  TaskStatus,
  TaskUpdate,
  TaskResult,
  ActionType,
  ActionPayload,
  createTask,
  createTaskExecution,
  updateTaskStatus,
  isTaskComplete,
  isTaskRunning,
  isTaskExpired,
  canRetryTask,
} from "nodelink-shared";

export class TaskManager extends EventEmitter {
  private tasks: Map<string, Task> = new Map();
  private nodeConnections: Map<string, any> = new Map(); // nodeId -> socket
  private taskTimeouts: Map<string, NodeJS.Timeout> = new Map();

  constructor() {
    super();
    this.startTaskCleanup();
  }

  // Register a node connection
  registerNode(nodeId: string, socket: any): void {
    this.nodeConnections.set(nodeId, socket);

    // Setup event handlers for this node
    socket.on("task.update", (update: TaskUpdate) =>
      this.handleTaskUpdate(update)
    );
    socket.on("task.result", (result: TaskResult) =>
      this.handleTaskResult(result)
    );
    socket.on("task.output", (output: any) => this.handleTaskOutput(output));

    socket.on("disconnect", () => {
      this.nodeConnections.delete(nodeId);
      this.cancelNodeTasks(nodeId);
    });
  }

  // Create and execute a task
  createTask<T extends ActionType>(
    type: T,
    nodeId: string,
    payload: ActionPayload<T>,
    options: {
      timeout?: number;
      maxRetries?: number;
    } = {}
  ): Task<T> {
    const task = createTask(type, nodeId, payload, options);
    this.tasks.set(task.id, task);

    this.emit("task.created", task);
    this.executeTask(task);

    return task;
  }

  // Execute a task on a node
  private executeTask(task: Task): void {
    const nodeSocket = this.nodeConnections.get(task.nodeId);
    if (!nodeSocket) {
      this.failTask(task, "Node not connected");
      return;
    }

    // Update task status to running
    const runningTask = updateTaskStatus(task, "running");
    this.updateTask(runningTask);

    // Create task execution payload
    const execution = createTaskExecution(task);

    // Send task to node
    nodeSocket.emit("task.execute", execution);

    // Set timeout for task
    if (task.timeout) {
      const timeoutId = setTimeout(() => {
        this.timeoutTask(task.id);
      }, task.timeout);

      this.taskTimeouts.set(task.id, timeoutId);
    }
  }

  // Handle task update from node
  private handleTaskUpdate(update: TaskUpdate): void {
    const task = this.tasks.get(update.taskId);
    if (!task) return;

    const updatedTask = updateTaskStatus(task, update.status);

    // Update additional fields
    if (update.progress !== undefined) {
      (updatedTask as any).progress = update.progress;
    }
    if (update.output !== undefined) {
      (updatedTask as any).output = update.output;
    }
    if (update.error !== undefined) {
      updatedTask.error = update.error;
    }

    this.updateTask(updatedTask);
  }

  // Handle task result from node
  private handleTaskResult(result: TaskResult): void {
    const task = this.tasks.get(result.taskId);
    if (!task) return;

    // Clear timeout
    const timeoutId = this.taskTimeouts.get(result.taskId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      this.taskTimeouts.delete(result.taskId);
    }

    if (result.success) {
      const completedTask = updateTaskStatus(task, "completed");
      completedTask.result = result.result;
      this.updateTask(completedTask);
    } else {
      const failedTask = updateTaskStatus(task, "failed");
      failedTask.error = result.error;
      this.updateTask(failedTask);

      // Retry if possible
      if (canRetryTask(failedTask)) {
        this.retryTask(failedTask);
      }
    }
  }

  // Handle task output from node
  private handleTaskOutput(output: any): void {
    this.emit("task.output", output);
  }

  // Update a task
  private updateTask(task: Task): void {
    this.tasks.set(task.id, task);
    this.emit("task.updated", task);

    if (isTaskComplete(task)) {
      this.emit("task.completed", task);
    }
  }

  // Fail a task
  private failTask(task: Task, error: string): void {
    const failedTask = updateTaskStatus(task, "failed");
    failedTask.error = error;
    this.updateTask(failedTask);
  }

  // Timeout a task
  private timeoutTask(taskId: string): void {
    const task = this.tasks.get(taskId);
    if (!task || isTaskComplete(task)) return;

    this.cancelTask(taskId);
    this.failTask(task, "Task timed out");
  }

  // Retry a task
  private retryTask(task: Task): void {
    const retryTask = {
      ...task,
      id: task.id, // Keep same ID for retry
      status: "pending" as TaskStatus,
      retryCount: (task.retryCount || 0) + 1,
      error: undefined,
      result: undefined,
      startedAt: undefined,
      completedAt: undefined,
    };

    this.tasks.set(retryTask.id, retryTask);
    this.emit("task.retry", retryTask);

    // Execute with delay
    setTimeout(() => this.executeTask(retryTask), 1000);
  }

  // Cancel a task
  cancelTask(taskId: string): void {
    const task = this.tasks.get(taskId);
    if (!task || isTaskComplete(task)) return;

    // Clear timeout
    const timeoutId = this.taskTimeouts.get(taskId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      this.taskTimeouts.delete(taskId);
    }

    // Notify node to cancel
    const nodeSocket = this.nodeConnections.get(task.nodeId);
    if (nodeSocket) {
      nodeSocket.emit("task.cancel", { taskId });
    }

    // Update task status
    const cancelledTask = updateTaskStatus(task, "cancelled");
    this.updateTask(cancelledTask);
  }

  // Cancel all tasks for a node
  private cancelNodeTasks(nodeId: string): void {
    for (const [taskId, task] of this.tasks) {
      if (task.nodeId === nodeId && isTaskRunning(task)) {
        this.cancelTask(taskId);
      }
    }
  }

  // Get task by ID
  getTask(taskId: string): Task | undefined {
    return this.tasks.get(taskId);
  }

  // Get all tasks
  getAllTasks(): Task[] {
    return Array.from(this.tasks.values());
  }

  // Get tasks by node
  getTasksByNode(nodeId: string): Task[] {
    return Array.from(this.tasks.values()).filter(
      (task) => task.nodeId === nodeId
    );
  }

  // Get running tasks
  getRunningTasks(): Task[] {
    return Array.from(this.tasks.values()).filter(isTaskRunning);
  }

  // Cleanup completed tasks periodically
  private startTaskCleanup(): void {
    setInterval(() => {
      const cutoff = Date.now() - 60 * 60 * 1000; // 1 hour ago

      for (const [taskId, task] of this.tasks) {
        if (
          isTaskComplete(task) &&
          task.completedAt &&
          task.completedAt.getTime() < cutoff
        ) {
          this.tasks.delete(taskId);

          // Clear any remaining timeouts
          const timeoutId = this.taskTimeouts.get(taskId);
          if (timeoutId) {
            clearTimeout(timeoutId);
            this.taskTimeouts.delete(taskId);
          }
        }
      }
    }, 10 * 60 * 1000); // Run every 10 minutes
  }
}
