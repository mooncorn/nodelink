const express = require('express');
const https = require('https');
const fs = require('fs');
const WebSocket = require('ws');

const app = express();
const server = https.createServer({
  cert: fs.readFileSync('./certs/localhost.pem'),
  key: fs.readFileSync('./certs/localhost-key.pem')
}, app);

const wss = new WebSocket.Server({ server });

// Hardcoded node credentials (for simplicity)
const validNodes = {
  "node1": "token-for-node1",
  "node2": "token-for-node2"
};

// Store connected nodes and frontend clients
const nodes = new Map(); // nodeId -> ws connection
const frontends = new Set(); // frontend ws connections
const commandMap = new Map(); // commandId -> { frontendWs, requestId }

wss.on('connection', (ws) => {
  ws.on('message', (message) => {
    const data = JSON.parse(message);

    // Handle node registration
    if (data.type === 'register') {
      const { id, token } = data;
      if (validNodes[id] === token) {
        nodes.set(id, ws);
        console.log(`Node ${id} registered`);
        broadcastNodeList();
      } else {
        ws.close();
      }
    }

    // Handle frontend command execution
    if (data.type === 'execute_command') {
      const { requestId, nodeId, command } = data;
      const nodeWs = nodes.get(nodeId);
      if (nodeWs) {
        const commandId = `cmd-${Date.now()}`;
        commandMap.set(commandId, { frontendWs: ws, requestId });
        nodeWs.send(JSON.stringify({ type: 'execute_command', commandId, command }));
      } else {
        ws.send(JSON.stringify({ type: 'command_output', requestId, output: `Node ${nodeId} not connected` }));
      }
    }

    // Handle node command output
    if (data.type === 'command_output') {
      const { commandId, output } = data;
      const cmdInfo = commandMap.get(commandId);
      if (cmdInfo) {
        cmdInfo.frontendWs.send(JSON.stringify({ type: 'command_output', requestId: cmdInfo.requestId, output }));
      }
    }

    // Handle node command completion
    if (data.type === 'command_finished') {
      const { commandId, exitCode } = data;
      const cmdInfo = commandMap.get(commandId);
      if (cmdInfo) {
        cmdInfo.frontendWs.send(JSON.stringify({ type: 'command_finished', requestId: cmdInfo.requestId, exitCode }));
        commandMap.delete(commandId);
      }
    }
  });

  ws.on('close', () => {
    for (let [id, nodeWs] of nodes) {
      if (nodeWs === ws) {
        nodes.delete(id);
        console.log(`Node ${id} disconnected`);
        broadcastNodeList();
        break;
      }
    }
    frontends.delete(ws);
  });

  // Add frontend connection
  frontends.add(ws);
  broadcastNodeList();
});

// Send updated node list to all frontends
function broadcastNodeList() {
  const nodeList = Array.from(nodes.keys()).map(id => ({ id, status: 'online' }));
  const message = JSON.stringify({ type: 'node_list', nodes: nodeList });
  frontends.forEach(ws => ws.send(message));
}

// Serve frontend
app.use(express.static('../frontend'));

server.listen(8443, () => {
  console.log('Server running on https://localhost:8443');
}); 