# Philotes Multi-Stage Dockerfile
# Builds philotes-api, philotes-worker, and philotes-cli binaries
#
# Build specific targets:
#   docker build --target api -t philotes-api .
#   docker build --target worker -t philotes-worker .
#   docker build --target cli -t philotes-cli .

# =============================================================================
# Build Stage
# =============================================================================
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build all binaries with static linking
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/philotes-api ./cmd/philotes-api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/philotes-worker ./cmd/philotes-worker

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/philotes ./cmd/philotes-cli

# =============================================================================
# API Runtime Image
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS api

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /app/bin/philotes-api /philotes-api

# Run as non-root user
USER nonroot:nonroot

# Expose API port
EXPOSE 8080

# Note: HEALTHCHECK not used with distroless (no shell).
# Use Kubernetes liveness/readiness probes instead.

ENTRYPOINT ["/philotes-api"]

# =============================================================================
# Worker Runtime Image
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS worker

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /app/bin/philotes-worker /philotes-worker

# Run as non-root user
USER nonroot:nonroot

ENTRYPOINT ["/philotes-worker"]

# =============================================================================
# CLI Runtime Image
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS cli

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /app/bin/philotes /philotes

# Run as non-root user
USER nonroot:nonroot

ENTRYPOINT ["/philotes"]
