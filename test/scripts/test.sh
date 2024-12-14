#!/usr/bin/env bash

set -e

VERSION_FILE="VERSION"

# Function to increment version
increment_version() {
    local version=$1
    echo $((version + 1))
}

# Check if VERSION file exists
if [ -f "$VERSION_FILE" ]; then
    # Read and increment version
    current_version=$(cat "$VERSION_FILE")
    new_version=$(increment_version "$current_version")
    echo "$new_version" > "$VERSION_FILE"
    echo "Incremented version from $current_version to $new_version"
else
    # Create VERSION file with initial version
    echo "1" > "$VERSION_FILE"
    echo "Created VERSION file with initial version 1"
fi