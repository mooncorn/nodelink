import { spawn, exec } from "child_process";
import { promisify } from "util";
import * as os from "os";
import * as fs from "fs";
import {
  TaskExecution,
  TaskResult,
  TaskOutput,
  ActionType,
  ShellExecuteAction,
  SystemInfoAction,
  SystemHealthAction,
  DockerStartAction,
  DockerStopAction,
  DockerListAction,
  ShellExecuteResponse,
  SystemInfoResponse,
  SystemHealthResponse,
  DockerStartResponse,
  DockerStopResponse,
  DockerListResponse,
} from "nodelink-shared";

import { DockerActions } from "./docker-actions";

const execAsync = promisify(exec);

export class ActionExecutor {
  private runningTasks: Map<string, any> = new Map();
  private dockerActions: DockerActions;

  constructor(private nodeId: string) {
    this.dockerActions = new DockerActions();
  }

  async executeTask(execution: TaskExecution): Promise<TaskResult> {
    const { taskId, action, payload } = execution;

    console.log(`Executing task ${taskId}: ${action}`);

    try {
      let result: any;

      switch (action) {
        case "shell.execute":
          result = await this.executeShell(
            taskId,
            payload as ShellExecuteAction
          );
          break;
        case "system.info":
          result = await this.getSystemInfo(payload as SystemInfoAction);
          break;
        case "system.health":
          result = await this.getSystemHealth(payload as SystemHealthAction);
          break;
        case "docker.start":
          result = await this.dockerActions.startContainer(
            payload as DockerStartAction
          );
          break;
        case "docker.stop":
          result = await this.dockerActions.stopContainer(
            payload as DockerStopAction
          );
          break;
        case "docker.list":
          result = await this.dockerActions.listContainers(
            payload as DockerListAction
          );
          break;
        default:
          throw new Error(`Unknown action type: ${action}`);
      }

      return {
        taskId,
        success: true,
        result,
        duration: 0, // TODO: implement timing
      };
    } catch (error) {
      console.error(`Task ${taskId} failed:`, error);
      return {
        taskId,
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
        duration: 0,
      };
    }
  }

  private async executeShell(
    taskId: string,
    action: ShellExecuteAction
  ): Promise<ShellExecuteResponse> {
    const { command, cwd, timeout = 30000, env } = action;

    return new Promise((resolve, reject) => {
      const startTime = Date.now();
      const childProcess = spawn(command, {
        shell: true,
        cwd: cwd || process.cwd(),
        env: { ...process.env, ...env },
      });

      let stdout = "";
      let stderr = "";

      // Store reference for cancellation
      this.runningTasks.set(taskId, childProcess);

      childProcess.stdout.on("data", (data) => {
        const output = data.toString();
        stdout += output;

        // Emit real-time output
        this.emit("task.output", {
          taskId,
          output,
          stream: "stdout",
          timestamp: new Date(),
        });
      });

      childProcess.stderr.on("data", (data) => {
        const output = data.toString();
        stderr += output;

        // Emit real-time output
        this.emit("task.output", {
          taskId,
          output,
          stream: "stderr",
          timestamp: new Date(),
        });
      });

      childProcess.on("close", (code) => {
        this.runningTasks.delete(taskId);
        const duration = Date.now() - startTime;

        resolve({
          exitCode: code || 0,
          output: stdout,
          error: stderr || undefined,
          duration,
        });
      });

      childProcess.on("error", (error) => {
        this.runningTasks.delete(taskId);
        reject(error);
      });

      // Set timeout
      if (timeout > 0) {
        setTimeout(() => {
          if (this.runningTasks.has(taskId)) {
            childProcess.kill("SIGTERM");
            this.runningTasks.delete(taskId);
            reject(new Error(`Command timed out after ${timeout}ms`));
          }
        }, timeout);
      }
    });
  }

