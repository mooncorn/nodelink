import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Server, Settings, Activity, Terminal, Command } from "lucide-react"
import { Link } from "react-router-dom"
import { apiService, type Node } from "@/lib/api"

export default function NodesPage() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
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

  if (loading) {
    return (
      <div className="flex-1 space-y-4 p-4 pt-6">
        <div className="flex items-center justify-between space-y-2">
          <h2 className="text-3xl font-bold tracking-tight">Nodes</h2>
          <Button>
            <Server className="mr-2 h-4 w-4" />
            Add Node
          </Button>
        </div>
        
        <Card>
          <CardHeader>
            <CardTitle>Remote Nodes</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="h-12 bg-muted rounded animate-pulse"></div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">Nodes</h2>
        <Button>
          <Server className="mr-2 h-4 w-4" />
          Add Node
        </Button>
      </div>
      
      <Card>
        <CardHeader>
          <CardTitle>Remote Nodes</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Host</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>OS</TableHead>
                <TableHead>CPU</TableHead>
                <TableHead>Memory</TableHead>
                <TableHead>Disk</TableHead>
                <TableHead>Uptime</TableHead>
                <TableHead>Last Seen</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {nodes.map((node) => (
                <TableRow 
                  key={node.id}
                  className={selectedNode === node.id ? "bg-muted/50" : ""}
                  onClick={() => setSelectedNode(selectedNode === node.id ? null : node.id)}
                >
                  <TableCell className="font-medium">
                    <Link to={`/nodes/${node.id}`} className="hover:underline">
                      {node.name}
                    </Link>
                  </TableCell>
                  <TableCell>{node.host}</TableCell>
                  <TableCell>
                    <Badge variant={node.status === "online" ? "default" : "destructive"}>
                      {node.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{node.os || "Unknown"}</TableCell>
                  <TableCell>-</TableCell>
                  <TableCell>-</TableCell>
                  <TableCell>-</TableCell>
                  <TableCell>{node.uptime || "-"}</TableCell>
                  <TableCell>{node.lastSeen}</TableCell>
                  <TableCell>
                    <div className="flex space-x-2">
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/nodes/${node.id}/system`}>
                          <Activity className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/nodes/${node.id}/command`}>
                          <Command className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/nodes/${node.id}/terminal`}>
                          <Terminal className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button variant="outline" size="sm">
                        <Settings className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
