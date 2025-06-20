FROM golang:1.24-alpine AS builder

# Install git and other build dependencies
RUN apk add --no-cache git

# Install xk6
RUN go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with gRPC extension
RUN xk6 build --with github.com/grafana/xk6-grpc@latest --replace go.k6.io/k6=go.k6.io/k6@v0.47.0

FROM alpine:latest

# Install k6 binary
COPY --from=builder /go/k6 /usr/local/bin/k6

# Set entrypoint
ENTRYPOINT ["k6"]
