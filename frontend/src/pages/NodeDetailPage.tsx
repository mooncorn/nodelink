import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { useApp } from "../context/AppContext";
import { apiService } from "../services/api";
import {
  ArrowLeft,
  Server,
  Terminal,
  Container,
  Activity,
  RefreshCw,
  Clock,
} from "lucide-react";
import type {
  Node,
  Task,
  ShellPayload,
  DockerRunPayload,
  DockerControlPayload,
  DockerListAction,
} from "../types";

interface TaskCardProps {
  task: Task;
}

function TaskCard({ task }: TaskCardProps) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-100 text-green-800";
      case "running":
        return "bg-blue-100 text-blue-800";
      case "failed":
        return "bg-red-100 text-red-800";
      case "cancelled":
        return "bg-gray-100 text-gray-800";
      case "pending":
        return "bg-yellow-100 text-yellow-800";
      default:
        return "bg-gray-100 text-gray-600";
    }
  };

  // Debug log for task status
  console.log("TaskCard rendering:", {
    id: task.id,
    type: task.type,
    status: task.status,
    hasResult: !!task.result,
    hasError: !!task.error,
    result: task.result,
  });

  return (
    <div className="bg-white border rounded-lg p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="font-medium text-gray-900">{task.type}</span>
        <span
          className={`px-2 py-1 rounded text-xs font-medium ${getStatusColor(
            task.status
          )}`}
        >
          {task.status}
        </span>
      </div>
      <div className="text-sm text-gray-600 mb-2">
        <Clock className="h-3 w-3 inline mr-1" />
        {new Date(task.createdAt).toLocaleString()}
        {task.completedAt && (
          <span className="ml-2 text-green-600">
            • Completed: {new Date(task.completedAt).toLocaleString()}
          </span>
        )}
      </div>

      {/* Debug information */}
      <div className="text-xs text-gray-500 mb-2">
        ID: {task.id} | Status: {task.status}
        {task.startedAt &&
          ` | Started: ${new Date(task.startedAt).toLocaleTimeString()}`}
      </div>
      {task.error && (
        <div className="text-sm text-red-600 bg-red-50 p-2 rounded">
          {task.error}
        </div>
      )}
      {task.result && (
        <pre className="text-xs bg-gray-50 p-2 rounded overflow-x-auto">
          {typeof task.result === "string"
            ? task.result
            : JSON.stringify(task.result, null, 2)}
        </pre>
      )}
    </div>
  );
}

function ShellExecuteForm({ nodeId }: { nodeId: string }) {
  const [command, setCommand] = useState("");
  const [cwd, setCwd] = useState("");
  const [timeout, setTimeout] = useState(30);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!command.trim()) return;

    setLoading(true);
    try {
      const payload: ShellPayload = { command, timeout };
      if (cwd.trim()) payload.cwd = cwd;

      await apiService.createTask(nodeId, "shell.execute", payload);
      setCommand("");
      setCwd("");
    } catch (error) {
      console.error("Failed to execute command:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Command
        </label>
        <input
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="ls -la"
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          required
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Working Directory
          </label>
          <input
            type="text"
            value={cwd}
            onChange={(e) => setCwd(e.target.value)}
            placeholder="/home/user"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Timeout (seconds)
          </label>
          <input
            type="number"
            value={timeout}
            onChange={(e) => setTimeout(Number(e.target.value))}
            min="1"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>
      <button
        type="submit"
        disabled={loading || !command.trim()}
        className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? (
          <RefreshCw className="h-4 w-4 animate-spin" />
        ) : (
          <Terminal className="h-4 w-4" />
        )}
        <span>Execute</span>
      </button>
    </form>
  );
}

