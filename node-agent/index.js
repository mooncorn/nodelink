const WebSocket = require('ws');
const { spawn } = require('child_process');

const nodeId = process.argv[2] || 'node1'; // Pass node ID via command line, default to 'node1'
const token = nodeId === 'node1' ? 'token-for-node1' : 'token-for-node2';

console.log(`Starting node agent: ${nodeId}`);

const ws = new WebSocket('wss://localhost:8443', {
  rejectUnauthorized: false // Trust self-signed cert for PoC
});

ws.on('open', () => {
  console.log(`Node ${nodeId} connecting...`);
  ws.send(JSON.stringify({ type: 'register', id: nodeId, token }));
});

ws.on('message', (message) => {
  const data = JSON.parse(message);
  
  if (data.type === 'execute_command') {
    const { commandId, command } = data;
    console.log(`Executing: ${command}`);
    
    // Create child process to execute command
    const proc = spawn(command, { shell: true });

    // Send stdout data in real-time
    proc.stdout.on('data', (data) => {
      ws.send(JSON.stringify({ 
        type: 'command_output', 
        commandId, 
        output: data.toString() 
      }));
    });

    // Send stderr data in real-time
    proc.stderr.on('data', (data) => {
      ws.send(JSON.stringify({ 
        type: 'command_output', 
        commandId, 
        output: data.toString() 
      }));
    });

    // Handle process completion
    proc.on('close', (code) => {
      console.log(`Command "${command}" finished with exit code ${code}`);
      ws.send(JSON.stringify({ 
        type: 'command_finished', 
        commandId, 
        exitCode: code 
      }));
    });

    // Handle process errors
    proc.on('error', (err) => {
      console.error(`Error executing command: ${err.message}`);
      ws.send(JSON.stringify({ 
        type: 'command_output', 
        commandId, 
        output: `Error: ${err.message}\n` 
      }));
      ws.send(JSON.stringify({ 
        type: 'command_finished', 
        commandId, 
        exitCode: 1 
      }));
    });
  }
});

ws.on('error', (err) => {
  console.error(`Node ${nodeId} error:`, err);
});

ws.on('close', () => {
  console.log(`Node ${nodeId} disconnected`);
  process.exit(0);
});

// Handle graceful shutdown
process.on('SIGINT', () => {
  console.log(`\nNode ${nodeId} shutting down...`);
  ws.close();
  process.exit(0);
});

process.on('SIGTERM', () => {
  console.log(`\nNode ${nodeId} shutting down...`);
  ws.close();
  process.exit(0);
}); 