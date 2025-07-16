import express, { Application } from "express";
import https from "https";
import fs from "fs";
import { Server, Socket } from "socket.io";
import {
  NodeRegistration,
  ServerToNodeEvents,
  NodeToServerEvents,
  ServerToFrontendEvents,
  FrontendToServerEvents,
  validateActionWithDetails,
  ActionType,
} from "nodelink-shared";

import { TaskManager } from "./task-manager";
import { NodeManager } from "./node-manager";

export class NodeLinkServer {
  private app: Application;
  private server: https.Server;
  private io: Server<
    NodeToServerEvents & FrontendToServerEvents,
    ServerToNodeEvents & ServerToFrontendEvents
  >;
  private taskManager: TaskManager;
  private nodeManager: NodeManager;
  private frontendConnections: Set<Socket> = new Set();

  constructor(port: number = 8443) {
    this.app = express();
    this.server = https.createServer(
      {
        cert: fs.readFileSync("./certs/localhost.pem"),
        key: fs.readFileSync("./certs/localhost-key.pem"),
      },
      this.app
    );

    this.io = new Server(this.server);
    this.taskManager = new TaskManager();
    this.nodeManager = new NodeManager();

    this.setupRoutes();
    this.setupSocketHandlers();
    this.setupEventHandlers();

    this.server.listen(port, () => {
      console.log(`NodeLink server running on https://localhost:${port}`);
    });
  }

  private setupRoutes(): void {
    // Enable JSON parsing
    this.app.use(express.json());

    // Serve static files
    this.app.use(express.static("../frontend"));

    // Health check endpoint
    this.app.get("/health", (req, res) => {
      res.json({
        status: "healthy",
        timestamp: new Date().toISOString(),
        nodes: this.nodeManager.getNodeList().length,
        runningTasks: this.taskManager.getRunningTasks().length,
      });
    });

    // API endpoints
    this.app.get("/api/nodes", (req, res) => {
      res.json({
        success: true,
        data: this.nodeManager.getNodeList(),
      });
    });

    this.app.get("/api/nodes/:nodeId", (req, res) => {
      const node = this.nodeManager.getNode(req.params.nodeId);
      if (!node) {
        return res.status(404).json({
          success: false,
          error: "Node not found",
        });
      }
      res.json({
        success: true,
        data: node,
      });
    });

    this.app.get("/api/tasks", (req, res) => {
      const { nodeId, status } = req.query;
      let tasks = this.taskManager.getAllTasks();

      if (nodeId) {
        tasks = tasks.filter((task) => task.nodeId === nodeId);
      }

      if (status) {
        tasks = tasks.filter((task) => task.status === status);
      }

      res.json({
        success: true,
        data: tasks,
      });
    });

    this.app.get("/api/tasks/:taskId", (req, res) => {
      const task = this.taskManager.getTask(req.params.taskId);
      if (!task) {
        return res.status(404).json({
          success: false,
          error: "Task not found",
        });
      }
      res.json({
        success: true,
        data: task,
      });
    });

    // Create task endpoint
    this.app.post("/api/tasks", (req, res) => {
      const { nodeId, type, payload, options } = req.body;

      if (!nodeId || !type || !payload) {
        return res.status(400).json({
          success: false,
          error: "Missing required fields: nodeId, type, payload",
        });
      }

      // Validate action
      const validation = validateActionWithDetails(type, payload);
      if (!validation.valid) {
        return res.status(400).json({
          success: false,
          error: "Invalid action",
          details: validation.error,
        });
      }

      // Check if node is online
      if (!this.nodeManager.isNodeOnline(nodeId)) {
        return res.status(400).json({
          success: false,
          error: "Node is not online",
        });
      }

      try {
        const task = this.taskManager.createTask(
          type as ActionType,
          nodeId,
          payload,
          options
        );

        res.status(201).json({
          success: true,
          data: task,
        });
      } catch (error) {
        res.status(500).json({
          success: false,
          error: error instanceof Error ? error.message : "Unknown error",
        });
      }
    });

    // Cancel task endpoint
    this.app.delete("/api/tasks/:taskId", (req, res) => {
      const task = this.taskManager.getTask(req.params.taskId);
      if (!task) {
        return res.status(404).json({
          success: false,
          error: "Task not found",
        });
      }

      this.taskManager.cancelTask(req.params.taskId);
      res.json({
        success: true,
        message: "Task cancelled successfully",
      });
    });

    // Get server statistics
    this.app.get("/api/stats", (req, res) => {
      res.json({
        success: true,
        data: this.getStats(),
      });
    });
  }

