FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (required for go-sqlite3)
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled (required for SQLite)
RUN CGO_ENABLED=1 GOOS=linux go build -o biblio-audiobook-builder-tts .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (including libc for CGO binary)
RUN apk add --no-cache ca-certificates tzdata ffmpeg libc6-compat

# Copy binary from builder
COPY --from=builder /app/biblio-audiobook-builder-tts .

# Copy static assets
COPY --from=builder /app/internal/server/templates ./internal/server/templates
COPY --from=builder /app/internal/server/assets ./internal/server/assets

# Create directories for database and temp files
RUN mkdir -p /db /temp

EXPOSE 8080

ENV ABB_TTS_HOST=0.0.0.0
ENV ABB_TTS_PORT=8080
ENV ABB_TTS_DATABASE_PATH=/db/abb_tts.db
ENV ABB_TTS_OUTPUT_DIR=/temp
ENV ABB_TTS_TEMP_DIR=/temp

CMD ["./biblio-audiobook-builder-tts", "server", "--headless"]
