# Use Python base image instead
FROM go

# Create working directory
WORKDIR /workspace

RUN apk add --no-cache git bash yq