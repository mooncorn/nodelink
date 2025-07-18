import axios from "axios";
import type { Node, Task, Stats, ApiResponse, ActionType } from "../types";

const API_BASE_URL = "https://localhost:8443";

// Configure axios for API calls
const api = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
});

export const apiService = {
  // Node endpoints
  async getNodes(): Promise<Node[]> {
    const response = await api.get<ApiResponse<Node[]>>("/api/nodes");
    return response.data.data || [];
  },

  async getNode(nodeId: string): Promise<Node | null> {
    try {
      const response = await api.get<ApiResponse<Node>>(`/api/nodes/${nodeId}`);
      return response.data.data || null;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 404) {
        return null;
      }
      throw error;
    }
  },

  // Task endpoints
  async getTasks(nodeId?: string, status?: string): Promise<Task[]> {
    const params = new URLSearchParams();
    if (nodeId) params.append("nodeId", nodeId);
    if (status) params.append("status", status);

    const url = `/api/tasks${params.toString() ? `?${params.toString()}` : ""}`;
    const response = await api.get<ApiResponse<Task[]>>(url);
    return response.data.data || [];
  },

  async getTask(taskId: string): Promise<Task | null> {
    try {
      const response = await api.get<ApiResponse<Task>>(`/api/tasks/${taskId}`);
      return response.data.data || null;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 404) {
        return null;
      }
      throw error;
    }
  },

  async createTask(
    nodeId: string,
    type: ActionType,
    payload: any,
    options?: any
  ): Promise<Task> {
    const response = await api.post<ApiResponse<Task>>("/api/tasks", {
      nodeId,
      type,
      payload,
      options,
    });

    if (!response.data.success) {
      throw new Error(response.data.error || "Failed to create task");
    }

    return response.data.data!;
  },

  async cancelTask(taskId: string): Promise<void> {
    await api.delete(`/api/tasks/${taskId}`);
  },

  // Stats endpoint
  async getStats(): Promise<Stats> {
    const response = await api.get<ApiResponse<Stats>>("/api/stats");
    return (
      response.data.data || {
        nodes: 0,
        onlineNodes: 0,
        totalTasks: 0,
        runningTasks: 0,
        frontendConnections: 0,
      }
    );
  },
};
