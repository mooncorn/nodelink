import { Link } from "react-router-dom";
import { useApp } from "../context/AppContext";
import { RefreshCw, Server, Monitor, Clock } from "lucide-react";
import type { Node } from "../types";

function NodeCard({ node }: { node: Node }) {
  const isOnline = node.status === "online";

  return (
    <Link
      to={`/nodes/${node.id}`}
      className="block bg-white rounded-lg border border-gray-200 hover:border-blue-300 hover:shadow-md transition-all duration-200"
    >
      <div className="p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-3">
            <Server
              className={`h-6 w-6 ${
                isOnline ? "text-green-600" : "text-gray-400"
              }`}
            />
            <div>
              <h3 className="text-lg font-semibold text-gray-900">{node.id}</h3>
              <p className="text-sm text-gray-500">
                {node.systemInfo.hostname}
              </p>
            </div>
          </div>
          <div
            className={`px-3 py-1 rounded-full text-xs font-medium ${
              isOnline
                ? "bg-green-100 text-green-800"
                : "bg-gray-100 text-gray-800"
            }`}
          >
            {node.status}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div className="flex items-center space-x-2">
            <Monitor className="h-4 w-4 text-gray-400" />
            <span className="text-gray-600">
              {node.systemInfo.platform} {node.systemInfo.arch}
            </span>
          </div>
          <div className="flex items-center space-x-2">
            <Clock className="h-4 w-4 text-gray-400" />
            <span className="text-gray-600">
              {new Date(node.lastSeen).toLocaleTimeString()}
            </span>
          </div>
        </div>

        <div className="mt-4">
          <div className="text-xs text-gray-500 mb-1">Capabilities:</div>
          <div className="flex flex-wrap gap-1">
            {node.capabilities.map((capability) => (
              <span
                key={capability}
                className="px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded"
              >
                {capability}
              </span>
            ))}
          </div>
        </div>
      </div>
    </Link>
  );
}

export function NodesPage() {
  const { state, refreshData } = useApp();

  const handleRefresh = () => {
    refreshData();
  };

  if (state.loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Nodes</h1>
          <p className="text-gray-600 mt-2">
            Manage your remote nodes and monitor their status
          </p>
        </div>
        <button
          onClick={handleRefresh}
          className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          <span>Refresh</span>
        </button>
      </div>

      {state.error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <p className="text-red-800">{state.error}</p>
        </div>
      )}

      {state.nodes.length === 0 ? (
        <div className="text-center py-12">
          <Server className="h-12 w-12 text-gray-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">
            No nodes found
          </h3>
          <p className="text-gray-600">
            No nodes are currently registered with the server.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {state.nodes.map((node) => (
            <NodeCard key={node.id} node={node} />
          ))}
        </div>
      )}
    </div>
  );
}
