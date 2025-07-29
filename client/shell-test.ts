import {EventSource} from 'eventsource';

const serverURL = "http://localhost:8080";

const executeShellTask = async (agentId: string, cmd: string, timeout?: number) => {
  try {
    console.log(`Creating shell task for agent ${agentId}: ${cmd}`);
    
    // Create the shell task using the new API
    const response = await fetch(`${serverURL}/tasks/shell`, {
      method: "POST",
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        agent_id: agentId,
        cmd: cmd,
        timeout: timeout || 300 // default 5 minutes
      })
    });

    if (!response.ok) {
      const errorData = await response.json();
      console.error('Failed to create task:', errorData);
      return;
    }

    const data = await response.json();
    const taskId = data.task_id;
    console.log(`Task created with ID: ${taskId}`);
    console.log(`Task status: ${data.status}`);
    
    // Connect to event stream using task ID as ref
    console.log(`Connecting to event stream for task ${taskId}...`);
    const es = new EventSource(`${serverURL}/stream?ref=${taskId}`);

    es.onopen = () => {
      console.log(`[${taskId}] Event stream connection opened`);
    };

    es.onmessage = (event) => {
      console.log(`[${taskId}] Generic message:`, event.data);
    };

    es.addEventListener("response", (event) => {
      try {
        const responseData = JSON.parse(event.data);
        console.log(`[${taskId}] Task response:`, responseData);
        
        // Check if task is completed
        if (responseData.status === 'completed' || responseData.status === 'failed') {
          console.log(`[${taskId}] Task finished with status: ${responseData.status}`);
          es.close();
        }
      } catch (error) {
        console.log(`[${taskId}] Raw response:`, event.data);
      }
    });

    es.onerror = (error) => {
      console.error(`[${taskId}] Event stream error:`, error);
    };

    // Close connection after 30 seconds if still open
    setTimeout(() => {
      if (es.readyState === EventSource.OPEN) {
        console.log(`[${taskId}] Closing connection after timeout`);
        es.close();
      }
    }, 30000);

  } catch (error) {
    console.error('Error executing shell command:', error);
  }
};

// Example usage
const main = async () => {
  console.log('Starting shell task test...');
  
  // Test with a simple command that should complete quickly
  await executeShellTask("agent1", "echo 'Hello from shell task!'");
  
  // Wait a bit before starting next test
  setTimeout(async () => {
    // Test with a command that produces multiple outputs
    await executeShellTask("agent2", "for i in {1..5}; do echo \"Line $i\"; sleep 1; done", 60);
  }, 2000);
};

main().catch(console.error);
