import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Server, Settings, Activity, Terminal, Command, Wifi, WifiOff } from "lucide-react"
import { Link } from "react-router-dom"
import { apiService, type Node, type StatusChangeEvent } from "@/lib/api"

export default function NodesPage() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'connecting'>('connecting')

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

    // Set up SSE connection for real-time agent status updates
    const eventSource = apiService.connectToAgentStatusEvents()
    
    if (eventSource) {
      eventSource.onopen = () => {
        setConnectionStatus('connected')
        console.log('Connected to agent status events')
      }

      eventSource.onerror = () => {
        setConnectionStatus('disconnected')
        console.error('Error connecting to agent status events')
      }

      eventSource.addEventListener('agent_status_change', (event) => {
        try {
          const statusChange: StatusChangeEvent = JSON.parse(event.data)
          setNodes(prevNodes => 
            prevNodes.map(node => 
              node.agent_id === statusChange.agent_id 
                ? { ...node, ...statusChange.agent }
                : node
            )
          )
        } catch (error) {
          console.error('Error parsing status change event:', error)
        }
      })

      // Cleanup on unmount
      return () => {
        eventSource.close()
      }
    }

    // Fallback: refresh data every 30 seconds if SSE is not available
    const interval = setInterval(fetchNodes, 30000)
    return () => clearInterval(interval)
  }, [])

  const formatLastSeen = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / (1000 * 60))
    
    if (diffMins < 1) return "just now"
    if (diffMins < 60) return `${diffMins} minutes ago`
    
    const diffHours = Math.floor(diffMins / 60)
    if (diffHours < 24) return `${diffHours} hours ago`
    
    const diffDays = Math.floor(diffHours / 24)
    return `${diffDays} days ago`
  }

  const getNodeDisplayName = (node: Node) => {
    return node.metadata?.name || node.agent_id
  }

  const getNodeHost = (node: Node) => {
    return node.metadata?.host || "Unknown"
  }

  const getNodeOS = (node: Node) => {
    return node.metadata?.os || "Unknown"
  }

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
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Nodes</h2>
          <div className="flex items-center space-x-2 mt-2">
            <div className="flex items-center space-x-1">
              {connectionStatus === 'connected' ? (
                <Wifi className="h-4 w-4 text-green-500" />
              ) : (
                <WifiOff className="h-4 w-4 text-red-500" />
              )}
              <span className="text-sm text-muted-foreground">
                {connectionStatus === 'connected' ? 'Live updates enabled' : 'Live updates disconnected'}
              </span>
            </div>
          </div>
        </div>
        <Button>
          <Server className="mr-2 h-4 w-4" />
          Add Node
        </Button>
      </div>
      
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            Remote Nodes
            <Badge variant="outline">
              {nodes.length} total, {nodes.filter(n => n.status === 'online').length} online
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Host</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>OS</TableHead>
                <TableHead>Last Seen</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {nodes.map((node) => (
                <TableRow 
                  key={node.agent_id}
                  className={selectedNode === node.agent_id ? "bg-muted/50" : ""}
                  onClick={() => setSelectedNode(selectedNode === node.agent_id ? null : node.agent_id)}
                >
                  <TableCell className="font-medium">
                    <Link to={`/nodes/${node.agent_id}`} className="hover:underline">
                      {getNodeDisplayName(node)}
                    </Link>
                  </TableCell>
                  <TableCell>{getNodeHost(node)}</TableCell>
                  <TableCell>
                    <Badge variant={node.status === "online" ? "default" : "destructive"}>
                      {node.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{getNodeOS(node)}</TableCell>
                  <TableCell>{formatLastSeen(node.last_seen)}</TableCell>
                  <TableCell>
                    <div className="flex space-x-2">
                      <Button 
                        variant="outline" 
                        size="sm" 
                        asChild
                        disabled={node.status === 'offline'}
                      >
                        <Link to={`/nodes/${node.agent_id}?tab=system`}>
                          <Activity className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button 
                        variant="outline" 
                        size="sm" 
                        asChild
                        disabled={node.status === 'offline'}
                      >
                        <Link to={`/nodes/${node.agent_id}/command`}>
                          <Command className="h-4 w-4" />
                        </Link>
                      </Button>
                      <Button 
                        variant="outline" 
                        size="sm" 
                        asChild
                        disabled={node.status === 'offline'}
                      >
                        <Link to={`/nodes/${node.agent_id}/terminal`}>
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
