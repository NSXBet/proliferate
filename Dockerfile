# Use Go base image
FROM golang:1.23-alpine

# Create working directory
WORKDIR /workspace

# Install necessary tools
RUN apk add --no-cache git bash yq

# Copy local source code
COPY . .

# Build and install proliferate
RUN cd cmd/pro && \
    go build -o /usr/local/bin/pro