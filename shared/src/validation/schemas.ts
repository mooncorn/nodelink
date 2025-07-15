import { z } from "zod";

// Action validation schemas
export const DockerStartActionSchema = z.object({
  image: z.string().min(1),
  containerName: z.string().optional(),
  ports: z
    .array(
      z.object({
        host: z.number().int().min(1).max(65535),
        container: z.number().int().min(1).max(65535),
        protocol: z.enum(["tcp", "udp"]).optional(),
      })
    )
    .optional(),
  environment: z.record(z.string()).optional(),
  volumes: z
    .array(
      z.object({
        host: z.string().min(1),
        container: z.string().min(1),
        mode: z.enum(["ro", "rw"]).optional(),
      })
    )
    .optional(),
});

export const DockerStopActionSchema = z.object({
  containerId: z.string().min(1),
  force: z.boolean().optional(),
});

export const DockerListActionSchema = z.object({
  all: z.boolean().optional(),
  filters: z.record(z.string()).optional(),
});

export const ShellExecuteActionSchema = z.object({
  command: z.string().min(1),
  cwd: z.string().optional(),
  timeout: z.number().int().min(1).optional(),
  env: z.record(z.string()).optional(),
});

export const SystemInfoActionSchema = z.object({
  includeMetrics: z.boolean().optional(),
  includeProcesses: z.boolean().optional(),
  includeNetwork: z.boolean().optional(),
});

export const SystemHealthActionSchema = z.object({
  checkDisk: z.boolean().optional(),
  checkMemory: z.boolean().optional(),
  checkCpu: z.boolean().optional(),
});

// Task validation schemas
export const TaskSchema = z.object({
  id: z.string().uuid(),
  type: z.string(),
  nodeId: z.string().min(1),
  payload: z.any(),
  status: z.enum(["pending", "running", "completed", "failed", "cancelled"]),
  createdAt: z.date(),
  startedAt: z.date().optional(),
  completedAt: z.date().optional(),
  timeout: z.number().int().min(1).optional(),
  retryCount: z.number().int().min(0).optional(),
  maxRetries: z.number().int().min(0).optional(),
  error: z.string().optional(),
  result: z.any().optional(),
});

export const TaskUpdateSchema = z.object({
  taskId: z.string().uuid(),
  status: z.enum(["pending", "running", "completed", "failed", "cancelled"]),
  progress: z.number().min(0).max(100).optional(),
  output: z.string().optional(),
  error: z.string().optional(),
  result: z.any().optional(),
  timestamp: z.date(),
});

export const TaskResultSchema = z.object({
  taskId: z.string().uuid(),
  success: z.boolean(),
  result: z.any().optional(),
  error: z.string().optional(),
  output: z.string().optional(),
  exitCode: z.number().int().optional(),
  duration: z.number().min(0).optional(),
});

// Event validation schemas
export const NodeRegistrationSchema = z.object({
  id: z.string().min(1),
  token: z.string().min(1),
  capabilities: z.array(z.string()),
  systemInfo: z.object({
    platform: z.string(),
    arch: z.string(),
    version: z.string(),
    hostname: z.string(),
  }),
});

export const NodeHeartbeatSchema = z.object({
  nodeId: z.string().min(1),
  timestamp: z.date(),
  status: z.enum(["online", "busy", "offline"]),
  runningTasks: z.array(z.string()),
  systemMetrics: z
    .object({
      cpuUsage: z.number().min(0).max(100),
      memoryUsage: z.number().min(0).max(100),
      diskUsage: z.number().min(0).max(100),
    })
    .optional(),
});

export const TaskOutputSchema = z.object({
  taskId: z.string().uuid(),
  output: z.string(),
  stream: z.enum(["stdout", "stderr"]),
  timestamp: z.date(),
});

// Action registry schema mapping
export const ActionSchemaRegistry = {
  "docker.start": DockerStartActionSchema,
  "docker.stop": DockerStopActionSchema,
  "docker.list": DockerListActionSchema,
  "shell.execute": ShellExecuteActionSchema,
  "system.info": SystemInfoActionSchema,
  "system.health": SystemHealthActionSchema,
} as const;

// Validation helper function
export function validateAction(type: string, payload: any): boolean {
  const schema =
    ActionSchemaRegistry[type as keyof typeof ActionSchemaRegistry];
  if (!schema) {
    return false;
  }

  try {
    schema.parse(payload);
    return true;
  } catch {
    return false;
  }
}

// Validation helper with error details
export function validateActionWithDetails(
  type: string,
  payload: any
): {
  valid: boolean;
  error?: string;
  details?: any;
} {
  const schema =
    ActionSchemaRegistry[type as keyof typeof ActionSchemaRegistry];
  if (!schema) {
    return {
      valid: false,
      error: `Unknown action type: ${type}`,
    };
  }

  try {
    schema.parse(payload);
    return { valid: true };
  } catch (error) {
    return {
      valid: false,
      error: "Validation failed",
      details: error,
    };
  }
}