  private setupSocketHandlers(): void {
    this.io.on("connection", (socket: Socket) => {
      console.log("Client connected:", socket.id);

      // Handle node registration
      socket.on("node.register", (registration: NodeRegistration) => {
        // TODO: validate registration
        const success = this.nodeManager.registerNode(socket, registration);

        if (success) {
          console.log(`Node ${registration.id} registered`);
          this.taskManager.registerNode(registration.id, socket);
          this.frontendConnections.delete(socket);
        } else {
          console.log(
            `Invalid registration attempt for node ${registration.id}`
          );
          socket.disconnect();
        }
      });

      // Handle frontend task creation
      // TODO: convert to use REST pattern
      socket.on(
        "task.create",
        (data: { nodeId: string; type: string; payload: any }) => {
          const { nodeId, type, payload } = data;

          // Validate action
          const validation = validateActionWithDetails(type, payload);
          if (!validation.valid) {
            socket.emit("error", {
              message: "Invalid action",
              details: validation.error,
            });
            return;
          }

          // Check if node is online
          if (!this.nodeManager.isNodeOnline(nodeId)) {
            socket.emit("error", {
              message: "Node is not online",
              nodeId,
            });
            return;
          }

          // Create task
          try {
            const task = this.taskManager.createTask(
              type as ActionType,
              nodeId,
              payload
            );

            // Notify frontend of task creation
            socket.emit("task.created", { task });
          } catch (error) {
            socket.emit("error", {
              message: "Failed to create task",
              error: error instanceof Error ? error.message : "Unknown error",
            });
          }
        }
      );

      // Handle task cancellation
      // TODO: convert to use REST pattern
      socket.on("task.cancel", (data: { taskId: string }) => {
        this.taskManager.cancelTask(data.taskId);
      });

      // Handle node list request
      // TODO: convert to use REST pattern
      socket.on("node.list", () => {
        socket.emit("node.list", {
          nodes: this.nodeManager.getNodeList(),
        });
      });

      // Handle disconnect
      socket.on("disconnect", () => {
        console.log("Client disconnected:", socket.id);
        this.frontendConnections.delete(socket);
      });

      // Add to frontend connections (assume non-node connections are frontends)
      this.frontendConnections.add(socket);

      // Send initial node list
      socket.emit("node.list", {
        nodes: this.nodeManager.getNodeList(),
      });
    });
  }

  private setupEventHandlers(): void {
    // Task manager events
    this.taskManager.on("task.created", (task) => {
      this.broadcastToFrontends("task.created", { task });
    });

    this.taskManager.on("task.updated", (task) => {
      this.broadcastToFrontends("task.updated", { task });
    });

    this.taskManager.on("task.completed", (task) => {
      this.broadcastToFrontends("task.completed", { task });
    });

    this.taskManager.on("task.output", (output) => {
      this.broadcastToFrontends("task.output", output);
    });

    // Node manager events
    this.nodeManager.on("node.registered", (node) => {
      console.log(`Node ${node.id} registered:`, node.systemInfo);
    });

    this.nodeManager.on("node.disconnected", (nodeId) => {
      console.log(`Node ${nodeId} disconnected`);
    });

    this.nodeManager.on("node.list.updated", (nodes) => {
      this.broadcastToFrontends("node.list", { nodes });
    });

    this.nodeManager.on("node.timeout", (nodeId) => {
      console.log(`Node ${nodeId} timed out`);
    });
  }

  private broadcastToFrontends(event: string, data: any): void {
    this.frontendConnections.forEach((socket) => {
      socket.emit(event, data);
    });
  }

  // Get server statistics
  getStats() {
    return {
      nodes: this.nodeManager.getNodeList().length,
      onlineNodes: this.nodeManager.getOnlineNodes().length,
      totalTasks: this.taskManager.getAllTasks().length,
      runningTasks: this.taskManager.getRunningTasks().length,
      frontendConnections: this.frontendConnections.size,
    };
  }
}

// Export for use in index.ts
export default NodeLinkServer;
