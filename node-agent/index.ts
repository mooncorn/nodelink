import { NodeAgent } from "./src/node-agent";

const nodeId = process.argv[2] || "node1";
const token = nodeId === "node1" ? "token-for-node1" : "token-for-node2";

const agent = new NodeAgent(nodeId, token);

// Start the agent
agent.start();

// Log agent statistics every 30 seconds
setInterval(() => {
  const stats = agent.getStats();
  console.log("Agent Stats:", stats);
}, 30000);

// Handle graceful shutdown
process.on("SIGINT", () => {
  console.log(`\nNode ${nodeId} shutting down...`);
  agent.stop();
  process.exit(0);
});

process.on("SIGTERM", () => {
  console.log(`\nNode ${nodeId} shutting down...`);
  agent.stop();
  process.exit(0);
});