function DockerForm({ nodeId }: { nodeId: string }) {
  const [action, setAction] = useState<
    "run" | "start" | "stop" | "delete" | "list"
  >("list");
  const [image, setImage] = useState("");
  const [containerName, setContainerName] = useState("");
  const [containerId, setContainerId] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      let payload: any = {};

      if (action === "run") {
        if (!image.trim()) return;
        payload = { image } as DockerRunPayload;
        if (containerName.trim()) payload.containerName = containerName;
      } else if (action === "list") {
        payload = { all: true } as DockerListAction; // Show all containers (including stopped ones)
      } else if (["start", "stop", "delete"].includes(action)) {
        if (!containerId.trim()) return;
        payload = { containerId } as DockerControlPayload;
      }

      await apiService.createTask(nodeId, `docker.${action}` as any, payload);

      // Reset form
      setImage("");
      setContainerName("");
      setContainerId("");
    } catch (error) {
      console.error("Failed to execute Docker action:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Action
        </label>
        <select
          value={action}
          onChange={(e) => setAction(e.target.value as any)}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="list">List Containers</option>
          <option value="run">Run Container</option>
          <option value="start">Start Container</option>
          <option value="stop">Stop Container</option>
          <option value="delete">Delete Container</option>
        </select>
      </div>

      {action === "run" && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Image
            </label>
            <input
              type="text"
              value={image}
              onChange={(e) => setImage(e.target.value)}
              placeholder="nginx:latest"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Container Name (optional)
            </label>
            <input
              type="text"
              value={containerName}
              onChange={(e) => setContainerName(e.target.value)}
              placeholder="my-container"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </>
      )}

      {["start", "stop", "delete"].includes(action) && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Container ID
          </label>
          <input
            type="text"
            value={containerId}
            onChange={(e) => setContainerId(e.target.value)}
            placeholder="container_id_or_name"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            required
          />
        </div>
      )}

      <button
        type="submit"
        disabled={loading}
        className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? (
          <RefreshCw className="h-4 w-4 animate-spin" />
        ) : (
          <Container className="h-4 w-4" />
        )}
        <span>Execute</span>
      </button>
    </form>
  );
}

