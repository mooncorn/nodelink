import {EventSource} from 'eventsource';

const serverURL = "http://localhost:8080";

interface TaskInfo {
  taskId: string;
  agentId: string;
  eventSource?: EventSource;
}

const createShellTask = async (agentId: string, cmd: string, timeout?: number): Promise<TaskInfo | null> => {
  try {
    console.log(`Creating shell task for agent ${agentId}: ${cmd}`);
    
    const response = await fetch(`${serverURL}/tasks/shell`, {
      method: "POST",
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        agent_id: agentId,
        cmd: cmd,
        timeout: timeout || 300
      })
    });

    if (!response.ok) {
      const errorData = await response.json();
      console.error('Failed to create task:', errorData);
      return null;
    }

    const data = await response.json();
    console.log(`✅ Task created with ID: ${data.task_id}, Status: ${data.status}`);
    
    return {
      taskId: data.task_id,
      agentId: agentId
    };
  } catch (error) {
    console.error('Error creating shell task:', error);
    return null;
  }
};

const cancelTask = async (taskId: string): Promise<boolean> => {
  try {
    console.log(`🚫 Attempting to cancel task: ${taskId}`);
    
    const response = await fetch(`${serverURL}/tasks/${taskId}`, {
      method: "DELETE"
    });

    if (!response.ok) {
      const errorData = await response.json();
      console.error('Failed to cancel task:', errorData);
      return false;
    }

    const data = await response.json();
    console.log(`✅ Task cancelled: ${data.message}`);
    return true;
  } catch (error) {
    console.error('Error cancelling task:', error);
    return false;
  }
};

const monitorTask = (taskInfo: TaskInfo): Promise<void> => {
  return new Promise((resolve) => {
    console.log(`📡 Monitoring task ${taskInfo.taskId}...`);
    
    const es = new EventSource(`${serverURL}/stream?ref=${taskInfo.taskId}`);
    taskInfo.eventSource = es;

    es.onopen = () => {
      console.log(`[${taskInfo.taskId}] Event stream connection opened`);
    };

    es.onmessage = (event) => {
      console.log(`[${taskInfo.taskId}] Generic message:`, event.data);
    };

    es.addEventListener("response", (event) => {
      try {
        const responseData = JSON.parse(event.data);
        console.log(`[${taskInfo.taskId}] Task response:`, {
          status: responseData.status,
          isFinal: responseData.isFinal,
          cancelled: responseData.cancelled,
          stdout: responseData.shellExecute?.stdout?.replace(/\n/g, '\\n') || '',
          stderr: responseData.shellExecute?.stderr?.replace(/\n/g, '\\n') || ''
        });
        
        // Check if task is completed
        if (responseData.isFinal) {
          console.log(`[${taskInfo.taskId}] 🏁 Task finished with status: ${responseData.status}, cancelled: ${responseData.cancelled}`);
          es.close();
          resolve();
        }
      } catch (error) {
        console.log(`[${taskInfo.taskId}] Raw response:`, event.data);
      }
    });

    es.onerror = (error) => {
      console.error(`[${taskInfo.taskId}] ❌ Event stream error:`, error);
      es.close();
      resolve();
    };

    // Timeout after 60 seconds
    setTimeout(() => {
      if (es.readyState === EventSource.OPEN) {
        console.log(`[${taskInfo.taskId}] ⏰ Closing connection after timeout`);
        es.close();
        resolve();
      }
    }, 60000);
  });
};

const testCancellation = async () => {
  console.log('🧪 Starting task cancellation test...\n');
  
  // Test 1: Create a long-running task and cancel it quickly
  console.log('📋 Test 1: Quick cancellation of long-running task');
  const task1 = await createShellTask("agent1", "for i in {1..20}; do echo \"Long task line $i\"; sleep 2; done", 300);
  
  if (task1) {
    // Start monitoring in background
    const monitorPromise1 = monitorTask(task1);
    
    // Wait 3 seconds then cancel
    setTimeout(async () => {
      await cancelTask(task1.taskId);
    }, 3000);
    
    await monitorPromise1;
  }
  
  console.log('\n' + '='.repeat(60) + '\n');
  
  // Test 2: Create a task and cancel it immediately
  console.log('📋 Test 2: Immediate cancellation');
  const task2 = await createShellTask("agent2", "sleep 30 && echo 'This should not appear'", 60);
  
  if (task2) {
    // Start monitoring in background
    const monitorPromise2 = monitorTask(task2);
    
    // Cancel immediately
    setTimeout(async () => {
      await cancelTask(task2.taskId);
    }, 500);
    
    await monitorPromise2;
  }
  
  console.log('\n' + '='.repeat(60) + '\n');
  
  // Test 3: Try to cancel a task that completes quickly (race condition test)
  console.log('📋 Test 3: Race condition test (task might complete before cancellation)');
  const task3 = await createShellTask("agent1", "echo 'Quick task' && sleep 1", 30);
  
  if (task3) {
    // Start monitoring in background
    const monitorPromise3 = monitorTask(task3);
    
    // Try to cancel after 2 seconds (task might already be done)
    setTimeout(async () => {
      await cancelTask(task3.taskId);
    }, 2000);
    
    await monitorPromise3;
  }
  
  console.log('\n🎉 Task cancellation test completed!');
};

// Handle graceful shutdown
process.on('SIGINT', () => {
  console.log('\n🛑 Shutting down test script...');
  process.exit(0);
});

// Run the test
testCancellation().catch(console.error);
