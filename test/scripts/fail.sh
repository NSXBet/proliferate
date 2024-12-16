#!/usr/bin/env bash

# Check if FORCE_EXIT is set
if [ -n "$FORCE_EXIT" ]; then
    echo "Force exit code: $FORCE_EXIT"
    exit "$FORCE_EXIT"
fi

exit 1