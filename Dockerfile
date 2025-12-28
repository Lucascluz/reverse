# --- Build stage ---
FROM golang:1.25.1-alpine AS builder

WORKDIR /build

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
# Copy only the minimal files required to build the binary to keep the image small
# - dependency files
# - cmd (main package)
# - internal (local packages)
COPY ./cmd ./cmd
COPY ./internal ./internal

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o /build/reverxy ./cmd/main.go

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -S appuser && adduser -S appuser -G appuser

# Copy the binary from the build stage
COPY --from=builder /build/reverxy /app/reverxy


# Create logs directory
RUN mkdir -p /app/logs && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 8085

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8085/healthz || exit 1

# Run the application
ENTRYPOINT ["/app/reverxy"]