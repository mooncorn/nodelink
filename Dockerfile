# Railway Dockerfile for nodelink server (root directory)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files from root and subdirectories
COPY go.mod* go.sum* ./
COPY proto/go.mod proto/go.sum ./proto/
COPY server/go.mod server/go.sum ./server/

# Download dependencies
RUN go mod download || true
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
