import NodeLinkServer from "./src/server";

// Create and start the server
const server = new NodeLinkServer(8443);

// Log server statistics every 30 seconds
setInterval(() => {
  const stats = server.getStats();
  console.log("Server Stats:", stats);
}, 30000);

// Handle graceful shutdown
process.on("SIGINT", () => {
  console.log("\nShutting down server...");
  process.exit(0);
});

process.on("SIGTERM", () => {
  console.log("\nShutting down server...");
  process.exit(0);
});
