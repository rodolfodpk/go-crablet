# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Set working directory for build
WORKDIR /app/internal/grpc-app/server

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -gcflags="-l=4" \
    -o /app/grpc-server .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates
# Install Go to build grpcurl
RUN apk add --no-cache go
# Install grpcurl for healthchecks
RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
# Remove Go after installing grpcurl to keep image small
RUN apk del go
# Add Go bin to PATH for grpcurl
ENV PATH="/root/go/bin:$PATH"

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/grpc-server .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set performance-related environment variables
ENV GOMAXPROCS=4
ENV GOGC=100
ENV GODEBUG=netdns=go

# Expose port
EXPOSE 9090

# Run the application
CMD ["./grpc-server"] 