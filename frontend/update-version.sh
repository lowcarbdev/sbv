#!/bin/bash
# CI script to update the frontend version
# Usage: ./update-version.sh <version>
# Example: ./update-version.sh 1.2.3

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 1.2.3"
  exit 1
fi

VERSION=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_FILE="$SCRIPT_DIR/version.json"

echo "Updating version to: $VERSION"
echo "{\"version\": \"$VERSION\"}" | jq '.' > "$VERSION_FILE"
echo "Version updated successfully in $VERSION_FILE"
