# syntax=docker/dockerfile:1
ARG GO_VERSION=1.24.5
ARG PLATFORM=linux/arm64
FROM --platform=$PLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev

# Copy the Go modules files first and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Install swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy the rest of your application code
COPY . .

# Generate swagger docs after copying source code
RUN swag init --output swagger --parseDependency --parseInternal --parseVendor --parseDepth 1 --generatedTime=false

# Build the application with optimizations for arm64 architecture
RUN CGO_ENABLED=1 GOARCH=arm64 go build \
    -ldflags="-w -s" \
    -o /admin-api . && \
    ls -la /admin-api

FROM --platform=$PLATFORM alpine:latest AS final
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1001 appgroup && \
    adduser -D -s /bin/sh -u 1001 -G appgroup appuser

# Create necessary directories
RUN mkdir -p /app/swagger /app/logs /app/storage/uploads /app/tmp && \
    chown -R appuser:appgroup /app

# Copy the binary
COPY --from=build --chown=appuser:appgroup /admin-api /app/admin-api
RUN chmod +x /app/admin-api && \
    ls -la /app/admin-api

# Copy application files with proper ownership
COPY --from=build --chown=appuser:appgroup /src/swagger/ /app/swagger/
COPY --from=build --chown=appuser:appgroup /src/static/ /app/static/

# Switch to non-root user
USER appuser

# Expose port (CapRover will handle this)
EXPOSE 8030

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl --silent --fail http://localhost:8030/health || exit 1

# Set environment variables for production
ENV GIN_MODE=release
ENV ENV=release
ENV SERVER_ADDRESS=:8030    

# Run the binary
CMD ["./admin-api"]