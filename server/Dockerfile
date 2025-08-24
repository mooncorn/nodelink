# Railway Dockerfile for nodelink server
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for fetching dependencies)
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

# Copy go mod files from root and subdirectories
# Railway builds from repo root even when using server/Dockerfile
COPY go.mod* go.sum* ./
COPY proto/go.mod ./proto/
COPY proto/go.sum* ./proto/
COPY server/go.mod server/go.sum ./server/

# Download dependencies with better error handling
RUN go mod download || echo "Root mod download failed, continuing..."
RUN cd proto && go mod download
RUN cd server && go mod download

# Copy source code
COPY proto/ ./proto/
COPY server/ ./server/

# Build the server binary
WORKDIR /app/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server/main .

# Expose ports
EXPOSE 8080 9090

CMD ["./main"]
