# Build stage
FROM golang:1.21-alpine AS builder

# Install ca-certificates and git
RUN apk add --no-cache ca-certificates git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o event-gateway \
    ./cmd/gateway

# Final stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S eventgateway && \
    adduser -u 1001 -S eventgateway -G eventgateway

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/event-gateway .

# Copy config file
COPY --from=builder /app/config.yaml .

# Change ownership
RUN chown -R eventgateway:eventgateway /app

# Switch to non-root user
USER eventgateway

# Expose port
EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/health || exit 1

# Run the application
CMD ["./event-gateway"]