  private async getSystemInfo(
    action: SystemInfoAction
  ): Promise<SystemInfoResponse> {
    const {
      includeMetrics = false,
      includeProcesses = false,
      includeNetwork = false,
    } = action;

    const info: SystemInfoResponse = {
      platform: os.platform(),
      arch: os.arch(),
      version: os.release(),
      hostname: os.hostname(),
      uptime: os.uptime(),
      memory: {
        total: os.totalmem(),
        free: os.freemem(),
        used: os.totalmem() - os.freemem(),
        usage: ((os.totalmem() - os.freemem()) / os.totalmem()) * 100,
      },
      cpu: {
        model: os.cpus()[0]?.model || "Unknown",
        cores: os.cpus().length,
        usage: 0, // TODO: implement CPU usage calculation
      },
      disk: {
        total: 0,
        free: 0,
        used: 0,
        usage: 0,
      },
    };

    // Get disk usage (simplified)
    try {
      const { stdout } = await execAsync("df -h / | tail -1");
      const parts = stdout.trim().split(/\s+/);
      if (parts.length >= 4) {
        info.disk.total = this.parseSize(parts[1]);
        info.disk.used = this.parseSize(parts[2]);
        info.disk.free = this.parseSize(parts[3]);
        info.disk.usage = parseFloat(parts[4].replace("%", ""));
      }
    } catch (error) {
      console.warn("Failed to get disk usage:", error);
    }

    if (includeNetwork) {
      info.network = {
        interfaces: Object.entries(os.networkInterfaces()).flatMap(
          ([name, interfaces]) =>
            (interfaces || []).map((iface) => ({
              name,
              address: iface.address,
              mac: iface.mac,
              internal: iface.internal,
            }))
        ),
      };
    }

    if (includeProcesses) {
      try {
        const { stdout } = await execAsync("ps aux | head -20");
        const lines = stdout.trim().split("\n").slice(1);
        info.processes = lines.map((line) => {
          const parts = line.trim().split(/\s+/);
          return {
            pid: parseInt(parts[1]) || 0,
            name: parts[10] || "unknown",
            cpu: parseFloat(parts[2]) || 0,
            memory: parseFloat(parts[3]) || 0,
          };
        });
      } catch (error) {
        console.warn("Failed to get process list:", error);
      }
    }

    return info;
  }

  private async getSystemHealth(
    action: SystemHealthAction
  ): Promise<SystemHealthResponse> {
    const { checkDisk = true, checkMemory = true, checkCpu = true } = action;

    const health: SystemHealthResponse = {
      healthy: true,
      checks: {},
      timestamp: new Date(),
    };

    if (checkMemory) {
      const memoryUsage =
        ((os.totalmem() - os.freemem()) / os.totalmem()) * 100;
      health.checks.memory = {
        healthy: memoryUsage < 85,
        usage: memoryUsage,
        threshold: 85,
      };
    }

    if (checkDisk) {
      try {
        const { stdout } = await execAsync("df -h / | tail -1");
        const parts = stdout.trim().split(/\s+/);
        if (parts.length >= 4) {
          const usage = parseFloat(parts[4].replace("%", ""));
          health.checks.disk = {
            healthy: usage < 90,
            usage,
            threshold: 90,
          };
        }
      } catch (error) {
        health.checks.disk = {
          healthy: false,
          usage: 0,
          threshold: 90,
        };
      }
    }

    if (checkCpu) {
      // Simplified CPU check - in real implementation, would calculate actual usage
      health.checks.cpu = {
        healthy: true,
        usage: 0,
        threshold: 80,
      };
    }

    // Overall health
    health.healthy = Object.values(health.checks).every(
      (check) => check.healthy
    );

    return health;
  }

  private async listDocker(
    action: DockerListAction
  ): Promise<DockerListResponse> {
    const { all = false } = action;

    const cmd = all
      ? 'docker ps -a --format "table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.CreatedAt}}"'
      : 'docker ps --format "table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.CreatedAt}}"';

    try {
      const { stdout } = await execAsync(cmd);
      const lines = stdout.trim().split("\n").slice(1); // Skip header

      const containers = lines.map((line) => {
        const parts = line.split("\t");
        return {
          id: parts[0] || "",
          name: parts[1] || "",
          image: parts[2] || "",
          status: parts[3] || "",
          ports: this.parsePorts(parts[4] || ""),
          created: new Date(parts[5] || ""),
        };
      });

      return { containers };
    } catch (error) {
      throw new Error(`Failed to list containers: ${error}`);
    }
  }

  private parseSize(sizeStr: string): number {
    const units = { K: 1024, M: 1024 ** 2, G: 1024 ** 3, T: 1024 ** 4 };
    const match = sizeStr.match(/^(\d+(?:\.\d+)?)\s*([KMGT]?)$/);
    if (!match) return 0;

    const value = parseFloat(match[1]);
    const unit = match[2] as keyof typeof units;
    return value * (units[unit] || 1);
  }

  private parsePorts(
    portStr: string
  ): Array<{ host: number; container: number; protocol: "tcp" | "udp" }> {
    const ports: Array<{
      host: number;
      container: number;
      protocol: "tcp" | "udp";
    }> = [];

    if (!portStr) return ports;

    // Simple port parsing - in real implementation, would be more robust
    const matches = portStr.match(/(\d+):(\d+)/g);
    if (matches) {
      for (const match of matches) {
        const [host, container] = match.split(":").map(Number);
        ports.push({ host, container, protocol: "tcp" });
      }
    }

    return ports;
  }

  cancelTask(taskId: string): void {
    const process = this.runningTasks.get(taskId);
    if (process) {
      process.kill("SIGTERM");
      this.runningTasks.delete(taskId);
    }
  }

  getRunningTasks(): string[] {
    return Array.from(this.runningTasks.keys());
  }

  // Event emitter placeholder - in real implementation, would extend EventEmitter
  private emit(event: string, data: any): void {
    // This would emit events to the main node agent
    console.log(`Event: ${event}`, data);
  }
}
