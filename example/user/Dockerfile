# syntax=docker/dockerfile:1
# Multi-stage Dockerfile for example/user/
# Stage 1: Builder
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/user .

# Stage 2: Minimal runtime
FROM scratch AS production

COPY --from=builder /app/user /user

EXPOSE 8080

ENTRYPOINT ["/user"]

# Stage 3: Final with shell for debugging
FROM alpine:3.19 AS final

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 1000 appuser

COPY --from=builder /app/user /usr/local/bin/user

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["user"]
