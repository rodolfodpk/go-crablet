# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY ../../../go.mod ../../../go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY ../../../ ./

# Set working directory for build
WORKDIR /app/internal/grpc-app/benchmark

# Build the benchmark application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o /app/grpc-benchmark .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/grpc-benchmark .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set performance-related environment variables
ENV GOMAXPROCS=4
ENV GOGC=100

# Run the benchmark
CMD ["./grpc-benchmark"] 