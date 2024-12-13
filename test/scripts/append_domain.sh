#!/usr/bin/env bash

# Check if a file was provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <yaml_file>"
    exit 1
fi

file="$1"
echo "Processing $file"

# Find the line number of the target domain
line_num=$(grep -n "sample-app-prod.nsx.services" "$file" | cut -d: -f1)

if [ -n "$line_num" ]; then
    # Get the indentation by looking at the target line
    indent=$(sed -n "${line_num}p" "$file" | sed -E 's/^( *).*/\1/')
    
    # Check if new domain already exists
    if ! grep -q "sample-app-prod2.nsx.services" "$file"; then
        # Insert new domain after the target line with same indentation
        sed -i "${line_num}a\\${indent}- sample-app-prod2.nsx.services" "$file"
        echo "Added new domain to $file"
    else
        echo "New domain already exists in $file"
    fi
else
    echo "Target domain not found in $file"
fi 