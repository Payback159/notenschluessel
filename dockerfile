# Build stage
FROM golang:1.25-alpine AS build-env

# Install security updates and ca-certificates
RUN apk update && apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY *.go ./
COPY pkg/ ./pkg/
COPY templates ./templates

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o notenschluessel .

# Create non-root user
RUN echo "notenschluessel:x:10001:10001:notenschluessel user:/app:/sbin/nologin" >> /etc/passwd_single && \
    echo "notenschluessel:x:10001:" >> /etc/group_single

# Final stage - minimal scratch image
FROM scratch

# Add CA certificates and timezone data
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /usr/share/zoneinfo /usr/share/zoneinfo

# Set up non-root user
COPY --from=build-env /etc/passwd_single /etc/passwd
COPY --from=build-env /etc/group_single /etc/group

# Create secure temp directory with proper permissions
COPY --from=build-env --chown=10001:10001 /tmp /tmp

WORKDIR /app

# Copy application files
COPY --from=build-env --chown=10001:10001 /app/notenschluessel /app/
COPY --from=build-env --chown=10001:10001 /app/templates/ /app/templates/

# Use non-root user
USER 10001:10001

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/notenschluessel", "--health-check"]

# Set security environment
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64

ENTRYPOINT ["/app/notenschluessel"]
