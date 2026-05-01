# Multi-stage build for optimized image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN make build-server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/build/translator-server /app/

# Copy certificates directory
RUN mkdir -p /app/certs
VOLUME /app/certs

# Create config directory
RUN mkdir -p /app/config
VOLUME /app/config

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to app user
USER appuser

# Expose ports
EXPOSE 8443/tcp
EXPOSE 8443/udp

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider https://localhost:8443/health || exit 1

# Run the server
ENTRYPOINT ["/app/translator-server"]
CMD ["-config", "/app/config/config.json"]
