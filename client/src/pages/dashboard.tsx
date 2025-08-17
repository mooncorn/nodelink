import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Activity, Server, Wifi, AlertTriangle } from "lucide-react"
import { apiService, type Node, type StatusChangeEvent } from "@/lib/api"
import { Link } from "react-router-dom"

export default function DashboardPage() {
  const [nodes, setNodes] = useState<Node[]>([])
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

    // Set up SSE connection for real-time updates
    const eventSource = apiService.connectToAgentStatusEvents()
    
    if (eventSource) {
      eventSource.onopen = () => {
        console.log('SSE connection opened to /agents/events')
        setConnectionStatus('connected')
      }

      eventSource.onerror = () => {
        console.error('SSE connection error')
        setConnectionStatus('disconnected')
      }

      // Listen for all messages to debug
      eventSource.onmessage = (event: MessageEvent) => {
        console.log('Received SSE message:', event.data)
        try {
          const parsedData = JSON.parse(event.data)
          console.log('Parsed SSE data:', parsedData)
          
          // Handle agent status change events
          if (parsedData.event === 'agent_status_change') {
            console.log('Processing agent_status_change event')
            const statusChange: StatusChangeEvent = parsedData.data
            console.log('Status change data:', statusChange)
            if (statusChange.agent) {
              console.log('Updating node:', statusChange.agent)
              setNodes(prevNodes => {
                const nodeIndex = prevNodes.findIndex(node => node.agent_id === statusChange.agent.agent_id)
                if (nodeIndex >= 0) {
                  const updatedNodes = [...prevNodes]
                  updatedNodes[nodeIndex] = statusChange.agent
                  console.log('Updated existing node at index:', nodeIndex)
                  return updatedNodes
                } else {
                  // New node connected
                  console.log('Adding new node:', statusChange.agent.agent_id)
                  return [...prevNodes, statusChange.agent]
                }
              })
            }
          }
        } catch (error) {
          console.error('Error parsing SSE message:', error)
        }
      }

      // Cleanup on unmount
      return () => {
        eventSource.close()
      }
    }

    // Fallback: refresh data every 30 seconds
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
          <Link key={node.agent_id} to={`/nodes/${node.agent_id}`}>
            <Card className="cursor-pointer hover:shadow-md transition-shadow">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-base font-medium">
                  {node.metadata?.name || node.agent_id}
                </CardTitle>
                <div className="flex items-center space-x-2">
                  <Badge variant={node.status === "online" ? "default" : "destructive"}>
                    {node.status}
                  </Badge>
                  {connectionStatus === 'connected' && (
                    <Wifi className="h-4 w-4 text-green-500" />
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>Agent ID:</span>
                    <span className="text-xs font-mono">{node.agent_id}</span>
                  </div>
                  {node.metadata?.host && (
                    <div className="flex justify-between text-sm">
                      <span>Host:</span>
                      <span>{node.metadata.host}</span>
                    </div>
                  )}
                  {node.metadata?.os && (
                    <div className="flex justify-between text-sm">
                      <span>OS:</span>
                      <span>{node.metadata.os}</span>
                    </div>
                  )}
                  <div className="text-xs text-muted-foreground mt-2">
                    Last seen: {new Date(node.last_seen).toLocaleString()}
                  </div>
                  {node.connected_at && (
                    <div className="text-xs text-muted-foreground">
                      Connected: {new Date(node.connected_at).toLocaleString()}
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}
