import express from "express";
import https from "https";
import fs from "fs";
import { Server, Socket } from "socket.io";

const app = express();
const server = https.createServer(
  {
    cert: fs.readFileSync("./certs/localhost.pem"),
    key: fs.readFileSync("./certs/localhost-key.pem"),
  },
  app
);

const io = new Server(server);

// Hardcoded node credentials (for simplicity)
const validNodes = {
  node1: "token-for-node1",
  node2: "token-for-node2",
};

// Store connected nodes and frontend clients
const nodes = new Map<string, Socket>(); // nodeId -> socket connection
const frontends = new Set<Socket>(); // frontend socket connections
const commandMap = new Map<string, { frontendSocket: Socket; requestId: string }>(); // commandId -> { frontendSocket, requestId }

io.on("connection", (client) => {
  console.log("Client connected");

  // Handle node registration
  client.on("register", (data) => {
    const { id, token } = data;
    if (validNodes[id as keyof typeof validNodes] === token) {
      nodes.set(id, client);
      console.log(`Node ${id} registered`);
      broadcastNodeList();
    } else {
      console.log(`Invalid registration attempt for node ${id}`);
      client.disconnect();
    }
  });

  // Handle frontend command execution
  client.on("execute_command", (data) => {
    const { requestId, nodeId, command } = data;
    const nodeSocket = nodes.get(nodeId);
    if (nodeSocket) {
      const commandId = `cmd-${Date.now()}`;
      commandMap.set(commandId, { frontendSocket: client, requestId });
      nodeSocket.emit("execute_command", { requestId, nodeId, command });
    } else {
      client.emit("command_output", {
        requestId,
        output: `Node ${nodeId} not connected`,
      });
    }
  });

  // Handle node command output
  client.on("command_output", (data) => {
    const { requestId, output } = data;
    // Find the frontend socket that initiated this command
    for (const [commandId, cmdInfo] of commandMap.entries()) {
      if (cmdInfo.requestId === requestId) {
        cmdInfo.frontendSocket.emit("command_output", {
          requestId,
          output,
        });
        break;
      }
    }
  });

  // Handle node command completion
  client.on("command_finished", (data) => {
    const { requestId, exitCode } = data;
    // Find and remove the command mapping
    for (const [commandId, cmdInfo] of commandMap.entries()) {
      if (cmdInfo.requestId === requestId) {
        cmdInfo.frontendSocket.emit("command_finished", {
          requestId,
          exitCode,
        });
        commandMap.delete(commandId);
        break;
      }
    }
  });

  client.on("disconnect", () => {
    console.log("Client disconnected");
    
    // Check if this was a node connection
    for (let [id, nodeSocket] of nodes) {
      if (nodeSocket === client) {
        nodes.delete(id);
        console.log(`Node ${id} disconnected`);
        broadcastNodeList();
        break;
      }
    }
    
    // Remove from frontends if it was a frontend connection
    frontends.delete(client);
  });

  // Add frontend connection (if not a node, assume it's a frontend)
  frontends.add(client);
  broadcastNodeList();
});

// Send updated node list to all frontends
function broadcastNodeList() {
  const nodeList = Array.from(nodes.keys()).map((id) => ({
    id,
    status: "online",
  }));
  frontends.forEach((client) => client.emit("node_list", { nodes: nodeList }));
}

// Serve frontend
app.use(express.static("../frontend"));

server.listen(8443, () => {
  console.log("Server running on https://localhost:8443");
});
