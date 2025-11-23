# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Build frontend with production API URL (relative path)
# This ensures the frontend uses relative paths instead of localhost:8081
ENV VITE_API_URL=/api
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

# Install build dependencies for libheif and SQLite FTS5
RUN apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    libheif-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY *.go ./
COPY internal/*.go internal/

# Build with FTS5 support
# Use CGO for SQLite and libheif
RUN CGO_ENABLED=1 go build -tags "fts5" -o sbv .

# Stage 3: Final runtime image
FROM alpine:3

WORKDIR /app

# Install runtime dependencies including su-exec for user switching
RUN apk add --no-cache \
    ca-certificates \
    wget \
    ffmpeg \
    libheif \
    su-exec

# Copy backend binary
COPY --from=backend-builder /app/sbv .

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Copy entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Create data directory for database
RUN mkdir -p /data

# Accept version as build argument and generate version.json
# This is done late in the build to maximize cache layer reuse
ARG VERSION=dev
RUN echo "{\"version\":\"${VERSION}\"}" > /app/version.json

# Set environment variables
ENV PORT=8081 \
    DB_PATH_PREFIX=/data \
    PUID=1000 \
    PGID=1000

# Expose port
EXPOSE 8081

# Use entrypoint to handle user switching
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Run the application
CMD ["./sbv"]
