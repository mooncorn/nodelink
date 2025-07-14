#!/bin/bash

# Node Manager - Start Node Agent Script
NODE_ID=${1:-node1}

echo "Starting Node Agent: $NODE_ID"

# Start the node agent
cd node-agent
npx ts-node index.ts $NODE_ID 