# Multi-stage Dockerfile for Go server and worker
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Pre-copy go.mod and go.sum for caching
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /src/backend
RUN go mod download

# Copy backend source code and migrations
WORKDIR /src
COPY backend/ ./backend/
COPY db/migrations/ ./db/migrations/

# Build server and worker binaries
WORKDIR /src/backend
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/worker ./cmd/worker

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server /app/server
COPY --from=builder /app/worker /app/worker
COPY --from=builder /src/db/migrations /app/db/migrations

EXPOSE 8080

CMD ["/app/server"]
