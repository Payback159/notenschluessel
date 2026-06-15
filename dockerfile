# Frontend build
FROM node:24-alpine AS frontend-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY tsconfig.json vite.config.ts ./
COPY index.html privacy.html ./
COPY src ./src
COPY public ./public
RUN npm run build

# Go static server build
FROM golang:1.26-alpine AS go-build
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o notenschluessel .

# Runtime image
FROM scratch
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-build /app/notenschluessel /app/notenschluessel
COPY --from=frontend-build /app/dist /app/dist
WORKDIR /app
EXPOSE 8080
ENV ENV=production STATIC_DIR=/app/dist
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/notenschluessel", "--health-check"]
ENTRYPOINT ["/app/notenschluessel"]
