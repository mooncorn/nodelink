# Nodelink - Claude Project Configuration

## Project Structure
This repository contains three interconnected projects:
- **agent/**: Go-based task execution agent
- **client/**: React/TypeScript web dashboard
- **server/**: Go-based central orchestrator

## Technology Stack
- **Backend**: Go, gRPC, Protocol Buffers
- **Frontend**: React, TypeScript, Vite
- **Communication**: HTTP APIs, Server-Sent Events
- **Build Tools**: Go modules, npm, Protocol Buffer compiler

## Development Workflow
1. Make changes to protocol buffers in `proto/agent.proto`
2. Run `./generate.sh` to regenerate protobuf code
3. Update server implementation in `server/`
4. Update agent implementation in `agent/`
5. Update client dashboard in `client/`

## Key Files
- `proto/agent.proto`: Source of truth for all message types
- `generate.sh`: Protobuf code generation script
