import { EventEmitter } from "events";
import { io } from "socket.io-client";
import * as os from "os";
import {
  NodeRegistration,
  NodeHeartbeat,
  NodeConfig,
  TaskExecution,
  TaskResult,
  TaskUpdate,
  createTaskUpdate,
} from "nodelink-shared";

import { ActionExecutor } from "./action-executor";

export class NodeAgent extends EventEmitter {
  private socket: any;
  private executor: ActionExecutor;
  private config: NodeConfig = {
    heartbeatInterval: 30000,
    maxConcurrentTasks: 5,
    taskTimeout: 300000,
    enableMetrics: true,
  };
  private heartbeatTimer?: NodeJS.Timeout;
  private runningTasks: Map<string, any> = new Map();
  private connected = false;
  private authenticationFailed = false;

  constructor(
    private nodeId: string,
    private token: string,
    private serverUrl: string = "https://localhost:8443"
  ) {
    super();

    this.executor = new ActionExecutor(nodeId);
    this.socket = io(this.serverUrl, {
      rejectUnauthorized: false, // For self-signed certificates
    });

    this.setupSocketHandlers();
  }

  private setupSocketHandlers(): void {
    this.socket.on("connect", () => {
      console.log(`Node ${this.nodeId} connected to server`);
      this.connected = true;
      this.register();
    });

    this.socket.on("disconnect", (reason: string) => {
      console.log(`Node ${this.nodeId} disconnected: ${reason}`);
      this.connected = false;
      this.stopHeartbeat();

      if (this.authenticationFailed) {
        // Authentication failed, stop trying to reconnect
        console.error(`Node ${this.nodeId} authentication failed`);
        this.stop();
        return;
      }

      if (reason === "io server disconnect") {
        // Reconnect if the server disconnected us
        this.socket.connect();
      }
    });

    this.socket.on("node.register.failed", (error: { error: string }) => {
      console.error(`Node ${this.nodeId} registration failed:`, error);
      this.authenticationFailed = true;
    });

    this.socket.on("connect_error", (error: any) => {
      console.error(`Node ${this.nodeId} connection error:`, error);
    });

    this.socket.on("task.execute", (execution: TaskExecution) => {
      this.executeTask(execution);
    });

    this.socket.on("task.cancel", (data: { taskId: string }) => {
      this.cancelTask(data.taskId);
    });

    this.socket.on("node.ping", (data: { timestamp: Date }) => {
      this.socket.emit("node.pong", { timestamp: data.timestamp });
    });

    this.socket.on("node.config", (data: { config: NodeConfig }) => {
      this.updateConfig(data.config);
    });
  }

  private register(): void {
    const registration: NodeRegistration = {
      id: this.nodeId,
      token: this.token,
      capabilities: this.getCapabilities(),
      systemInfo: {
        platform: os.platform(),
        arch: os.arch(),
        version: os.release(),
        hostname: os.hostname(),
      },
    };

    this.socket.emit("node.register", registration);
  }

  private getCapabilities(): string[] {
    const capabilities = ["shell.execute", "system.info", "system.health"];

    // Check if Docker is available using dockerode
    this.checkDockerAvailability()
      .then((available) => {
        if (available) {
          capabilities.push(
            "docker.run",
            "docker.delete",
            "docker.start",
            "docker.stop",
            "docker.list"
          );
        }
      })
      .catch(() => {
        console.warn("Docker not available on this node");
      });

    return capabilities;
  }

  private async checkDockerAvailability(): Promise<boolean> {
    try {
      // Import dockerode dynamically to avoid issues if Docker is not available
      const { DockerActions } = await import("./docker-actions");
      const dockerActions = new DockerActions();
      return await dockerActions.isDockerAvailable();
    } catch (error) {
      return false;
    }
  }

  private updateConfig(config: NodeConfig): void {
    this.config = { ...this.config, ...config };

    // Restart heartbeat with new interval
    this.stopHeartbeat();
    this.startHeartbeat();

    console.log(`Node ${this.nodeId} configuration updated:`, this.config);
  }

  private startHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
    }

    this.heartbeatTimer = setInterval(() => {
      this.sendHeartbeat();
    }, this.config.heartbeatInterval);

    // Send initial heartbeat
    this.sendHeartbeat();
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = undefined;
    }
  }

  private sendHeartbeat(): void {
    if (!this.connected) return;

    const heartbeat: NodeHeartbeat = {
      nodeId: this.nodeId,
      timestamp: new Date(),
      status: this.runningTasks.size > 0 ? "busy" : "online",
      runningTasks: Array.from(this.runningTasks.keys()),
    };

    // Add system metrics if enabled
    if (this.config.enableMetrics) {
      heartbeat.systemMetrics = {
        cpuUsage: 0, // TODO: implement CPU usage calculation
        memoryUsage: ((os.totalmem() - os.freemem()) / os.totalmem()) * 100,
        diskUsage: 0, // TODO: implement disk usage calculation
      };
    }

    this.socket.emit("node.heartbeat", heartbeat);
  }

  private async executeTask(execution: TaskExecution): Promise<void> {
    const { taskId } = execution;

    // Check if we're at max capacity
    if (this.runningTasks.size >= this.config.maxConcurrentTasks) {
      const result: TaskResult = {
        taskId,
        success: false,
        error: `Node at maximum capacity (${this.config.maxConcurrentTasks} tasks)`,
      };

      this.socket.emit("task.result", result);
      return;
    }

    // Add to running tasks
    this.runningTasks.set(taskId, execution);

    // Send task started update
    const startUpdate = createTaskUpdate(taskId, "running");
    this.socket.emit("task.update", startUpdate);

    try {
      // Execute the task
      const result = await this.executor.executeTask(execution);

      // Send result
      this.socket.emit("task.result", result);

      // Send completion update
      const completeUpdate = createTaskUpdate(
        taskId,
        result.success ? "completed" : "failed"
      );
      this.socket.emit("task.update", completeUpdate);
    } catch (error) {
      // Send error result
      const result: TaskResult = {
        taskId,
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };

      this.socket.emit("task.result", result);

      // Send failure update
      const failureUpdate = createTaskUpdate(taskId, "failed", {
        error: result.error,
      });
      this.socket.emit("task.update", failureUpdate);
    } finally {
      // Remove from running tasks
      this.runningTasks.delete(taskId);
    }
  }

  private cancelTask(taskId: string): void {
    if (this.runningTasks.has(taskId)) {
      this.executor.cancelTask(taskId);
      this.runningTasks.delete(taskId);

      // Send cancellation update
      const cancelUpdate = createTaskUpdate(taskId, "cancelled");
      this.socket.emit("task.update", cancelUpdate);

      console.log(`Task ${taskId} cancelled`);
    }
  }

  // Public methods
  start(): void {
    console.log(`Starting Node Agent: ${this.nodeId}`);
    this.socket.connect();
  }

  stop(): void {
    console.log(`Stopping Node Agent: ${this.nodeId}`);
    this.connected = false;
    this.stopHeartbeat();
    this.socket.disconnect();
  }

  isConnected(): boolean {
    return this.connected;
  }

  getRunningTasks(): string[] {
    return Array.from(this.runningTasks.keys());
  }

  getStats() {
    return {
      nodeId: this.nodeId,
      connected: this.connected,
      runningTasks: this.runningTasks.size,
      maxConcurrentTasks: this.config.maxConcurrentTasks,
      capabilities: this.getCapabilities(),
    };
  }
}
