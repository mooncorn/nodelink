import { spawn } from "child_process";
import { io } from "socket.io-client";

const nodeId = process.argv[2] || "node1"; // Pass node ID via command line, default to 'node1'
const token = nodeId === "node1" ? "token-for-node1" : "token-for-node2";

console.log(`Starting node agent: ${nodeId}`);

const socket = io("https://localhost:8443", { rejectUnauthorized: false });

socket.on("connect", () => {
  console.log(`Node ${nodeId} connected to server`);
  socket.emit("register", { id: nodeId, token });
});

socket.on("execute_command", (data) => {
  const { requestId, nodeId: targetNodeId, command } = data;

  // Only execute if this command is for this node
  if (targetNodeId !== nodeId) {
    return;
  }

  console.log(`Executing: ${command}`);

  // Create child process to execute command
  const proc = spawn(command, { shell: true });

  // Send stdout data in real-time
  proc.stdout.on("data", (data) => {
    socket.emit("command_output", {
      requestId,
      output: data.toString(),
    });
  });

  // Send stderr data in real-time
  proc.stderr.on("data", (data) => {
    socket.emit("command_output", {
      requestId,
      output: data.toString(),
    });
  });

  // Handle process completion
  proc.on("close", (code) => {
    console.log(`Command "${command}" finished with exit code ${code}`);
    socket.emit("command_finished", {
      requestId,
      exitCode: code,
    });
  });

  // Handle process errors
  proc.on("error", (err) => {
    console.error(`Error executing command: ${err.message}`);
    socket.emit("command_output", {
      requestId,
      output: `Error: ${err.message}\n`,
    });
    socket.emit("command_finished", {
      requestId,
      exitCode: 1,
    });
  });
});

socket.on("connect_error", (err) => {
  console.error(`Node ${nodeId} connection error:`, err);
});

socket.on("disconnect", (reason) => {
  console.log(`Node ${nodeId} disconnected: ${reason}`);
  if (reason === "io server disconnect") {
    // the disconnection was initiated by the server, you need to reconnect manually
    socket.connect();
  }
});

// Handle graceful shutdown
process.on("SIGINT", () => {
  console.log(`\nNode ${nodeId} shutting down...`);
  socket.disconnect();
  process.exit(0);
});

process.on("SIGTERM", () => {
  console.log(`\nNode ${nodeId} shutting down...`);
  socket.disconnect();
  process.exit(0);
});
