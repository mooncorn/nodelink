#!/bin/bash

# Node Manager - Start Server Script
echo "Starting Node Manager Server..."

# Check if certificates exist
if [ ! -f "server/certs/localhost.pem" ] || [ ! -f "server/certs/localhost-key.pem" ]; then
    echo "SSL certificates not found. Please run the setup first:"
    echo "1. Install mkcert: brew install mkcert"
    echo "2. Install CA: mkcert -install"
    echo "3. Generate certificates: cd server/certs && mkcert localhost"
    exit 1
fi

# Start the server
cd server
echo "Starting HTTPS server on port 8443..."
npx ts-node index.ts 