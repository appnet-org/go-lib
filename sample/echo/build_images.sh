#!/bin/bash
set -ex

# # Ensure the script is being run from the correct directory
# SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
# cd "$SCRIPT_DIR"

go build -o server server.go
go build -o frontend frontend.go


# Build the frontend image
sudo docker build --tag echo-frontend:latest -f Dockerfile-frontend ../..

# Build the server image
sudo docker build --tag echo-server:latest -f Dockerfile-server ../..

# Tag the images
sudo docker tag echo-frontend  appnetorg/echo-frontend-grpc:latest
sudo docker tag echo-server  appnetorg/echo-server-grpc:latest

# Push the images to the registry
sudo docker push  appnetorg/echo-frontend-grpc:latest
sudo docker push  appnetorg/echo-server-grpc:latest

set +ex