export function NodeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { state } = useApp();
  const [node, setNode] = useState<Node | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"shell" | "docker" | "system">(
    "shell"
  );

  useEffect(() => {
    if (!id) return;

    const loadData = async () => {
      setLoading(true);
      try {
        const [nodeData, tasksData] = await Promise.all([
          apiService.getNode(id),
          apiService.getTasks(id),
        ]);
        setNode(nodeData);
        setTasks(tasksData);
      } catch (error) {
        console.error("Failed to load node data:", error);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [id]);

  // Update tasks from global state
  useEffect(() => {
    if (id) {
      const nodeTasks = state.tasks.filter((task) => task.nodeId === id);

      // Deduplicate tasks by ID to prevent React key conflicts
      const uniqueTasks = nodeTasks.reduce((acc, task) => {
        const existingIndex = acc.findIndex((t) => t.id === task.id);
        if (existingIndex >= 0) {
          // Replace with the more recent version (keep the one with latest update)
          acc[existingIndex] = task;
        } else {
          acc.push(task);
        }
        return acc;
      }, [] as Task[]);

      // Sort tasks by creation date, newest first
      const sortedTasks = uniqueTasks.sort(
        (a, b) =>
          new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      );
      setTasks(sortedTasks);
      console.log(
        `Updated tasks for node ${id}:`,
        sortedTasks.length,
        "unique tasks"
      );
    }
  }, [state.tasks, id]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (!node) {
    return (
      <div className="text-center py-12">
        <Server className="h-12 w-12 text-gray-400 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">
          Node not found
        </h3>
        <Link to="/nodes" className="text-blue-600 hover:text-blue-800">
          ← Back to nodes
        </Link>
      </div>
    );
  }

  const isOnline = node.status === "online";

  return (
    <div>
      {/* Header */}
      <div className="mb-8">
        <Link
          to="/nodes"
          className="inline-flex items-center text-blue-600 hover:text-blue-800 mb-4"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to nodes
        </Link>

        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Server
              className={`h-8 w-8 ${
                isOnline ? "text-green-600" : "text-gray-400"
              }`}
            />
            <div>
              <h1 className="text-3xl font-bold text-gray-900">{node.id}</h1>
              <p className="text-gray-600">{node.systemInfo.hostname}</p>
            </div>
          </div>
          <div
            className={`px-4 py-2 rounded-full text-sm font-medium ${
              isOnline
                ? "bg-green-100 text-green-800"
                : "bg-gray-100 text-gray-800"
            }`}
          >
            {node.status}
          </div>
        </div>
      </div>

      {/* Node Info */}
      <div className="bg-white rounded-lg border p-6 mb-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          System Information
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div className="text-gray-500">Platform</div>
            <div className="font-medium">{node.systemInfo.platform}</div>
          </div>
          <div>
            <div className="text-gray-500">Architecture</div>
            <div className="font-medium">{node.systemInfo.arch}</div>
          </div>
          <div>
            <div className="text-gray-500">Version</div>
            <div className="font-medium">{node.systemInfo.version}</div>
          </div>
          <div>
            <div className="text-gray-500">Last Seen</div>
            <div className="font-medium">
              {new Date(node.lastSeen).toLocaleString()}
            </div>
          </div>
        </div>
        <div className="mt-4">
          <div className="text-gray-500 text-sm mb-2">Capabilities</div>
          <div className="flex flex-wrap gap-2">
            {node.capabilities.map((capability) => (
              <span
                key={capability}
                className="px-3 py-1 bg-blue-100 text-blue-700 text-sm rounded-full"
              >
                {capability}
              </span>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Controls */}
        <div>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Actions</h2>

          {/* Tabs */}
          <div className="flex space-x-1 mb-4">
            {[
              { id: "shell", label: "Shell", icon: Terminal },
              { id: "docker", label: "Docker", icon: Container },
              { id: "system", label: "System", icon: Activity },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`flex items-center space-x-2 px-4 py-2 rounded-md text-sm font-medium ${
                  activeTab === tab.id
                    ? "bg-blue-100 text-blue-700"
                    : "text-gray-600 hover:text-gray-900"
                }`}
              >
                <tab.icon className="h-4 w-4" />
                <span>{tab.label}</span>
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="bg-white border rounded-lg p-6">
            {!isOnline && (
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
                <p className="text-yellow-800">
                  Node is offline. Actions may not work.
                </p>
              </div>
            )}

            {activeTab === "shell" && <ShellExecuteForm nodeId={node.id} />}
            {activeTab === "docker" && <DockerForm nodeId={node.id} />}
            {activeTab === "system" && (
              <div className="space-y-4">
                <button
                  onClick={() =>
                    apiService.createTask(node.id, "system.info", {})
                  }
                  className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  <Activity className="h-4 w-4" />
                  <span>Get System Info</span>
                </button>
                <button
                  onClick={() =>
                    apiService.createTask(node.id, "system.health", {})
                  }
                  className="flex items-center space-x-2 px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700"
                >
                  <Activity className="h-4 w-4" />
                  <span>Health Check</span>
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Tasks */}
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">
              Recent Tasks
            </h2>
            <button
              onClick={async () => {
                if (id) {
                  const refreshedTasks = await apiService.getTasks(id);
                  setTasks(refreshedTasks);
                  console.log("Manually refreshed tasks:", refreshedTasks);
                }
              }}
              className="flex items-center space-x-1 px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-md"
            >
              <RefreshCw className="h-3 w-3" />
              <span>Refresh</span>
            </button>
          </div>
          <div className="space-y-4 max-h-96 overflow-y-auto">
            {tasks.length === 0 ? (
              <div className="text-center py-8 text-gray-500">
                No tasks found
              </div>
            ) : (
              tasks
                .slice(0, 10)
                .map((task) => <TaskCard key={task.id} task={task} />)
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
