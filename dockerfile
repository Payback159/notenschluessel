# build stage
FROM golang:1.24-alpine AS build-env

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY *.go ./
COPY templates ./templates

# Build the binary
RUN CGO_ENABLED=0 go build -o notenschluessel .

# Set up user for the final stage
RUN echo "notenschluessel:x:10001:10001:notenschluessel user:/app:/sbin/nologin" >> /etc/passwd_single && \
    echo "notenschluessel:x:10001:" >> /etc/group_single

# final stage
FROM scratch

EXPOSE 8080
WORKDIR /app

# Copy binary from build stage
COPY --from=build-env /app/notenschluessel /app/
COPY --from=build-env /app/templates/ /app/templates/
COPY --from=build-env /etc/passwd_single /etc/passwd
COPY --from=build-env /etc/group_single /etc/group

USER 10001

ENTRYPOINT ["/app/notenschluessel"]
