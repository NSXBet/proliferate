#!/usr/bin/env bash

# Find all staging and production yaml files
find . -type f \( -name "staging.yaml" -o -name "production.yaml" \) | while read -r file; do
    echo "Processing $file"
    
    # Create a temporary file
    temp_file="${file}.tmp"
    
    # Read the file and normalize indentation (2 spaces)
    yq -P '.' "$file" > "$temp_file"
    
    # Check if there are actual changes
    if ! cmp -s "$file" "$temp_file"; then
        # Replace original with normalized version
        mv "$temp_file" "$file"
        echo "Normalized indentation in $file"
    else
        rm "$temp_file"
        echo "No indentation changes needed in $file"
    fi
done 