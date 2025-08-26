# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy shared module from root
# Since we're building from root context, shared/ is at the root
COPY shared/ ./shared/

# Copy the downloader worker code
COPY workers/downloader/ ./workers/downloader/

# Set working directory to downloader
WORKDIR /build/workers/downloader

# Copy go mod files and download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o handler cmd/main.go

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

# Set working directory
WORKDIR /home/app

# Copy binary from builder
COPY --from=builder /build/workers/downloader/handler .

# Change ownership
RUN chown -R app:app /home/app

# Switch to non-root user
USER app

# Expose port for HTTP mode
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["./handler", "-health"] || exit 1

# Run the handler
CMD ["./handler"]