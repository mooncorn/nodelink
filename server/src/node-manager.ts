import { EventEmitter } from "events";
import {
  NodeRegistration,
  NodeHeartbeat,
  NodeInfo,
  NodeConfig,
} from "nodelink-shared";

export class NodeManager extends EventEmitter {
  private nodes: Map<string, NodeInfo> = new Map();
  private nodeConnections: Map<string, any> = new Map(); // nodeId -> socket
  private validTokens: Map<string, string> = new Map(); // nodeId -> token
  private heartbeatTimeouts: Map<string, NodeJS.Timeout> = new Map();

  constructor() {
    super();

    // Initialize valid tokens (in production, this should come from database)
    this.validTokens.set("node1", "token-for-node1");
    this.validTokens.set("node2", "token-for-node2");

    this.startHeartbeatMonitoring();
  }

  // Register a node
  registerNode(socket: any, registration: NodeRegistration): boolean {
    const { id, token, capabilities, systemInfo } = registration;

    // Validate token
    if (this.validTokens.get(id) !== token) {
      return false;
    }

    // Create node info
    const nodeInfo: NodeInfo = {
      id,
      status: "online",
      capabilities,
      systemInfo,
      lastSeen: new Date(),
      runningTasks: 0,
    };

    // Store node info and connection
    this.nodes.set(id, nodeInfo);
    this.nodeConnections.set(id, socket);

    // Setup socket event handlers
    socket.on("node.heartbeat", (heartbeat: NodeHeartbeat) => {
      this.handleHeartbeat(heartbeat);
    });

    socket.on("node.pong", (data: { timestamp: Date }) => {
      this.handlePong(id, data.timestamp);
    });

    socket.on("disconnect", () => {
      this.handleDisconnect(id);
    });

    // Send node configuration
    const config: NodeConfig = {
      heartbeatInterval: 30000, // 30 seconds
      maxConcurrentTasks: 5,
      taskTimeout: 300000, // 5 minutes
      enableMetrics: true,
    };

    socket.emit("node.config", { config });

    // Start heartbeat timeout
    this.resetHeartbeatTimeout(id);

    this.emit("node.registered", nodeInfo);
    this.emit("node.list.updated", this.getNodeList());

    return true;
  }

  // Handle heartbeat from node
  private handleHeartbeat(heartbeat: NodeHeartbeat): void {
    const node = this.nodes.get(heartbeat.nodeId);
    if (!node) return;

    // Update node info
    node.lastSeen = heartbeat.timestamp;
    node.status = heartbeat.status;
    node.runningTasks = heartbeat.runningTasks.length;

    // Store metrics if available
    if (heartbeat.systemMetrics) {
      (node as any).systemMetrics = heartbeat.systemMetrics;
    }

    this.nodes.set(heartbeat.nodeId, node);
    this.resetHeartbeatTimeout(heartbeat.nodeId);

    this.emit("node.heartbeat", heartbeat);
    this.emit("node.list.updated", this.getNodeList());
  }

  // Handle pong response
  private handlePong(nodeId: string, timestamp: Date): void {
    const node = this.nodes.get(nodeId);
    if (!node) return;

    node.lastSeen = timestamp;
    this.nodes.set(nodeId, node);
    this.resetHeartbeatTimeout(nodeId);
  }

  // Handle node disconnect
  private handleDisconnect(nodeId: string): void {
    const node = this.nodes.get(nodeId);
    if (!node) return;

    // Update status to offline
    node.status = "offline";
    node.lastSeen = new Date();
    this.nodes.set(nodeId, node);

    // Remove connection
    this.nodeConnections.delete(nodeId);

    // Clear heartbeat timeout
    const timeoutId = this.heartbeatTimeouts.get(nodeId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      this.heartbeatTimeouts.delete(nodeId);
    }

    this.emit("node.disconnected", nodeId);
    this.emit("node.list.updated", this.getNodeList());
  }

  // Reset heartbeat timeout for a node
  private resetHeartbeatTimeout(nodeId: string): void {
    // Clear existing timeout
    const existingTimeout = this.heartbeatTimeouts.get(nodeId);
    if (existingTimeout) {
      clearTimeout(existingTimeout);
    }

    // Set new timeout (2x heartbeat interval)
    const timeout = setTimeout(() => {
      this.handleHeartbeatTimeout(nodeId);
    }, 60000); // 60 seconds

    this.heartbeatTimeouts.set(nodeId, timeout);
  }

  // Handle heartbeat timeout
  private handleHeartbeatTimeout(nodeId: string): void {
    const node = this.nodes.get(nodeId);
    if (!node) return;

    // Mark as offline
    node.status = "offline";
    node.lastSeen = new Date();
    this.nodes.set(nodeId, node);

    // Remove connection if it exists
    this.nodeConnections.delete(nodeId);

    this.emit("node.timeout", nodeId);
    this.emit("node.list.updated", this.getNodeList());
  }

  // Start heartbeat monitoring
  private startHeartbeatMonitoring(): void {
    setInterval(() => {
      for (const [nodeId, socket] of this.nodeConnections) {
        socket.emit("node.ping", { timestamp: new Date() });
      }
    }, 30000); // Ping every 30 seconds
  }

  // Get node by ID
  getNode(nodeId: string): NodeInfo | undefined {
    return this.nodes.get(nodeId);
  }

  // Get node connection
  getNodeConnection(nodeId: string): any {
    return this.nodeConnections.get(nodeId);
  }

  // Get node list
  getNodeList(): NodeInfo[] {
    return Array.from(this.nodes.values());
  }

  // Get online nodes
  getOnlineNodes(): NodeInfo[] {
    return Array.from(this.nodes.values()).filter(
      (node) => node.status === "online"
    );
  }

  // Check if node is online
  isNodeOnline(nodeId: string): boolean {
    const node = this.nodes.get(nodeId);
    return node?.status === "online";
  }

  // Add a new valid token (for dynamic node registration)
  addValidToken(nodeId: string, token: string): void {
    this.validTokens.set(nodeId, token);
  }

  // Remove a valid token
  removeValidToken(nodeId: string): void {
    this.validTokens.delete(nodeId);
  }
}
