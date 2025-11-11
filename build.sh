#!/bin/bash
# Build script for SMS/MMS Backup Viewer
# Builds with FTS5 (Full-Text Search) support enabled
# Optional: Build with HEIC support by setting HEIC=1 or passing 'heic' as an argument
#   Usage: ./build.sh heic
#   Or:    HEIC=1 ./build.sh

set -e

# Check if HEIC support should be enabled
BUILD_TAGS="fts5"
if [ "$1" = "heic" ] || [ "$HEIC" = "1" ]; then
    BUILD_TAGS="fts5 heic"
    echo "Building SMS/MMS Backup Viewer with FTS5 and HEIC support..."
    echo "Note: This requires libheif library to be installed"
else
    echo "Building SMS/MMS Backup Viewer with FTS5 support..."
    echo "Note: HEIC images will use placeholders. Build with HEIC=1 ./build.sh or ./build.sh heic to enable HEIC conversion"
fi

go build -tags "$BUILD_TAGS" -o sbv .

echo "Build complete! Binary: ./sbv"
