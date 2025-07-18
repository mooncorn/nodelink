import { createContext, useContext, useReducer, useEffect } from "react";
import type { ReactNode } from "react";
import { apiService } from "../services/api";
import { socketService } from "../services/socket";
import type { Node, Task, Stats } from "../types";

interface AppState {
  nodes: Node[];
  tasks: Task[];
  stats: Stats;
  loading: boolean;
  connected: boolean;
  error: string | null;
}

type AppAction =
  | { type: "SET_LOADING"; payload: boolean }
  | { type: "SET_CONNECTED"; payload: boolean }
  | { type: "SET_ERROR"; payload: string | null }
  | { type: "SET_NODES"; payload: Node[] }
  | { type: "SET_TASKS"; payload: Task[] }
  | { type: "SET_STATS"; payload: Stats }
  | { type: "UPDATE_TASK"; payload: Task }
  | { type: "ADD_TASK"; payload: Task }
  | { type: "ADD_TASK_OUTPUT"; payload: { taskId: string; output: string } };

const initialState: AppState = {
  nodes: [],
  tasks: [],
  stats: {
    nodes: 0,
    onlineNodes: 0,
    totalTasks: 0,
    runningTasks: 0,
    frontendConnections: 0,
  },
  loading: true,
  connected: false,
  error: null,
};

function appReducer(state: AppState, action: AppAction): AppState {
  switch (action.type) {
    case "SET_LOADING":
      return { ...state, loading: action.payload };
    case "SET_CONNECTED":
      return { ...state, connected: action.payload };
    case "SET_ERROR":
      return { ...state, error: action.payload };
    case "SET_NODES":
      return { ...state, nodes: action.payload };
    case "SET_TASKS":
      return { ...state, tasks: action.payload };
    case "SET_STATS":
      return { ...state, stats: action.payload };
    case "UPDATE_TASK":
      return {
        ...state,
        tasks: state.tasks.map((task) =>
          task.id === action.payload.id ? action.payload : task
        ),
      };
    case "ADD_TASK":
      // Check if task already exists to prevent duplicates
      const taskExists = state.tasks.some(
        (task) => task.id === action.payload.id
      );
      if (taskExists) {
        // If task exists, update it instead of adding duplicate
        return {
          ...state,
          tasks: state.tasks.map((task) =>
            task.id === action.payload.id ? action.payload : task
          ),
        };
      }
      return {
        ...state,
        tasks: [action.payload, ...state.tasks],
      };
    case "ADD_TASK_OUTPUT":
      return {
        ...state,
        tasks: state.tasks.map((task) =>
          task.id === action.payload.taskId
            ? {
                ...task,
                result: task.result
                  ? task.result + "\n" + action.payload.output
                  : action.payload.output,
              }
            : task
        ),
      };
    default:
      return state;
  }
}

interface AppContextType {
  state: AppState;
  refreshData: () => Promise<void>;
  refreshNodes: () => Promise<void>;
  refreshTasks: () => Promise<void>;
  refreshStats: () => Promise<void>;
}

const AppContext = createContext<AppContextType | undefined>(undefined);

export function AppProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(appReducer, initialState);

  const refreshNodes = async () => {
    try {
      const nodes = await apiService.getNodes();
      dispatch({ type: "SET_NODES", payload: nodes });
    } catch (error) {
      console.error("Failed to fetch nodes:", error);
      dispatch({ type: "SET_ERROR", payload: "Failed to fetch nodes" });
    }
  };

  const refreshTasks = async () => {
    try {
      const tasks = await apiService.getTasks();
      dispatch({ type: "SET_TASKS", payload: tasks });
    } catch (error) {
      console.error("Failed to fetch tasks:", error);
      dispatch({ type: "SET_ERROR", payload: "Failed to fetch tasks" });
    }
  };

  const refreshStats = async () => {
    try {
      const stats = await apiService.getStats();
      dispatch({ type: "SET_STATS", payload: stats });
    } catch (error) {
      console.error("Failed to fetch stats:", error);
    }
  };

  const refreshData = async () => {
    dispatch({ type: "SET_LOADING", payload: true });
    dispatch({ type: "SET_ERROR", payload: null });

    try {
      await Promise.all([refreshNodes(), refreshTasks(), refreshStats()]);
    } finally {
      dispatch({ type: "SET_LOADING", payload: false });
    }
  };

  useEffect(() => {
    // Initialize socket connection
    socketService.connect();

    // Set up socket event listeners
    socketService.on("connect", () => {
      dispatch({ type: "SET_CONNECTED", payload: true });
    });

    socketService.on("disconnect", () => {
      dispatch({ type: "SET_CONNECTED", payload: false });
    });

    socketService.on("node.list", (data) => {
      dispatch({ type: "SET_NODES", payload: data.nodes });
    });

    socketService.on("task.created", (data) => {
      console.log("WebSocket: task.created", data.task.id);
      dispatch({ type: "ADD_TASK", payload: data.task });
    });

    socketService.on("task.updated", (data) => {
      console.log("WebSocket: task.updated", data.task.id, data.task.status);
      dispatch({ type: "UPDATE_TASK", payload: data.task });
    });

    socketService.on("task.completed", (data) => {
      console.log("WebSocket: task.completed", data.task.id);
      dispatch({ type: "UPDATE_TASK", payload: data.task });
    });

    socketService.on("task.output", (data) => {
      dispatch({ type: "ADD_TASK_OUTPUT", payload: data });
    });

    socketService.on("error", (error) => {
      dispatch({ type: "SET_ERROR", payload: error.message });
    });

    // Initial data fetch
    refreshData();

    // Cleanup on unmount
    return () => {
      socketService.disconnect();
    };
  }, []);

  const value: AppContextType = {
    state,
    refreshData,
    refreshNodes,
    refreshTasks,
    refreshStats,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp() {
  const context = useContext(AppContext);
  if (context === undefined) {
    throw new Error("useApp must be used within an AppProvider");
  }
  return context;
}
