import { useState } from "react"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { Send, Terminal, Clock, ArrowLeft } from "lucide-react"
import { apiService, type CommandExecution } from "@/lib/api"

export default function NodeCommandPage() {
  const { id } = useParams<{ id: string }>()
  const [command, setCommand] = useState("")
  const [isExecuting, setIsExecuting] = useState(false)
  const [executions, setExecutions] = useState<CommandExecution[]>([
    {
      id: "1",
      command: "ls -la",
      output: `total 32
drwxr-xr-x  8 ubuntu ubuntu 4096 Nov 20 10:30 .
drwxr-xr-x  3 root   root   4096 Nov 19 09:15 ..
-rw-------  1 ubuntu ubuntu  220 Nov 19 09:15 .bash_logout
-rw-------  1 ubuntu ubuntu 3771 Nov 19 09:15 .bashrc
drwx------  2 ubuntu ubuntu 4096 Nov 19 09:20 .cache
-rw-------  1 ubuntu ubuntu  807 Nov 19 09:15 .profile`,
      exitCode: 0,
      timestamp: new Date("2024-11-20T10:30:00"),
      duration: 120
    },
    {
      id: "2", 
      command: "ps aux | head -10",
      output: `USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root           1  0.0  0.1 168704 11776 ?        Ss   Nov19   0:02 /sbin/init
root           2  0.0  0.0      0     0 ?        S    Nov19   0:00 [kthreadd]
root           3  0.0  0.0      0     0 ?        I<   Nov19   0:00 [rcu_gp]
root           4  0.0  0.0      0     0 ?        I<   Nov19   0:00 [rcu_par_gp]
root           6  0.0  0.0      0     0 ?        I<   Nov19   0:00 [kworker/0:0H-kblockd]
root           9  0.0  0.0      0     0 ?        I<   Nov19   0:00 [mm_percpu_wq]
root          10  0.0  0.0      0     0 ?        S    Nov19   0:00 [ksoftirqd/0]
root          11  0.0  0.0      0     0 ?        I    Nov19   0:01 [rcu_preempt]`,
      exitCode: 0,
      timestamp: new Date("2024-11-20T10:28:00"),
      duration: 85
    }
  ])

  const executeCommand = async () => {
    if (!command.trim() || !id) return

    setIsExecuting(true)
    
    try {
      const result = await apiService.executeCommand(id, command.trim())
      setExecutions(prev => [result, ...prev])
      setCommand("")
    } catch (error) {
      console.error('Failed to execute command:', error)
      // Add error execution record
      const errorExecution: CommandExecution = {
        id: Date.now().toString(),
        command: command.trim(),
        output: `Error: Failed to execute command - ${error}`,
        exitCode: 1,
        timestamp: new Date(),
        duration: 0
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

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div className="flex items-center space-x-4">
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${id}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h2 className="text-3xl font-bold tracking-tight">Command Execution</h2>
        </div>
        <Badge variant="outline">Node: {id}</Badge>
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
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Command History</CardTitle>
        </CardHeader>
        <CardContent>
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
                      <Badge variant={execution.exitCode === 0 ? "default" : "destructive"}>
                        Exit: {execution.exitCode}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {execution.duration}ms
                      </span>
                    </div>
                  </div>
                  
                  <div className="text-xs text-muted-foreground">
                    {execution.timestamp.toLocaleString()}
                  </div>
                  
                  <ScrollArea className="h-40">
                    <pre className="text-sm bg-black text-green-400 p-3 rounded font-mono overflow-x-auto">
                      {execution.output}
                    </pre>
                  </ScrollArea>
                </div>
              ))}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  )
}
