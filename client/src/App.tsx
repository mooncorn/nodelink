import { Routes, Route } from "react-router-dom"
import DashboardLayout from "@/components/dashboard-layout"
import DashboardPage from "@/pages/dashboard"
import NodesPage from "@/pages/nodes"
import NodeDetailPage from "@/pages/node-detail"
import NodeSystemPage from "@/pages/node-system"
import NodeCommandPage from "@/pages/node-command"
import NodeTerminalPage from "@/pages/node-terminal"

function App() {
  return (
    <Routes>
      <Route path="/" element={<DashboardLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="nodes" element={<NodesPage />} />
        <Route path="nodes/:id" element={<NodeDetailPage />} />
        <Route path="nodes/:id/system" element={<NodeSystemPage />} />
        <Route path="nodes/:id/command" element={<NodeCommandPage />} />
        <Route path="nodes/:id/terminal" element={<NodeTerminalPage />} />
      </Route>
    </Routes>
  )
}

export default App
