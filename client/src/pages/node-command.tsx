import { useState } from "react"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { Send, Terminal, Clock, ArrowLeft, AlertCircle } from "lucide-react"
import { apiService, type ExecuteCommandResponse } from "@/lib/api"

interface CommandExecution {
  id: string
  command: string
  response: ExecuteCommandResponse
  timestamp: Date
}

export default function NodeCommandPage() {
  const { nodeId } = useParams<{ nodeId: string }>()
  const [command, setCommand] = useState("")
  const [isExecuting, setIsExecuting] = useState(false)
  const [executions, setExecutions] = useState<CommandExecution[]>([])

  const executeCommand = async () => {
    if (!command.trim() || !nodeId) return

    setIsExecuting(true)
    
    try {
      const response = await apiService.executeCommand(nodeId, command.trim())
      
      const execution: CommandExecution = {
        id: response.request_id,
        command: command.trim(),
        response,
        timestamp: new Date()
      }
      
      setExecutions(prev => [execution, ...prev])
      setCommand("")
    } catch (error) {
      console.error('Failed to execute command:', error)
      
      // Add error execution record
      const errorResponse: ExecuteCommandResponse = {
        request_id: Date.now().toString(),
        exit_code: 1,
        stdout: "",
        stderr: `Error: Failed to execute command - ${error instanceof Error ? error.message : 'Unknown error'}`,
        error: error instanceof Error ? error.message : 'Unknown error',
        timeout: false
      }
      
      const errorExecution: CommandExecution = {
        id: errorResponse.request_id,
        command: command.trim(),
        response: errorResponse,
        timestamp: new Date()
      }
      
      setExecutions(prev => [errorExecution, ...prev])
      setCommand("")
    } finally {
      setIsExecuting(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      executeCommand()
    }
  }

  const getStatusBadge = (execution: CommandExecution) => {
    if (execution.response.timeout) {
      return <Badge variant="outline">Timeout</Badge>
    }
    if (execution.response.error) {
      return <Badge variant="destructive">Error</Badge>
    }
    if (execution.response.exit_code === 0) {
      return <Badge variant="default">Success</Badge>
    }
    return <Badge variant="destructive">Exit: {execution.response.exit_code}</Badge>
  }

  const getOutput = (execution: CommandExecution) => {
    const { stdout, stderr, error } = execution.response
    
    if (error) {
      return `Error: ${error}`
    }
    
    let output = ""
    if (stdout) output += stdout
    if (stderr) {
      if (output) output += "\n\n--- STDERR ---\n"
      output += stderr
    }
    
    return output || "No output"
  }

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div className="flex items-center space-x-4">
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${nodeId}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h2 className="text-3xl font-bold tracking-tight">Command Execution</h2>
        </div>
        <Badge variant="outline">Node: {nodeId}</Badge>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Terminal className="h-5 w-5" />
            <span>Execute Command</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex space-x-2">
            <Input
              placeholder="Enter command to execute..."
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              onKeyPress={handleKeyPress}
              disabled={isExecuting}
              className="flex-1"
            />
            <Button 
              onClick={executeCommand} 
              disabled={isExecuting || !command.trim()}
            >
              {isExecuting ? (
                <>
                  <Clock className="mr-2 h-4 w-4 animate-spin" />
                  Executing...
                </>
              ) : (
                <>
                  <Send className="mr-2 h-4 w-4" />
                  Execute
                </>
              )}
            </Button>
          </div>
          <p className="text-sm text-muted-foreground mt-2">
            Press Enter to execute, or click the Execute button
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            Command History
            <Badge variant="outline">{executions.length} commands</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {executions.length === 0 ? (
            <div className="text-center py-8">
              <Terminal className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No commands executed yet</p>
              <p className="text-sm text-muted-foreground">Enter a command above to get started</p>
            </div>
          ) : (
            <ScrollArea className="h-[600px]">
              <div className="space-y-4">
                {executions.map((execution) => (
                  <div key={execution.id} className="border rounded-lg p-4 space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-2">
                        <Terminal className="h-4 w-4 text-muted-foreground" />
                        <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                          {execution.command}
                        </code>
                      </div>
                      <div className="flex items-center space-x-2">
                        {getStatusBadge(execution)}
                        {execution.response.timeout && (
                          <AlertCircle className="h-4 w-4 text-orange-500" />
                        )}
                      </div>
                    </div>
                    
                    <div className="text-xs text-muted-foreground flex items-center justify-between">
                      <span>{execution.timestamp.toLocaleString()}</span>
                      <span>Request ID: {execution.response.request_id}</span>
                    </div>
                    
                    <ScrollArea className="h-40">
                      <pre className={`text-sm p-3 rounded font-mono overflow-x-auto ${
                        execution.response.error || execution.response.exit_code !== 0
                          ? 'bg-red-950 text-red-100'
                          : 'bg-black text-green-400'
                      }`}>
                        {getOutput(execution)}
                      </pre>
                    </ScrollArea>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
