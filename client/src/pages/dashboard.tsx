import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Activity, Server, Wifi, AlertTriangle } from "lucide-react"
import { apiService, type Node } from "@/lib/api"
import { Link } from "react-router-dom"

export default function DashboardPage() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchNodes = async () => {
      try {
        const fetchedNodes = await apiService.getNodes()
        setNodes(fetchedNodes)
      } catch (error) {
        console.error('Failed to fetch nodes:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchNodes()
    // Refresh data every 30 seconds
    const interval = setInterval(fetchNodes, 30000)
    return () => clearInterval(interval)
  }, [])

  const onlineNodes = nodes.filter(node => node.status === "online")
  const offlineNodes = nodes.filter(node => node.status === "offline")

  if (loading) {
    return (
      <div className="flex-1 space-y-4 p-4 pt-6">
        <div className="flex items-center justify-between space-y-2">
          <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="animate-pulse">
                <div className="h-4 bg-muted rounded w-3/4"></div>
              </CardHeader>
              <CardContent>
                <div className="h-8 bg-muted rounded w-1/2 animate-pulse"></div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
      </div>
      
      {/* Overview Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Nodes</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{nodes.length}</div>
            <p className="text-xs text-muted-foreground">
              Active nodes in the system
            </p>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Online</CardTitle>
            <Wifi className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">{onlineNodes.length}</div>
            <p className="text-xs text-muted-foreground">
              Currently connected
            </p>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Offline</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{offlineNodes.length}</div>
            <p className="text-xs text-muted-foreground">
              Disconnected nodes
            </p>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Health Status</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {nodes.length > 0 ? Math.round((onlineNodes.length / nodes.length) * 100) : 0}%
            </div>
            <p className="text-xs text-muted-foreground">
              Overall system health
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Node List */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {nodes.map((node) => (
          <Link key={node.id} to={`/nodes/${node.id}`}>
            <Card className="cursor-pointer hover:shadow-md transition-shadow">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-base font-medium">{node.name}</CardTitle>
                <Badge variant={node.status === "online" ? "default" : "destructive"}>
                  {node.status}
                </Badge>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>Host:</span>
                    <span>{node.host}</span>
                  </div>
                  {node.os && (
                    <div className="flex justify-between text-sm">
                      <span>OS:</span>
                      <span>{node.os}</span>
                    </div>
                  )}
                  {node.uptime && (
                    <div className="flex justify-between text-sm">
                      <span>Uptime:</span>
                      <span>{node.uptime}</span>
                    </div>
                  )}
                  <div className="text-xs text-muted-foreground mt-2">
                    Last seen: {node.lastSeen}
                  </div>
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}
