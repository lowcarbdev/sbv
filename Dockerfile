# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Build frontend
RUN npm run build

# Stage 2: Build backend
FROM golang:bookworm AS backend-builder

WORKDIR /app

# Install build dependencies for libheif and SQLite FTS5
# Using bookworm-backports to get a newer version of libheif
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" >> /etc/apt/sources.list && \
    apt-get update && apt-get install -y \
    build-essential \
    && apt-get install -y -t bookworm-backports libheif-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY *.go ./
COPY internal/*.go internal/

# Build with FTS5 support
RUN go build -tags "fts5 heic" -o sbv .

# Stage 3: Final runtime image
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
# Using bookworm-backports to get matching runtime library
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" >> /etc/apt/sources.list && \
    apt-get update && apt-get install -y \
    ca-certificates \
    wget \
    ffmpeg \
    && apt-get install -y -t bookworm-backports libheif1 \
    && rm -rf /var/lib/apt/lists/*

# Copy backend binary
COPY --from=backend-builder /app/sbv .

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create data directory for database
RUN mkdir -p /data

# Set environment variables
ENV PORT=8081
ENV DB_PATH_PREFIX=/data

# Expose port
EXPOSE 8081

# Run the application
CMD ["./sbv"]
