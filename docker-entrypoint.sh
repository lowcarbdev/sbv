#!/bin/sh
set -e

# Default UID and GID if not specified
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

# Create group if it doesn't exist
if ! getent group sbv >/dev/null 2>&1; then
    addgroup -g "${PGID}" sbv
fi

# Create user if it doesn't exist
if ! getent passwd sbv >/dev/null 2>&1; then
    adduser -D -u "${PUID}" -G sbv sbv
fi

# Ensure the user has the correct UID/GID
if [ "$(id -u sbv)" != "${PUID}" ] || [ "$(id -g sbv)" != "${PGID}" ]; then
    deluser sbv >/dev/null 2>&1 || true
    delgroup sbv >/dev/null 2>&1 || true
    addgroup -g "${PGID}" sbv
    adduser -D -u "${PUID}" -G sbv sbv
fi

# Ensure data directory exists and has correct permissions
mkdir -p "${DB_PATH_PREFIX:-/data}"
chown -R sbv:sbv "${DB_PATH_PREFIX:-/data}"

# Log the user we're running as
echo "Running as UID=${PUID} GID=${PGID}"

# Switch to the sbv user and execute the application
exec su-exec sbv "$@"
