import { Link } from "react-router-dom";
import { useApp } from "../context/AppContext";
import { Activity, Wifi, WifiOff } from "lucide-react";

export function Navbar() {
  const { state } = useApp();

  return (
    <nav className="bg-white shadow-sm border-b">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/" className="flex items-center space-x-2">
            <Activity className="h-8 w-8 text-blue-600" />
            <span className="text-xl font-bold text-gray-900">NodeLink</span>
          </Link>

          {/* Navigation Links */}
          <div className="flex items-center space-x-6">
            <Link
              to="/nodes"
              className="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
            >
              Nodes
            </Link>

            {/* Connection Status */}
            <div className="flex items-center space-x-2">
              {state.connected ? (
                <Wifi className="h-4 w-4 text-green-600" />
              ) : (
                <WifiOff className="h-4 w-4 text-red-600" />
              )}
              <span
                className={`text-sm ${
                  state.connected ? "text-green-600" : "text-red-600"
                }`}
              >
                {state.connected ? "Connected" : "Disconnected"}
              </span>
            </div>

            {/* Stats */}
            <div className="flex items-center space-x-4 text-sm text-gray-600">
              <span>
                Nodes:{" "}
                <span className="font-medium">
                  {state.stats.onlineNodes}/{state.stats.nodes}
                </span>
              </span>
              <span>
                Tasks:{" "}
                <span className="font-medium">{state.stats.runningTasks}</span>
              </span>
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
}
