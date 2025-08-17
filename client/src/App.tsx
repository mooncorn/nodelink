import { Routes, Route } from "react-router-dom"
import DashboardLayout from "@/components/dashboard-layout"
import DashboardPage from "@/pages/dashboard"
import NodesPage from "@/pages/nodes"
import NodeDetailPage from "@/pages/node-detail"
import NodeCommandPage from "@/pages/node-command"
import NodeTerminalPage from "@/pages/node-terminal"

function App() {
  return (
    <Routes>
      <Route path="/" element={<DashboardLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="nodes" element={<NodesPage />} />
        <Route path="nodes/:nodeId" element={<NodeDetailPage />} />
        <Route path="nodes/:nodeId/command" element={<NodeCommandPage />} />
        <Route path="nodes/:nodeId/terminal" element={<NodeTerminalPage />} />
      </Route>
    </Routes>
  )
}

export default App
