import { io, Socket } from "socket.io-client";
import type { Node, Task } from "../types";

const SOCKET_URL = "https://localhost:8443";

export interface SocketEvents {
  connect: () => void;
  disconnect: () => void;
  "node.list": (data: { nodes: Node[] }) => void;
  "task.created": (data: { task: Task }) => void;
  "task.updated": (data: { task: Task }) => void;
  "task.completed": (data: { task: Task }) => void;
  "task.output": (data: { taskId: string; output: string }) => void;
  error: (error: { message: string }) => void;
}

class SocketService {
  private socket: Socket | null = null;
  private eventListeners: Map<string, Set<Function>> = new Map();

  connect(): void {
    if (this.socket?.connected) return;

    this.socket = io(SOCKET_URL, {
      withCredentials: true,
    });

    // Set up event forwarding
    this.socket.on("connect", () => {
      console.log("Connected to server");
      this.emit("connect");
    });

    this.socket.on("disconnect", () => {
      console.log("Disconnected from server");
      this.emit("disconnect");
    });

    this.socket.on("node.list", (data) => {
      this.emit("node.list", data);
    });

    this.socket.on("task.created", (data) => {
      this.emit("task.created", data);
    });

    this.socket.on("task.updated", (data) => {
      this.emit("task.updated", data);
    });

    this.socket.on("task.completed", (data) => {
      this.emit("task.completed", data);
    });

    this.socket.on("task.output", (data) => {
      this.emit("task.output", data);
    });

    this.socket.on("error", (error) => {
      console.error("Socket error:", error);
      this.emit("error", error);
    });
  }

  disconnect(): void {
    this.socket?.disconnect();
    this.socket = null;
  }

  isConnected(): boolean {
    return this.socket?.connected || false;
  }

  on<K extends keyof SocketEvents>(event: K, callback: SocketEvents[K]): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, new Set());
    }
    this.eventListeners.get(event)!.add(callback);
  }

  off<K extends keyof SocketEvents>(event: K, callback: SocketEvents[K]): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.delete(callback);
    }
  }

  private emit(event: string, ...args: any[]): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach((callback) => {
        try {
          callback(...args);
        } catch (error) {
          console.error(`Error in socket event listener for ${event}:`, error);
        }
      });
    }
  }
}

export const socketService = new SocketService();
